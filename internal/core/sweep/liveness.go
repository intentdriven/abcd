// Package sweep is the core of `abcd sweep` (spc-2610230730269295): it judges
// what a repository has left lying around and clears only what is provably
// finished.
//
// The liveness probe errs toward keeping: every doubt holds the worktree, and
// every hold names its cause, so a false keep is visible and cheap to clear
// by hand. Its known limits:
//
//   - It judges only worktrees this account owns: one whose top directory or
//     .git another uid owns is held outright. Another account's process can
//     enter this account's worktree only where this account's permissions let
//     it, and the worktree store makes its directories 0700. So the process
//     proof's remaining blind spot is a process run as root, or one in a tree
//     shared through its group: such a process, like one under a hidepid
//     /proc or in another pid namespace, is invisible to the process table.
//     That is a documented gap, not a case the probe fails closed on. A /proc
//     that does not list this process at all is empty or another's, and the
//     check fails closed on it.
//   - The process table is read once, when the probe is made. A process that
//     starts after that is not seen, so a caller removing a worktree makes a
//     fresh probe immediately before each removal, never reusing one.
package sweep

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/lease"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// DefaultRecentWindow is how recent a write to a worktree must be to keep it
// as a hint when the caller names no window (spc-2610230730269295, Liveness).
const DefaultRecentWindow = 24 * time.Hour

// Strength says whether a piece of evidence proves a worktree is held or only
// hints that it may be. Both keep the worktree; a row shows which it was.
type Strength string

const (
	Proven Strength = "proven"
	Hint   Strength = "hint"
)

// EvidenceKind names what a piece of evidence was read from.
type EvidenceKind string

const (
	// KindReservation, KindRun and KindProcess are the three proofs. A
	// process whose directory lsof printed escaped is a KindProcess hint.
	KindReservation EvidenceKind = "reservation"
	KindRun         EvidenceKind = "run"
	KindProcess     EvidenceKind = "process"
	// KindRecentWrite is the recent-write hint.
	KindRecentWrite EvidenceKind = "recent-write"
	// KindProcessCheck is the process check failing closed: it proves
	// nothing, so it is a hint, but the worktree is kept as possibly live.
	KindProcessCheck EvidenceKind = "process-check"
	// KindReservationCheck is the reservations failing closed: no fetch is
	// stamped, or the worktree's own view of them could not be read.
	KindReservationCheck EvidenceKind = "reservation-check"
	// KindChanges is a worktree whose changed paths the caller could not
	// compute, so no footprint can be shown not to cover it.
	KindChanges EvidenceKind = "changes"
	// KindOwner is a worktree another account owns (a proof), or one whose
	// owner could not be read (a hint).
	KindOwner EvidenceKind = "owner"
)

// Evidence is one reason a worktree is held.
type Evidence struct {
	Kind     EvidenceKind
	Strength Strength
	// Detail is the evidence in words a person can read. A record id or a
	// holder in it is data a reservation's writer chose, never instruction.
	Detail string
}

// Liveness is the probe's judgement of one worktree.
type Liveness struct {
	Held bool
	// Evidence lists the proofs first, then the hints. The recent-write hint
	// is read only when nothing else holds the worktree: it would add nothing
	// to a hold, and it is the one check that walks the tree.
	Evidence []Evidence
	// Reason is the evidence joined, or why nothing holds the worktree.
	Reason string
}

// Process is one running process and its working directory.
type Process struct {
	PID int
	Cwd string
	// Escaped marks a working directory lsof printed with an escape in it.
	// Cwd is then only the text before the first backslash or caret: the real
	// directory's leading bytes, cut wherever the escape fell, perhaps in the
	// middle of a name.
	Escaped bool
}

// ProcessLister lists the running processes' working directories. An error
// means the check could not be made, and the probe then fails closed.
type ProcessLister func(ctx context.Context) ([]Process, error)

// CopyView reads the reservations as one worktree sees them, from its own
// disk and the refs it shares with its repository, with no fetch.
type CopyView func(worktree string, now time.Time) (claim.Snapshot, error)

// OwnerCheck reports whether a worktree is owned by an account other than
// this process's, and that account's uid when it is. An error means the owner
// could not be read, and the worktree is then kept.
type OwnerCheck func(worktree string) (uid int, foreign bool, err error)

