package sweep

// sweep.go is the dry run (spc-2610230730269295, step 1): one row per
// worktree git lists, in the store or outside it, per stray in the store's
// lane and for the embargo store, each with the action `--apply` would take
// or the reason it leaves the item. It writes nothing.
//
// The store's list is the one reading of the worktrees, their state and the
// merged judgement, and its Held is the build loop's hold as prune reads it:
// the sweep composes those judgements with the liveness probe and never
// re-makes them.
//
// runDry, runApply, newProbe and hostProcesses are the package's entry points
// and stay unexported until the surfaces (step 5) call them: an exported core
// function no front door reaches is scaffolding the reach audit refuses.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	store "github.com/intentdriven/abcd/internal/core/ahoy/worktree"
	"github.com/intentdriven/abcd/internal/core/embargo/storepath"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// dirtyKept is the reason on a dirty worktree nothing holds, the dry run's and
// the apply's alike: patch salvage is not built, so it is kept and its
// changes are not saved.
const dirtyKept = "dirty, and nothing holds it: kept, since the sweep does not yet save uncommitted changes"

// changesCap bounds each read of a worktree's changed paths: past it the
// changes read as unknown, which keeps the worktree.
const changesCap = 4 << 20

// Options is one dry run. The caller fetches first (criterion 11) and hands
// the reservations it read in Probe.Reservations: the dry run itself reaches
// no remote.
type Options struct {
	// Probe is what the liveness probe reads: the reservations already
	// fetched, the build loop's runs, the process lister and the clock.
	Probe ProbeOptions
	// Forge is the store's forge client for the merged judgement; nil asks
	// none, and merged is judged from git alone.
	Forge store.GH
	// Held reads what the build loop holds, as the store's prune reads it. It
	// is required: like prune, a sweep that cannot tell what the loop holds
	// acts on nothing.
	Held store.HeldReader
	// Cwd is the sweep's own working directory; the worktree holding it is
	// kept, as prune keeps its own. It is required, as Held is: a sweep that
	// does not know where it runs could remove the worktree it runs in.
	Cwd string

	// changes reads the paths a worktree changes, running git in it; nil is
	// changedPaths. A test sets it to see which worktrees git ran in.
	changes func(abs, def string) []string
	// pruneOne reclaims one worktree of the lane; nil is the store's
	// PruneOne. A test sets it to see which worktrees the prune was asked
	// about.
	pruneOne func(repoRoot, name string, opts store.PruneOptions) (store.Pruned, error)
}

// runDry judges every worktree of the repository at repoRoot and every entry
// in its lane, and writes nothing. A store refusal, a failure to list the
// worktrees, or a failure to read what the build loop holds is returned as
// the error, and the result carries it as its Refusal, so ExitCode reads
// ExitRefused. A run without Held or Cwd is refused the same way.
func runDry(ctx context.Context, repoRoot string, o Options) (Result, error) {
	res, _, _, err := judgeRepo(ctx, repoRoot, o)
	return res, err
}

// judgeRepo is runDry, also answering with the store's row behind each of the
// result's rows, in order, and the judge that made them, for runApply to act on.
func judgeRepo(ctx context.Context, repoRoot string, o Options) (Result, []store.Row, judge, error) {
	fail := func(err error) (Result, []store.Row, judge, error) {
		res, err := refused(err)
		return res, nil, judge{}, err
	}
	if o.Held == nil {
		return fail(errors.New("sweep: a dry run needs a reader of what the build loop holds"))
	}
	if o.Cwd == "" {
		return fail(errors.New("sweep: a dry run needs its own working directory, to keep the worktree holding it"))
	}
	listing, err := store.List(repoRoot, store.ListOptions{Forge: o.Forge})
	if err != nil {
		return fail(err)
	}
	held, err := o.Held(repoRoot)
	if err != nil {
		return fail(fmt.Errorf("reading what the build loop holds: %w", err))
	}
	home, _ := os.UserHomeDir()
	p := newProbe(ctx, o.Probe)
	j := judge{
		home:    home,
		def:     gitutil.DefaultRef(repoRoot),
		probe:   p,
		held:    held,
		changes: o.changes,
	}
	if j.changes == nil {
		j.changes = changedPaths
	}
	j.cwd = fsutil.RealExistingPath(o.Cwd)
	res := Result{
		Lane:                    listing.Lane,
		Rows:                    make([]Row, 0, len(listing.Worktrees)),
		ProcessCheckUnavailable: clean(p.ProcessCheckUnavailable()),
		ReservationsUnread:      clean(p.ReservationsUnread()),
		ForgeNote:               listing.ForgeNote,
	}
	for _, r := range listing.Worktrees {
		res.Rows = append(res.Rows, j.row(r))
	}
	return res, listing.Worktrees, j, nil
}

