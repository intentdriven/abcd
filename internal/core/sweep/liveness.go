// Package sweep is the core of `abcd sweep` (spc-2610230730269295): it judges
// what a repository has left lying around and clears only what is provably
// finished.
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
	"strconv"
	"strings"
	"time"

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
	// KindReservation, KindRun and KindProcess are the three proofs.
	KindReservation EvidenceKind = "reservation"
	KindRun         EvidenceKind = "run"
	KindProcess     EvidenceKind = "process"
	// KindRecentWrite is the recent-write hint.
	KindRecentWrite EvidenceKind = "recent-write"
	// KindProcessCheck is the process check failing closed: it proves
	// nothing, so it is a hint, but the worktree is kept as possibly live.
	KindProcessCheck EvidenceKind = "process-check"
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
}

// ProcessLister lists the running processes' working directories. An error
// means the check could not be made, and the probe then fails closed.
type ProcessLister func(ctx context.Context) ([]Process, error)

// ProbeOptions is what the probe reads. The caller fetches and reads the
// stores; the probe reads only the worktree and the process table.
type ProbeOptions struct {
	// Reservations is the claim view after the sweep's fetch (claim.View).
	Reservations claim.Snapshot
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
	Changed []string
}

// Probe judges worktrees against one reading of the process table, taken when
// it is made: a sweep runs the process check once, not once per worktree.
type Probe struct {
	now          time.Time
	window       time.Duration
	reservations claim.Snapshot
	runs         []loop.State
	processes    []Process
	unavailable  string
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
	p := &Probe{now: now(), window: window, reservations: o.Reservations, runs: o.Runs}
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
		if filepath.IsAbs(proc.Cwd) {
			p.processes = append(p.processes, Process{PID: proc.PID, Cwd: fsutil.RealExistingPath(proc.Cwd)})
		}
	}
	return p
}

// ProcessCheckUnavailable is why the process check could not be made, "" when
// it was. Every worktree is then held as possibly live, and a dry run says so
// once at the top.
func (p *Probe) ProcessCheckUnavailable() string { return p.unavailable }

// Judge says whether the worktree is held, and by what.
func (p *Probe) Judge(w Worktree) Liveness {
	wt := fsutil.RealExistingPath(w.Path)
	var ev []Evidence
	ev = append(ev, p.reservationsCovering(w.Changed)...)
	ev = append(ev, p.lanesNaming(wt)...)
	if e, ok := p.processIn(wt); ok {
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
		return Liveness{Reason: fmt.Sprintf("nothing holds it: no live reservation covers its changes, no unfinished run names it, "+
			"no process works in it, and nothing in it was written in the last %s", durationWords(p.window))}
	}
	details := make([]string, len(ev))
	for i, e := range ev {
		details[i] = e.Detail
	}
	return Liveness{Held: true, Evidence: ev, Reason: strings.Join(details, "; ")}
}

// reservationsCovering is one proof per live reservation whose footprint
// overlaps a path the worktree changes. This copy's own reservations count
// too: a session in this checkout is as live as one elsewhere.
func (p *Probe) reservationsCovering(changed []string) []Evidence {
	var out []Evidence
	for _, r := range p.reservations.Reservations {
		if r.State != claim.StateLive {
			continue
		}
		if shared, ok := firstOverlap(r.Footprint, changed); ok {
			out = append(out, Evidence{Kind: KindReservation, Strength: Proven, Detail: fmt.Sprintf(
				"reserved as %s by %s until %s, its footprint covering %s",
				r.Record, r.Holder, r.EffectiveExpiry.UTC().Format(time.RFC3339), shared)})
		}
	}
	return out
}

// firstOverlap is the first changed path a footprint path covers, or is
// covered by, compared on a path boundary. A path that is not a clean
// repository-relative one is skipped: "" or "." would cover everything.
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
	detail := fmt.Sprintf("process %d works in it", first)
	if n > 1 {
		detail += fmt.Sprintf(" (and %d more)", n-1)
	}
	return Evidence{Kind: KindProcess, Strength: Proven, Detail: detail}, true
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
			if fd == "cwd" {
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
// cannot be listed, or a link that fails any other way, is an error.
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
	return procs, nil
}