// ProbeOptions is what the probe reads. The caller fetches and reads the
// stores; the probe reads only the worktree and the process table.
type ProbeOptions struct {
	// Reservations is the claim view after the sweep's fetch (claim.View).
	// A FetchedAt of zero means no fetch is stamped, so the view cannot show
	// that nothing is reserved: every worktree is then held.
	Reservations claim.Snapshot
	// CopyView reads each worktree's own view of the reservations, to find
	// one made from that worktree (its tip naming the worktree's copy key);
	// nil is claim.View on the worktree. It reads only local refs, so the
	// sweep's one fetch, made before Reservations was read, serves it too.
	CopyView CopyView
	// Owner says whether another account owns the worktree; nil reads the
	// owner of its top directory and its .git.
	Owner OwnerCheck
	// Runs is the build loop's run state (loop.Runs).
	Runs []loop.State
	// Processes lists the processes; nil is HostProcesses.
	Processes ProcessLister
	// Now is the clock, read once; nil is time.Now.
	Now func() time.Time
	// RecentWindow is the recent-write hint's window; zero or less is
	// DefaultRecentWindow.
	RecentWindow time.Duration
}

// Worktree is one worktree as the probe judges it.
type Worktree struct {
	// Path is the worktree's absolute path.
	Path string
	// Changed is the repository-relative, slash-separated paths the worktree
	// changes: its branch's diff from the default branch and its uncommitted
	// and untracked files. A footprint is a set of repository paths, never a
	// worktree's location, so a reservation names a worktree by covering a
	// path it changes.
	//
	// nil means the caller could not compute the changes, and the worktree is
	// held as "changes unknown"; an empty, non-nil slice means it changes
	// nothing.
	Changed []string
}

// Probe judges worktrees against one reading of the process table, taken when
// it is made: a sweep runs the process check once, not once per worktree.
type Probe struct {
	now          time.Time
	window       time.Duration
	reservations claim.Snapshot
	copyView     CopyView
	owner        OwnerCheck
	runs         []loop.State
	processes    []Process
	escaped      []escapedCwd
	unavailable  string
	unread       string
}

// NewProbe reads the process table once and returns the probe that judges
// each worktree against it (spc-2610230730269295, Liveness).
func NewProbe(ctx context.Context, o ProbeOptions) *Probe {
	now, list, window := time.Now, o.Processes, o.RecentWindow
	if o.Now != nil {
		now = o.Now
	}
	if list == nil {
		list = HostProcesses
	}
	if window <= 0 {
		window = DefaultRecentWindow
	}
	p := &Probe{now: now(), window: window, reservations: o.Reservations, copyView: o.CopyView, owner: o.Owner, runs: o.Runs}
	if p.copyView == nil {
		p.copyView = worktreeView
	}
	if p.owner == nil {
		p.owner = foreignOwner
	}
	if o.Reservations.FetchedAt.IsZero() {
		p.unread = "no fetch of the reservations is stamped, so none can be shown absent"
	}
	procs, err := list(ctx)
	if err != nil {
		p.unavailable = err.Error()
		if p.unavailable == "" {
			p.unavailable = "the process lister failed without saying why"
		}
		return p
	}
	// Resolved once here, not once per worktree: the comparison is between
	// real locations on both sides, so a cwd reached through a symlink, or a
	// /var that is really /private/var, still lands inside the worktree.
	for _, proc := range procs {
		switch {
		case !filepath.IsAbs(proc.Cwd):
		case proc.Escaped:
			p.escaped = append(p.escaped, escapedCwd{lit: proc.Cwd, real: realPrefix(proc.Cwd)})
		default:
			p.processes = append(p.processes, Process{PID: proc.PID, Cwd: fsutil.RealExistingPath(proc.Cwd)})
		}
	}
	return p
}

// escapedCwd is the literal prefix of a working directory lsof printed
// escaped: lit as printed, and real with its existing part's symlinks resolved.
type escapedCwd struct {
	lit, real string
}

// realPrefix resolves a literal prefix as the plain directories are resolved,
// keeping a trailing slash, which marks the prefix as ending on a name
// boundary: resolving would drop it and widen the prefix to every sibling
// whose name extends the last one.
func realPrefix(lit string) string {
	real := fsutil.RealExistingPath(lit)
	if strings.HasSuffix(lit, "/") && !strings.HasSuffix(real, "/") {
		real += "/"
	}
	return real
}

// ProcessCheckUnavailable is why the process check could not be made, "" when
// it was. Every worktree is then held as possibly live, and a dry run says so
// once at the top.
func (p *Probe) ProcessCheckUnavailable() string { return p.unavailable }