// refused is the result and error of a run that refused before judging. The
// error says only the cleaned text the Refusal carries, so a front door that
// prints it shows no home path; the cause stays reachable through errors.Is.
func refused(err error) (Result, error) {
	msg := clean(err.Error())
	return Result{Rows: []Row{}, Refusal: msg}, &refusal{msg: msg, cause: err}
}

// refusal is a refused run's error: the cleaned text, wrapping its cause.
type refusal struct {
	msg   string
	cause error
}

func (r *refusal) Error() string { return r.msg }
func (r *refusal) Unwrap() error { return r.cause }

// clean is text made display-ready: home-redacted, and masked so a byte a
// terminal would act on is shown instead.
func clean(s string) string { return termsafe.Sanitize(fsutil.RedactHome(s)) }

// judge is what every row of one dry run is judged against.
type judge struct {
	home    string
	def     string
	cwd     string
	probe   *Probe
	held    store.Held
	changes func(abs, def string) []string
}

// row judges one store row, its item at the row's Abs: the store's own
// absolute path, as git or the lane names it, so no row needs matching back to
// a worktree by its displayed name. The order is the spec's: the embargo store and what the store
// did not judge first, read no further; then liveness, an external lock, the
// sweep's own directory and the loop's hold; then outside the store; then
// dirty, unmerged and, last, merged and clean.
func (j judge) row(r store.Row) Row {
	abs := r.Abs
	row := Row{Kind: RowWorktree, Name: r.Name, Path: r.Path, Action: ActionKeep}
	keep := func(reason string, ev ...Evidence) Row {
		row.Reason = clean(reason)
		for _, e := range ev {
			e.Detail = clean(e.Detail)
			row.Evidence = append(row.Evidence, e)
		}
		return row
	}
	storeSaid := func(s Strength, detail string) Evidence {
		return Evidence{Kind: KindStore, Strength: s, Detail: detail}
	}

	if r.Where == store.WhereSibling || storepath.IsStorePath(j.home, abs) {
		row.Kind, row.State = RowEmbargoStore, "abcd's embargo store; not read"
		return keep("abcd's embargo store: never read, saved or removed",
			storeSaid(Proven, firstNonEmpty(r.Reason, "the embargo store's path")))
	}
	if r.Branch == "" {
		if fi, err := os.Lstat(abs); err == nil && !fi.IsDir() {
			row.Kind, row.State = RowStray, "an entry in the store's lane that is not a directory"
			return keep("stray: not a worktree; listed and never touched", storeSaid(Proven, firstNonEmpty(r.Reason, "not a directory")))
		}
	}
	row.State = stateOf(r)
	if r.Reason != "" {
		return keep("cannot classify: "+r.Reason, storeSaid(Proven, r.Reason))
	}
	if !slices.Contains([]string{"clean", "dirty"}, r.State) || !slices.Contains([]string{"merged", "unmerged"}, r.Merged) {
		return keep(fmt.Sprintf("cannot classify: the store read its state as %q and its merge as %q", r.State, r.Merged),
			storeSaid(Hint, "an unknown state or merge"))
	}

	live := j.liveness(abs)
	ev := slices.Clone(live.Evidence)
	var reasons []string
	if live.Held {
		reasons = append(reasons, live.Reason)
	}
	if r.Locked {
		lock := lockedBy(r.LockReason)
		ev = append(ev, Evidence{Kind: KindLock, Strength: Proven, Detail: lock})
		reasons = append(reasons, lock)
	}
	if j.cwd != "" && fsutil.PathWithin(j.cwd, fsutil.RealExistingPath(abs), fsutil.CaseFoldingFS()) {
		ev = append(ev, Evidence{Kind: KindCwd, Strength: Proven, Detail: "the sweep's own working directory"})
		reasons = append(reasons, "the sweep's own working directory")
	}
	if r.Where == store.WhereStore {
		if run := j.held.By(r.Name); run != "" {
			held := "held by the build loop's run " + run
			ev = append(ev, Evidence{Kind: KindLoop, Strength: Proven, Detail: held})
			reasons = append(reasons, held)
		}
	}
	ev = append(ev, Evidence{Kind: KindState, Strength: Proven, Detail: r.State})
	merged := r.Merged
	if r.Merged == "merged" {
		merged = "merged, judged by " + r.JudgedBy
	}
	ev = append(ev, Evidence{Kind: KindMerged, Strength: Proven, Detail: merged})

	switch {
	case len(reasons) > 0:
		return keep(reasons[0], ev...)
	case r.Where != store.WhereStore:
		return keep("outside the store; never touched", ev...)
	case r.State == "dirty":
		row.Action = ActionSavePatch
		return keep(dirtyKept, ev...)
	case r.Merged == "unmerged":
		return keep("unmerged", ev...)
	}
	row.Action = ActionRemove
	return keep(merged+", clean, and "+live.Reason, ev...)
}

