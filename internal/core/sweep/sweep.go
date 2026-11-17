package sweep

// sweep.go is the dry run (spc-2610230730269295, step 1): one row per
// worktree git lists, in the store or outside it, per stray in the store's
// lane and for the embargo store, each with the action `--apply` would take
// or the reason it leaves the item. It writes nothing: a dirty worktree's
// salvage is collected here, so the row says what apply would save, and
// written only by apply (salvage.go).
//
// The store's list is the one reading of the worktrees, their state and the
// merged judgement, and its Held is the build loop's hold as prune reads it:
// the sweep composes those judgements with the liveness probe and never
// re-makes them.
//
// Run is the package's entry point, the one the surfaces call (step 5): the
// sweep's one fetch, then runSweep, which runs every half of a sweep in the
// spec's order.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	store "github.com/intentdriven/abcd/internal/core/ahoy/worktree"
	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/embargo/storepath"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/record/match"
	"github.com/intentdriven/abcd/internal/core/report"
	"github.com/intentdriven/abcd/internal/core/reservesweep"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

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
	// Forge is the forge client every half asks: the store's merged
	// judgement, the pull requests on origin's branches and the reservation
	// workflow's dispatch. nil asks none, and merged is judged from git
	// alone. Every call is pinned to origin's repository (forge.go); when it
	// cannot be, none is made.
	Forge store.GH
	// Held reads what the build loop holds, as the store's prune reads it. It
	// is required: like prune, a sweep that cannot tell what the loop holds
	// acts on nothing.
	Held store.HeldReader
	// Cwd is the sweep's own working directory; the worktree holding it is
	// kept, as prune keeps its own. It is required, as Held is: a sweep that
	// does not know where it runs could remove the worktree it runs in.
	Cwd string
	// Match is the filing-time match each promoted inbox report's record
	// runs, configured as the inbox's promote verb configures it; nil files
	// each unmatched.
	Match *match.Config

	// changes reads the paths a worktree changes, running git in it; nil is
	// changedPaths. A test sets it to see which worktrees git ran in.
	changes func(abs, def string) []string
	// pruneOne reclaims one worktree of the lane; nil is the store's
	// PruneOne. A test sets it to see which worktrees the prune was asked
	// about.
	pruneOne func(repoRoot, name string, opts store.PruneOptions) (store.Pruned, error)
	// reservations is the reservation sweep's judgement; nil is
	// reservesweep.Sweep. It is always asked with Apply false. A test sets
	// it to judge reservations without a fetch.
	reservations func(root string, o reservesweep.Options) (reservesweep.Report, error)
	// inbox lists the reports waiting in abcd's inbox; nil is report.List.
	// promote files one of them; nil is report.Promote. A test sets either
	// to see what was asked, or to have a promotion fail.
	inbox   func() ([]report.Entry, error)
	promote func(ledgerRoot, id string, mc *match.Config) (report.Promoted, error)
	// onLastCheck, when set, is told each stage the last check before a
	// delete on origin has passed (lastCheckWorktrees, lastCheckProbe). A
	// test sets it to end the run's context at that point.
	onLastCheck func(stage string)
	// ours, when set, says which branches the sweep may delete, locally and
	// on origin: the sweep abcd runs on its own sets it to the branches abcd
	// made (abcdMade), and every other branch judged deletable is kept and
	// listed for the applying sweep a person runs. nil is every branch, as
	// the applying sweep judges them.
	ours func(name string) bool
	// ignored lists what a removal would delete unarchived from a worktree
	// judged spent: its ignored paths outside the local tier; nil is
	// store.IgnoredForRemoval. Every sweep names them on the worktree's row
	// (withIgnored). A test sets it to stand in for a listing that fails.
	ignored func(dir string) (store.IgnoredState, error)
	// keepIgnored is set by the sweep abcd runs on its own (Automatic): a
	// worktree holding any ignored path outside its local tier, or whose
	// listing fails, is kept for the applying sweep a person runs, and the
	// last check before its removal reads them once more. Unset, as in the
	// sweep a person runs, the row names them and the worktree is removed.
	keepIgnored bool

	// forgeChecked says Forge was pinned to origin's repository already
	// (pinnedForge), forgeKey is that repository's key, and forgeUnmatched
	// is why it could not be, when Forge was given and now is nil.
	forgeChecked   bool
	forgeKey       string
	forgeUnmatched string
}