// ReservationsUnread is why the reservations cannot show that nothing is
// reserved, "" when they can. Every worktree is then held, and a dry run says
// so once at the top.
func (p *Probe) ReservationsUnread() string { return p.unread }

// Judge says whether the worktree is held, and by what.
func (p *Probe) Judge(w Worktree) Liveness {
	wt := fsutil.RealExistingPath(w.Path)
	// Another account's worktree is held outright, and nothing else in it is
	// read: this account's sweep is not the judge of another's tree.
	uid, foreign, err := p.owner(wt)
	if foreign {
		return held([]Evidence{{Kind: KindOwner, Strength: Proven, Detail: fmt.Sprintf("owned by another account (uid %d)", uid)}})
	}
	var ev []Evidence
	if err != nil {
		ev = append(ev, Evidence{Kind: KindOwner, Strength: Hint, Detail: "owner unreadable: " + err.Error()})
	}
	if p.unread != "" {
		ev = append(ev, Evidence{Kind: KindReservationCheck, Strength: Hint, Detail: "reservations unread: " + p.unread})
	}
	ev = append(ev, p.reservationsHolding(wt, w.Changed)...)
	if w.Changed == nil {
		ev = append(ev, Evidence{Kind: KindChanges, Strength: Hint, Detail: "changes unknown"})
	} else if e, ok := unreadableChange(w.Changed); ok {
		ev = append(ev, e)
	}
	ev = append(ev, p.lanesNaming(wt)...)
	if e, ok := p.processIn(wt); ok {
		ev = append(ev, e)
	} else if e, ok := p.escapedNear(wt); ok {
		ev = append(ev, e)
	}
	if p.unavailable != "" {
		ev = append(ev, Evidence{Kind: KindProcessCheck, Strength: Hint, Detail: "process check unavailable: " + p.unavailable})
	}
	if len(ev) == 0 {
		if e, ok := p.recentWrite(wt); ok {
			ev = append(ev, e)
		}
	}
	if len(ev) == 0 {
		return Liveness{Reason: fmt.Sprintf("nothing holds it: no live reservation was made from it or covers its changes, "+
			"no unfinished run names it, no process works in it, and nothing in it was written in the last %s", durationWords(p.window))}
	}
	return held(ev)
}

// held is the liveness of a worktree the evidence holds, its reason the
// evidence joined.
func held(ev []Evidence) Liveness {
	details := make([]string, len(ev))
	for i, e := range ev {
		details[i] = e.Detail
	}
	return Liveness{Held: true, Evidence: ev, Reason: strings.Join(details, "; ")}
}

// reservationsHolding is the reservation proof and its hints. A live
// reservation holds the worktree when its tip names the worktree's own copy
// key, so a session that reserved from it holds it before changing anything,
// or when its footprint overlaps a path the worktree changes; one found both
// ways is named once. A reservation carries no branch, so none is matched to
// the worktree's branch. An unreadable reservation still within its hold has
// a footprint no one can read, so it keeps every worktree, as a hint.
func (p *Probe) reservationsHolding(wt string, changed []string) []Evidence {
	var out []Evidence
	named := map[string]bool{}
	mine, err := p.copyView(wt, p.now)
	if err != nil {
		out = append(out, Evidence{Kind: KindReservationCheck, Strength: Hint, Detail: "this worktree's reservations unread: " + err.Error()})
	}
	for _, r := range mine.Reservations {
		if r.State == claim.StateLive && r.Copy == claim.CopyThis && !named[r.Record] {
			named[r.Record] = true
			out = append(out, Evidence{Kind: KindReservation, Strength: Proven, Detail: fmt.Sprintf(
				"reserved as %s by %s from this worktree until %s",
				r.Record, r.Holder, r.EffectiveExpiry.UTC().Format(time.RFC3339))})
		}
	}
	for _, r := range p.reservations.Reservations {
		switch {
		case named[r.Record]:
		case r.State == claim.StateLive:
			if shared, ok := firstOverlap(r.Footprint, changed); ok {
				named[r.Record] = true
				out = append(out, Evidence{Kind: KindReservation, Strength: Proven, Detail: fmt.Sprintf(
					"reserved as %s by %s until %s, its footprint covering %s",
					r.Record, r.Holder, r.EffectiveExpiry.UTC().Format(time.RFC3339), shared)})
			}
		case r.State == claim.StateUnreadable && r.EffectiveExpiry.After(p.now):
			holder := r.Holder
			if holder == "" {
				holder = "a holder its tip does not name readably"
			}
			out = append(out, Evidence{Kind: KindReservation, Strength: Hint, Detail: fmt.Sprintf(
				"reservation %s by %s is unreadable (%s) and holds until %s, its footprint unknown",
				r.Record, holder, r.Unreadable, r.EffectiveExpiry.UTC().Format(time.RFC3339))})
		}
	}
	return out
}