// liveness is the probe's judgement of the worktree at abs. Its owner is asked
// before any git command: a worktree another account owns, or whose owner
// cannot be read, has no git run in it by this account, and the probe keeps it
// on the owner (its changes then read as unknown, which keeps it too).
func (j judge) liveness(abs string) Liveness {
	w := Worktree{Path: abs}
	if _, foreign, err := j.probe.owner(fsutil.RealExistingPath(abs)); !foreign && err == nil {
		w.Changed = j.changes(abs, j.def)
	}
	return j.probe.Judge(w)
}

// lockedBy is the keep reason for git's lock: its reason shown whole, and the
// owner a JSON reason names, as text or as the JSON it was given.
func lockedBy(reason string) string {
	if reason == "" {
		return "locked by git"
	}
	var v struct {
		Owner json.RawMessage `json:"owner"`
	}
	if json.Unmarshal([]byte(reason), &v) == nil && len(v.Owner) > 0 && string(v.Owner) != "null" {
		var owner string
		if json.Unmarshal(v.Owner, &owner) != nil {
			owner = string(v.Owner)
		}
		if owner = strings.TrimSpace(owner); owner != "" {
			return "locked by " + owner + ": " + reason
		}
	}
	return "locked by git: " + reason
}

// stateOf is what the store read of a worktree, in words.
func stateOf(r store.Row) string {
	var parts []string
	switch r.Branch {
	case "":
	case "detached":
		parts = append(parts, "detached HEAD")
	default:
		parts = append(parts, "branch "+r.Branch)
	}
	if r.Reason != "" {
		parts = append(parts, "not judged by the store")
	}
	if r.State != "" {
		parts = append(parts, r.State)
	}
	switch r.Merged {
	case "merged":
		parts = append(parts, "merged by "+r.JudgedBy)
	case "":
	default:
		parts = append(parts, r.Merged)
	}
	if r.Locked {
		parts = append(parts, "locked")
	}
	if r.Where == store.WhereOutside {
		parts = append(parts, "outside the store")
	}
	if len(parts) == 0 {
		return "a directory in the store"
	}
	return strings.Join(parts, ", ")
}

// changedPaths is the repository-relative paths the worktree at abs changes,
// as the liveness probe compares them with a reservation's footprint: its
// branch's diff from where it left the default branch, and its staged,
// unstaged and untracked paths. nil, which the probe holds as "changes
// unknown", when there is no default branch or either read fails.
func changedPaths(abs, def string) []string {
	if def == "" {
		return nil
	}
	out, err := gitutil.RunCappedBytes(abs, changesCap, "diff", "--name-only", "-z", "--no-renames", def+"...HEAD", "--")
	if err != nil {
		return nil
	}
	st, err := gitutil.Status(abs, changesCap, gitutil.StatusOptions{})
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	changed := []string{}
	add := func(p string) {
		p = strings.TrimSuffix(p, "/")
		if p != "" && !seen[p] {
			seen[p] = true
			changed = append(changed, p)
		}
	}
	for _, p := range strings.Split(string(out), "\x00") {
		add(p)
	}
	for _, e := range st {
		add(e.Path)
		add(e.Orig)
	}
	slices.Sort(changed)
	return changed
}

func firstNonEmpty(s ...string) string {
	for _, v := range s {
		if v != "" {
			return v
		}
	}
	return ""
}