// runDry judges every worktree of the repository at repoRoot and every entry
// in its lane, and writes nothing. A store refusal, a failure to list the
// worktrees, or a failure to read what the build loop holds is returned as
// the error, and the result carries it as its Refusal, so ExitCode reads
// ExitRefused. A run without Held or Cwd is refused the same way.
func runDry(ctx context.Context, repoRoot string, o Options) (Result, error) {
	res, _, err := judgeRepo(ctx, repoRoot, o, false)
	return res, err
}

// plan is what runApply acts on beside the result: the store's row behind
// each of the result's rows, in order, the salvage each save-patch row would
// write, by the row's index, the lane and judge that made them, and the
// options they were made with, the forge pinned.
type plan struct {
	srcs     []store.Row
	salvages map[int]salvage
	lane     store.Lane
	j        judge
	o        Options
}

// judgeRepo is runDry, also answering with the plan behind the result's rows,
// for runApply to act on. retain keeps each salvage's bytes for apply to
// write; a dry run keeps only each salvage's manifest and identity.
func judgeRepo(ctx context.Context, repoRoot string, o Options, retain bool) (Result, plan, error) {
	fail := func(err error) (Result, plan, error) {
		res, err := refused(err)
		return res, plan{}, err
	}
	if err := o.required(); err != nil {
		return fail(err)
	}
	if o.ignored == nil {
		o.ignored = store.IgnoredForRemoval
	}
	o = o.pinnedForge(ctx, repoRoot)
	listing, err := store.List(repoRoot, store.ListOptions{Forge: o.Forge})
	if err != nil {
		return fail(err)
	}
	held, err := o.Held(repoRoot)
	if err != nil {
		return fail(fmt.Errorf("reading what the build loop holds: %w", err))
	}
	// The lane List derived, again: its Listing carries only the redacted
	// path, and a salvage is written into the lane's notes archive.
	lane, err := store.LaneFor(repoRoot)
	if err != nil {
		return fail(err)
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
	if o.forgeUnmatched != "" {
		// The store asked no forge, and would say none was given.
		listing.ForgeNote = clean(o.forgeUnmatched + "; merged is judged from git alone")
	}
	res := Result{
		Lane:                    listing.Lane,
		Rows:                    make([]Row, 0, len(listing.Worktrees)),
		ProcessCheckUnavailable: clean(p.ProcessCheckUnavailable()),
		ReservationsUnread:      clean(p.ReservationsUnread()),
		RunsUnread:              clean(p.runsUnread),
		ForgeNote:               listing.ForgeNote,
	}
	pl := plan{srcs: listing.Worktrees, salvages: map[int]salvage{}, lane: lane, j: j, o: o}
	sv := &salvager{ctx: ctx, repoRoot: repoRoot, home: home, lane: lane, retain: retain}
	for i, r := range listing.Worktrees {
		row := j.row(r)
		if row.Action == ActionRemove {
			row = withIgnored(row, r.Abs, o.ignored, o.keepIgnored)
		}
		if row.Action == ActionSavePatch {
			if s, ok := sv.plan(r, &row); ok {
				pl.salvages[i] = s
			}
		}
		res.Rows = append(res.Rows, row)
	}
	return res, pl, nil
}

// withIgnored is row, a worktree judged spent, with what its removal deletes
// unarchived (iss-2611161722120909): its ignored paths outside the local
// tier, listed by list, or that they could not be listed. With nothing to
// name, row is as it was. Otherwise row carries them (Row.Ignored), and:
//
//   - keep set, as the sweep abcd runs on its own sets it: the worktree is
//     kept, its row saying how many and the first of them, or that they could
//     not be listed, and that it is left for the applying sweep a person runs,
//     whose rows name them first (keptForIgnored);
//   - keep unset, as in the sweep a person runs: the row says so as
//     information and the worktree is still removed, since the person, who
//     reads the row, decides.
func withIgnored(row Row, abs string, list func(string) (store.IgnoredState, error), keep bool) Row {
	st := ignoredFor(abs, list)
	if st == nil {
		return row
	}
	row.Ignored = st
	if keep {
		return keptForIgnored(row, *st)
	}
	strength := Proven
	if st.Unread {
		strength = Hint
	}
	going := ignoredGoing(*st)
	row.Reason = clean(row.Reason + "; " + going)
	row.Evidence = append(row.Evidence, Evidence{Kind: KindIgnored, Strength: strength, Detail: clean(going)})
	return row
}

// ignoredFor is list's answer for the worktree at abs as a row carries it:
// nil when nothing outside the local tier is ignored, and a listing that
// fails, or reads a wholly ignored .abcd/ it cannot read past, marked unread
// with why.
func ignoredFor(abs string, list func(string) (store.IgnoredState, error)) *store.IgnoredState {
	got, err := list(abs)
	switch {
	case err != nil:
		return &store.IgnoredState{Paths: []string{}, Unread: true, Why: clean(err.Error())}
	case got.Unread:
		got.Why = "a wholly ignored .abcd/ could not be read past"
		return &got
	case got.Count > 0:
		return &got
	}
	return nil
}

// ignoredNames is st's first paths, and how many more there are.
func ignoredNames(st store.IgnoredState) string {
	names := strings.Join(st.Paths, ", ")
	if st.Count > len(st.Paths) {
		names += fmt.Sprintf(", and %d more", st.Count-len(st.Paths))
	}
	return names
}

// ignoredGoing is what a person's sweep says on a remove row of the ignored
// files that go with the worktree, as information.
func ignoredGoing(st store.IgnoredState) string {
	if st.Unread {
		return "its ignored files could not be listed (" + st.Why + "), and any go with it, unarchived"
	}
	return fmt.Sprintf("%d ignored path(s) outside its local tier go with it, unarchived: %s; copy any you need first", st.Count, ignoredNames(st))
}

// keptForIgnored is row, a worktree judged spent whose removal would delete
// st unarchived, as the sweep abcd runs on its own judges it: kept, its row
// naming how many and the first of them, or that they could not be listed,
// and saying it is left for the applying sweep a person runs, whose rows
// name them first.
func keptForIgnored(row Row, st store.IgnoredState) Row {
	const left = "left for `abcd sweep --apply`: "
	why, strength := ignoredKept(st)
	row.Action = ActionKeep
	row.Reason = clean(left + why + "; " + row.Reason)
	row.Evidence = append(row.Evidence, Evidence{Kind: KindIgnored, Strength: strength, Detail: clean(why)})
	return row
}

// ignoredKept is why the sweep abcd runs on its own keeps a worktree whose
// removal would delete st, and how strong the evidence is.
func ignoredKept(st store.IgnoredState) (string, Strength) {
	if st.Unread {
		return "its ignored files could not be listed (" + st.Why + "), and a removal would delete any unarchived, so abcd does not remove it on its own", Hint
	}
	return fmt.Sprintf("it holds %d ignored path(s) outside its local tier (%s), which a removal deletes unarchived, so abcd does not remove it on its own; copy any you need first", st.Count, ignoredNames(st)), Proven
}

// required is why the options cannot run a sweep, nil when they can: Held
// and Cwd are required, as the store's prune requires them.
func (o Options) required() error {
	if o.Held == nil {
		return errors.New("sweep: a dry run needs a reader of what the build loop holds")
	}
	if o.Cwd == "" {
		return errors.New("sweep: a dry run needs its own working directory, to keep the worktree holding it")
	}
	return nil
}

// Run is one whole sweep of the repository at repoRoot, as `abcd sweep`
// runs it: a dry run, or with apply an applied one. It fetches first
// (criterion 11): origin's branches, pruned, so the half that judges them
// reads origin as it is, then the reservations (claim.Fetch). It then reads
// into the probe the reservations as fetched and the build loop's runs,
// unless the options already carry them. A checkout with no origin fetches
// nothing and runs on: no fetch of the reservations is then stamped, so every
// worktree is held, and origin's half has no branch to judge. A fetch that
// fails refuses the run before anything is judged, since every judgement
// after it would read origin as it was at some earlier fetch.
func Run(ctx context.Context, repoRoot string, o Options, apply bool) (Result, error) {
	o, err := prepare(ctx, repoRoot, o)
	if err != nil {
		return refused(err)
	}
	return runSweep(ctx, repoRoot, o, apply)
}

// prepare is what Run does before any half judges: the options checked, the
// fetch, and the probe given the reservations as fetched and the build loop's
// runs, unless the options already carry them. An error refuses the run before
// anything is judged.
func prepare(ctx context.Context, repoRoot string, o Options) (Options, error) {
	if err := o.required(); err != nil {
		return o, err
	}
	if err := ctx.Err(); err != nil {
		return o, fmt.Errorf("sweep: stopped before the fetch: %w", err)
	}
	fetched, err := fetchFirst(ctx, repoRoot)
	if err != nil {
		return o, err
	}
	if fetched && o.Probe.Reservations.FetchedAt.IsZero() {
		snap, err := claim.View(repoRoot, o.probeNow())
		if err != nil {
			return o, fmt.Errorf("sweep: reading the reservations after the fetch: %w; nothing was judged", err)
		}
		o.Probe.Reservations = snap
	}
	if o.Probe.Runs == nil && o.Probe.RunsUnread == "" {
		runs, err := loop.Runs(repoRoot)
		if err != nil {
			o.Probe.RunsUnread = err.Error()
		}
		o.Probe.Runs = runs
	}
	return o, nil
}

// fetchFirst is the sweep's fetch: origin's branches with pruning, then the
// reservations and origin's default branch (claim.FetchContext). Both are
// bounded by ctx as well as by gitutil.NetTimeout, through the hooked
// runner Net uses, so the sweep abcd runs on its own ends within its own
// bound. fetched is false, with no error, when the checkout has no remote
// named origin.
func fetchFirst(ctx context.Context, repoRoot string) (fetched bool, err error) {
	if _, err := gitutil.Run(repoRoot, "remote", "get-url", claim.Remote); err != nil {
		return false, nil
	}
	if _, err := gitutil.NetInputContext(ctx, repoRoot, "", "fetch", "--prune", "--quiet", claim.Remote); err != nil {
		return false, fmt.Errorf("sweep: fetching %s: %w; nothing was judged", claim.Remote, err)
	}
	if _, err := claim.FetchContext(ctx, repoRoot, claim.Options{}); err != nil {
		if errors.Is(err, claim.ErrNoOrigin) {
			return false, nil
		}
		return false, fmt.Errorf("sweep: fetching the reservations: %w; nothing was judged", err)
	}
	return true, nil
}

// probeNow is the probe's clock read once, as newProbe reads it.
func (o Options) probeNow() time.Time {
	if o.Probe.Now != nil {
		return o.Probe.Now()
	}
	return time.Now()
}

// runSweep is one whole sweep of the repository at repoRoot, a dry run or,
// with apply, an applied one: its halves in the order of the spec's Rows,
// the worktrees with the strays and the embargo store (runDry or runApply),
// the local branches (sweepBranches), the branches on origin (sweepRemote),
// the reservations (sweepReservations) and the inbox (sweepInbox), each
// half's rows after the one before. Options without Held or Cwd refuse the
// whole run before any half runs.
//
// Every other half runs whatever the one before it answered: each judges its
// own items, keeps what its own reads cannot clear, and refuses or faults on
// its own, and once the run's context has ended each later half refuses
// itself before it reads anything. The result's Refusal is every half's
// refusal or fault, each distinct one once, named by its half, in the order
// the halves ran, and the error carries the same text and wraps every
// half's error, so ExitCode reads ExitRefused when any half refused or
// faulted. Each failed-closed check is said once: the first half's note for
// each, and every distinct forge note.
func runSweep(ctx context.Context, repoRoot string, o Options, apply bool) (Result, error) {
	return runHalves(ctx, repoRoot, o, apply, false)
}

// runHalves is runSweep, with automatic the sweep abcd runs on its own
// (auto.go): the worktrees, the local branches and the branches on origin
// alone, so it dispatches no reservation workflow and files no inbox report.
func runHalves(ctx context.Context, repoRoot string, o Options, apply, automatic bool) (Result, error) {
	if err := o.required(); err != nil {
		return refused(err)
	}
	// One pin serves every half: origin's repository is asked of the forge
	// once a run.
	o = o.pinnedForge(ctx, repoRoot)
	halves := []struct {
		name string
		run  func() (Result, error)
	}{
		{"worktrees", func() (Result, error) {
			if apply {
				return runApply(ctx, repoRoot, o)
			}
			return runDry(ctx, repoRoot, o)
		}},
		{"local branches", func() (Result, error) { return sweepBranches(ctx, repoRoot, apply, o.ours) }},
		{"origin's branches", func() (Result, error) { return sweepRemote(ctx, repoRoot, o, apply) }},
		{"reservations", func() (Result, error) { return sweepReservations(ctx, repoRoot, o, apply) }},
		{"inbox", func() (Result, error) { return sweepInbox(ctx, repoRoot, o, apply) }},
	}
	if automatic {
		// The reservations' half dispatches a workflow and the inbox's files
		// records: both stay with the sweep a person runs.
		halves = slices.DeleteFunc(halves, func(h struct {
			name string
			run  func() (Result, error)
		}) bool {
			return h.name == "reservations" || h.name == "inbox"
		})
	}
	out := Result{Rows: []Row{}, Applied: apply}
	var (
		errs     []error
		refusals []string
	)
	for _, half := range halves {
		res, err := half.run()
		out.Rows = append(out.Rows, res.Rows...)
		out.Lane = firstNonEmpty(out.Lane, res.Lane)
		out.ProcessCheckUnavailable = firstNonEmpty(out.ProcessCheckUnavailable, res.ProcessCheckUnavailable)
		out.ReservationsUnread = firstNonEmpty(out.ReservationsUnread, res.ReservationsUnread)
		out.RunsUnread = firstNonEmpty(out.RunsUnread, res.RunsUnread)
		out.Workflow = firstNonEmpty(out.Workflow, res.Workflow)
		if res.ForgeNote != "" && !strings.Contains(out.ForgeNote, res.ForgeNote) {
			out.ForgeNote = strings.TrimPrefix(out.ForgeNote+"; "+res.ForgeNote, "; ")
		}
		if err != nil {
			errs = append(errs, err)
			if r := half.name + ": " + firstNonEmpty(res.Refusal, clean(err.Error())); !slices.Contains(refusals, r) {
				refusals = append(refusals, r)
			}
		}
	}
	if len(errs) == 0 {
		return out, nil
	}
	out.Refusal = strings.Join(refusals, "; ")
	return out, &refusal{msg: out.Refusal, cause: errors.Join(errs...)}
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
		return keep(dirtyToSave, ev...)
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