// worktreeView is the default CopyView: claim.View on the worktree, which
// reads only its local refs and its own copy key. A worktree that keeps no
// copy key has made no reservation, so nothing is read from it.
func worktreeView(wt string, now time.Time) (claim.Snapshot, error) {
	if claim.OwnCopyKey(wt) == "" {
		return claim.Snapshot{}, nil
	}
	return claim.View(wt, now)
}

// foreignOwner is the default OwnerCheck: the owner of the worktree's top
// directory, then of its .git (a file in a linked worktree, a directory in a
// main one), read through fsutil.OwnerUID and compared with this process's
// uid. A .git that is absent is passed over; any other failure to read an
// owner is an error, and the worktree is kept.
func foreignOwner(wt string) (int, bool, error) {
	self := os.Getuid()
	for _, name := range []string{wt, filepath.Join(wt, ".git")} {
		uid, err := fsutil.OwnerUID(name)
		if name != wt && errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			var pe *fs.PathError
			if errors.As(err, &pe) {
				err = pe.Err
			}
			return 0, false, fmt.Errorf("cannot read the owner of %s: %v", filepath.Base(name), err)
		}
		if int(uid) != self {
			return int(uid), true, nil
		}
	}
	return self, false, nil
}

// unreadableChange is the hint for a changed path that is not a clean
// repository-relative one (a backslash in it, say): no footprint can be
// compared with it, so none can be shown not to cover it, and the worktree is
// kept as it is when its changes are unknown. "" and "." name no path at all
// and are passed over.
func unreadableChange(changed []string) (Evidence, bool) {
	var first string
	n := 0
	for _, c := range changed {
		if c == "" || c == "." || fsutil.ValidRelPath(c) {
			continue
		}
		if n == 0 {
			first = c
		}
		n++
	}
	if n == 0 {
		return Evidence{}, false
	}
	detail := "changes unreadable: " + printable(first)
	if n > 1 {
		detail += fmt.Sprintf(" (and %d more)", n-1)
	}
	return Evidence{Kind: KindChanges, Strength: Hint, Detail: detail}, true
}

// printable is s as it is, or quoted when it holds a byte a terminal would act
// on rather than show, so a reason stays one readable line.
func printable(s string) string {
	if strings.IndexFunc(s, func(r rune) bool { return !unicode.IsPrint(r) }) >= 0 {
		return strconv.Quote(s)
	}
	return s
}

// firstOverlap is the first changed path a footprint path covers, or is
// covered by, compared on a path boundary. A path that is not a clean
// repository-relative one is skipped here, since "" or "." would cover
// everything; unreadableChange keeps the worktree for any such path but those
// two.
func firstOverlap(footprint, changed []string) (string, bool) {
	for _, c := range changed {
		if !fsutil.ValidRelPath(c) {
			continue
		}
		for _, f := range footprint {
			if fsutil.ValidRelPath(f) && fsutil.PathsOverlap(filepath.FromSlash(f), filepath.FromSlash(c), fsutil.CaseFoldingFS()) {
				return c, true
			}
		}
	}
	return "", false
}

// lanesNaming is one proof per lane of an unfinished run within its lease
// whose worktree is this one (or inside it). A run takes its claim for
// lease.Max (implement.MaxLease, loop.go) and every step it takes rewrites
// UpdatedAt, so a state no step has written for that long is a run whose
// session is gone; it is judged through lease.Judge, the one liveness
// judgement every claim in abcd is read through, so a state dated ahead of
// the clock holds no longer than a day from now.
func (p *Probe) lanesNaming(wt string) []Evidence {
	var out []Evidence
	for _, st := range p.runs {
		if st.Complete() {
			continue
		}
		from := st.UpdatedAt
		if st.CreatedAt.After(from) {
			from = st.CreatedAt
		}
		v := lease.Judge(lease.Stamp{ClaimedAt: from, ExpiresAt: from.Add(lease.Max)}, 0, p.now)
		if !v.Live {
			continue
		}
		for _, l := range st.Lanes {
			if l.Worktree != "" && within(fsutil.RealExistingPath(l.Worktree), wt) {
				out = append(out, Evidence{Kind: KindRun, Strength: Proven, Detail: fmt.Sprintf(
					"%s of build run %s (%s) works in it, its lease running to %s",
					l.ID, st.RunID, st.Key, v.EffectiveExpiry.UTC().Format(time.RFC3339))})
			}
		}
	}
	return out
}

// processIn is the process proof: one piece of evidence naming the first
// process whose working directory is the worktree or inside it.
func (p *Probe) processIn(wt string) (Evidence, bool) {
	var first, n int
	for _, proc := range p.processes {
		if within(proc.Cwd, wt) {
			if n == 0 {
				first = proc.PID
			}
			n++
		}
	}
	if n == 0 {
		return Evidence{}, false
	}
	detail := fmt.Sprintf("in use: process %d works in it", first)
	if n > 1 {
		detail += fmt.Sprintf(" (and %d more)", n-1)
	}
	return Evidence{Kind: KindProcess, Strength: Proven, Detail: detail}, true
}

// escapedNear is the hint for a process whose working directory lsof printed
// escaped: it names the first such directory whose literal prefix and the
// worktree are one a plain string prefix of the other, after the folding
// within uses. Every byte before the escape is the directory's own, so a
// prefix that leads into the worktree, or that the worktree extends, may be
// it; one that leads elsewhere cannot be. The comparison is not on a name
// boundary because the escape can fall in the middle of a name. It is a hint,
// not a proof: the directory is known only as far as the escape.
func (p *Probe) escapedNear(wt string) (Evidence, bool) {
	fold := fsutil.CaseFoldingFS()
	w := fsutil.FoldPath(wt, fold)
	for _, e := range p.escaped {
		for _, lit := range []string{e.lit, e.real} {
			l := fsutil.FoldPath(lit, fold)
			if strings.HasPrefix(l, w) || strings.HasPrefix(w, l) {
				return Evidence{Kind: KindProcess, Strength: Hint,
					Detail: "in use: a process works in a directory lsof printed escaped, under " + e.lit}, true
			}
		}
	}
	return Evidence{}, false
}

// within reports whether child is parent or inside it, on a path boundary:
// /a/bc is not inside /a/b. Both sides are real paths already.
func within(child, parent string) bool {
	return fsutil.PathWithin(child, parent, fsutil.CaseFoldingFS())
}

// errFound stops the tree walk at the first recent write.
var errFound = errors.New("found a recent write")

// recentWrite is the hint: the worktree's index, or anything in its tree,
// written within the window. A time ahead of the clock counts as recent,
// erring toward keeping the worktree.
func (p *Probe) recentWrite(wt string) (Evidence, bool) {
	cutoff := p.now.Add(-p.window)
	window := durationWords(p.window)
	if fi, err := os.Stat(filepath.Join(ownGitDir(wt), "index")); err == nil && fi.ModTime().After(cutoff) {
		return Evidence{Kind: KindRecentWrite, Strength: Hint, Detail: fmt.Sprintf(
			"its index was written at %s, within the last %s", fi.ModTime().UTC().Format(time.RFC3339), window)}, true
	}
	var found string
	var when time.Time
	_ = filepath.WalkDir(wt, func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			// An unreadable directory is passed over rather than failing the
			// hint: the hint only ever adds a keep, so a gap in it can free
			// nothing a proof would hold.
			if d != nil && d.IsDir() && name != wt {
				return fs.SkipDir
			}
			return nil
		}
		// The worktree's own .git is git's bookkeeping, which a fetch or gc
		// touches with no session at work; the index is read above instead.
		if d.Name() == ".git" && filepath.Dir(name) == wt {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		fi, err := d.Info()
		if err == nil && fi.ModTime().After(cutoff) {
			found, when = name, fi.ModTime()
			return errFound
		}
		return nil
	})
	if found == "" {
		return Evidence{}, false
	}
	rel, err := filepath.Rel(wt, found)
	if err != nil || rel == "." {
		rel = "the worktree's top directory"
	}
	return Evidence{Kind: KindRecentWrite, Strength: Hint, Detail: fmt.Sprintf(
		"%s was written at %s, within the last %s", filepath.ToSlash(rel), when.UTC().Format(time.RFC3339), window)}, true
}

// ownGitDir is the worktree's own git dir: .git itself in a main worktree, and
// in a linked one the directory its .git file points at, where that worktree's
// index lives. The file is read bounded; one that cannot be read names a git
// dir with no index in it, and the index adds no hint.
func ownGitDir(wt string) string {
	dotGit := filepath.Join(wt, ".git")
	fi, err := os.Lstat(dotGit)
	if err != nil || fi.IsDir() || !fi.Mode().IsRegular() {
		return dotGit
	}
	f, err := os.Open(dotGit)
	if err != nil {
		return dotGit
	}
	defer f.Close()
	line, _ := bufio.NewReader(io.LimitReader(f, 4<<10)).ReadString('\n')
	dir, ok := strings.CutPrefix(strings.TrimSpace(line), "gitdir: ")
	if !ok || dir == "" {
		return dotGit
	}
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(wt, dir)
	}
	return dir
}

// durationWords is a window as a person writes it: 24h, not 24h0m0s.
func durationWords(d time.Duration) string {
	s := d.String()
	if strings.HasSuffix(s, "m0s") {
		s = strings.TrimSuffix(s, "0s")
	}
	if strings.HasSuffix(s, "h0m") {
		s = strings.TrimSuffix(s, "0m")
	}
	return s
}

// hostLsof is the macOS process check. It runs by absolute path, never
// resolved through PATH, so a directory early on PATH cannot stand in an lsof
// that answers "nothing is running"; the deadline and the read bound keep a
// wedged or runaway lsof from stalling the sweep or filling memory.
var hostLsof = lsofCommand{path: "/usr/sbin/lsof", timeout: 5 * time.Second, limit: 4 << 20}

// HostProcesses lists this host's processes with no cgo, since the released
// binaries are cross-compiled: /proc on Linux, lsof on macOS. Any other host
// has no process check, which fails closed.
func HostProcesses(ctx context.Context) ([]Process, error) {
	switch runtime.GOOS {
	case "linux":
		return procCwds("/proc")
	case "darwin":
		return hostLsof.list(ctx)
	}
	return nil, fmt.Errorf("abcd has no process check on %s", runtime.GOOS)
}

// lsofCommand is an lsof to run: its path, deadline and read bound.
type lsofCommand struct {
	path    string
	timeout time.Duration
	limit   int64
}

// list runs lsof for every process's working directory. A missing lsof, one
// that times out, exits non-zero, prints more than the bound or prints what
// parseLsof cannot read is an error, never an empty list: an empty list would
// free every worktree.
func (c lsofCommand) list(ctx context.Context) ([]Process, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, c.path, "-a", "-d", "cwd", "-Fn")
	// lsof escapes a byte its locale cannot print (é as \xc3\xa9) and still
	// exits 0, so it runs under a UTF-8 locale, and with nothing else from
	// this process's environment. en_US.UTF-8 is the name every macOS ships;
	// C.UTF-8 is missing from older ones. A name it escapes anyway (a
	// newline, say) parseLsof keeps only up to the escape.
	cmd.Env = []string{"LC_ALL=en_US.UTF-8"}
	out := &boundedBuffer{limit: c.limit, over: cancel}
	cmd.Stdout = out
	// A child lsof leaves holding the pipe cannot keep Wait past the deadline.
	cmd.WaitDelay = time.Second
	err := cmd.Run()
	switch {
	case out.exceeded:
		return nil, fmt.Errorf("lsof printed more than %s", sizeWords(c.limit))
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return nil, fmt.Errorf("lsof timed out after %s", c.timeout)
	case errors.Is(err, exec.ErrWaitDelay):
		return nil, fmt.Errorf("lsof exited but left a process holding its output")
	case errors.Is(err, fs.ErrNotExist):
		return nil, fmt.Errorf("lsof is missing at %s", c.path)
	case err != nil:
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return nil, fmt.Errorf("lsof exited with status %d", exit.ExitCode())
		}
		return nil, fmt.Errorf("lsof could not run: %v", err)
	}
	procs, err := parseLsof(out.buf)
	if err != nil {
		return nil, fmt.Errorf("lsof printed unreadable output: %v", err)
	}
	return procs, nil
}

// boundedBuffer keeps at most limit bytes. Past it, it stops lsof through
// over rather than refusing the write: a refused write would leave lsof
// blocked on a full pipe until the deadline.
type boundedBuffer struct {
	buf      []byte
	limit    int64
	over     func()
	exceeded bool
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	if b.exceeded {
		return len(p), nil
	}
	if int64(len(b.buf))+int64(len(p)) > b.limit {
		b.exceeded = true
		b.over()
		return len(p), nil
	}
	b.buf = append(b.buf, p...)
	return len(p), nil
}

// sizeWords is a read bound as a person writes it.
func sizeWords(n int64) string {
	if n >= 1<<20 && n%(1<<20) == 0 {
		return fmt.Sprintf("%d MiB", n>>20)
	}
	return fmt.Sprintf("%d bytes", n)
}

// parseLsof reads `lsof -a -d cwd -Fn`: per process a `p<pid>` line, then per
// descriptor an `f<fd>` line and an `n<name>` line. It is strict, since a line
// it misread could hide a process: an unknown field, a blank line, a NUL, a
// field out of order, a pid that is not a number or a name that is not an
// absolute path is an error. So is no process at all, because lsof always
// lists itself. Errors name the line number, never its text.
//
// A name holding a backslash or a caret is kept, cut before the first of them
// and marked Escaped. lsof doubles a literal backslash, writes a byte it will
// not print as a backslash escape, and writes most control bytes in caret
// form with no backslash, but every byte before the first backslash or caret
// is the directory's own. Read whole, an escaped name would match no
// worktree and hide its process; refused, it would stop the check for every
// worktree. The probe holds instead only the worktrees its prefix may reach.
func parseLsof(out []byte) ([]Process, error) {
	if len(out) == 0 {
		return nil, errors.New("no process listed, though lsof always lists itself")
	}
	var procs []Process
	pid, fd := -1, ""
	for i, line := range strings.Split(strings.TrimSuffix(string(out), "\n"), "\n") {
		n := i + 1
		if line == "" {
			return nil, fmt.Errorf("line %d is blank", n)
		}
		if strings.IndexByte(line, 0) >= 0 {
			return nil, fmt.Errorf("line %d holds a NUL", n)
		}
		val := line[1:]
		switch line[0] {
		case 'p':
			v, err := strconv.Atoi(val)
			if err != nil || v < 0 {
				return nil, fmt.Errorf("line %d: a process id that is not a number", n)
			}
			pid, fd = v, ""
		case 'f':
			if pid < 0 {
				return nil, fmt.Errorf("line %d: a descriptor before any process", n)
			}
			fd = val
		case 'n':
			if pid < 0 || fd == "" {
				return nil, fmt.Errorf("line %d: a name before its process and descriptor", n)
			}
			if !path.IsAbs(val) {
				return nil, fmt.Errorf("line %d: a name that is not an absolute path", n)
			}
			if fd != "cwd" {
				continue
			}
			if i := strings.IndexAny(val, `\\^`); i >= 0 {
				procs = append(procs, Process{PID: pid, Cwd: val[:i], Escaped: true})
			} else {
				procs = append(procs, Process{PID: pid, Cwd: val})
			}
		default:
			return nil, fmt.Errorf("line %d: unknown field %q", n, line[0])
		}
	}
	return procs, nil
}

// procCwds reads every process's working directory from a /proc tree. A
// process that exited mid-listing, or one another account owns, is passed
// over: neither is a session working in this account's worktrees, and
// refusing on them would fail every check on a shared host. A /proc that
// cannot be listed, or a link that fails any other way, is an error. So is a
// /proc that does not list this process: one that is empty or another's, as
// in a chroot or a container, cannot show that nothing else is running.
func procCwds(root string) ([]Process, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		var pe *fs.PathError
		if errors.As(err, &pe) {
			err = pe.Err
		}
		return nil, fmt.Errorf("cannot read %s: %v", root, err)
	}
	var procs []Process
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid <= 0 {
			continue
		}
		cwd, err := os.Readlink(filepath.Join(root, e.Name(), "cwd"))
		switch {
		case errors.Is(err, fs.ErrNotExist), errors.Is(err, fs.ErrPermission):
			continue
		case err != nil:
			return nil, fmt.Errorf("cannot read the working directory of process %d: %v", pid, err)
		}
		procs = append(procs, Process{PID: pid, Cwd: cwd})
	}
	self := os.Getpid()
	if !slices.ContainsFunc(procs, func(p Process) bool { return p.PID == self }) {
		return nil, fmt.Errorf("this process is not in %s", root)
	}
	return procs, nil
}
