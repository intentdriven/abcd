package matrix

// launch.go launches a runner cell's copies (spc-2610161949559050 step 11,
// acceptance criteria 9 and 10; dcn-2611251746554510). A consented run hands
// every runner cell's copy off awaiting its runner (handoff.go); the launch
// dispatches the cell's runner directly in the copy's clone, in the runners'
// cell mode, on the blind brief a host cell's copy is made from, so within a
// task the cells differ only by model and harness. It does not go through the
// build loop's trial mode, whose lane worktree sits outside the clone that the
// compliance check, the review and the check collection read.
//
// The trust boundary: the launch starts a model's harness on this machine,
// in a clone of the frozen commit, writing outside every working tree.
//
//   - Each try is one launch through runner.Dispatcher, whose launcher starts
//     the harness as the leader of its own process group and, at its bound or
//     when the launch's context ends, kills that group whole through the pid
//     its handle holds: never a pattern, never another process.
//   - The cell's route is placed as the implementer's for this launch alone
//     (runner.Config.Override, the person's own flag layer), and the cell's
//     source and pin are read again and held to the run's record, so a copy
//     never runs on a pin the person did not consent to.
//   - Nothing stands in for the cell's runner: a runner that fails, refuses or
//     answers unparsably is the cell's failure, with no fallback receipt and no
//     host dispatch; a provider down before the copy began is retried on the
//     same pin and recorded as not started, never scored (runCopy).
//   - A host cell's copy and a refused cell's copy are never launched, and a
//     provider-bound clone passes the launch bundle's scan again just before
//     its runner starts.
//   - A metered copy is launched only while the spending cap's remainder, less
//     what the copies in flight may spend, covers its declared cost; the
//     remainder is reckoned from the run's records (newSpendBudget), the
//     copies earlier launches made and left without a trial included, and one
//     launch per run runs at a time, holding the run's launch lock.
//   - The brief and the receipt sit in a third store, the hand-off store
//     (handoffStore), outside the clone, the clone store and the lab state,
//     each copy's in a folder named by its alias; the compliance check treats a
//     path in that store outside the copy's own folder as a read outside its
//     copy (own_copy_only).
//   - Each copy is claimed once, by a file created exclusively in its folder,
//     so two launches never run one copy, and its launch is recorded beside
//     its hand-off, once as it begins and once as it ends (launchRel), never
//     over it; a stop ends the launch as the run's
//     error, never as the copy's score, and leaves the copy interrupted, never
//     launched again, since its clone holds what the runner left.
//   - What the copy hands back (its receipt or its answer) is untrusted input,
//     read through an os.Root of its folder within a bound, and a trial is
//     written through the same compliance check and secret screen as a host
//     copy's.
//
// The runners' live proof is still owed, so this is tested against a fake
// runner with no network, and the report says so among its limitations.

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/core/lab"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// handoffDirName is the folder in the account's abcd home every repository's
// hand-off store sits in.
const handoffDirName = "matrix-handoff"

// handoffStore is where the repository with root commit rootSHA keeps its
// launched copies' briefs and receipts: under the account's abcd home, beside
// the clone store and the labs and inside neither.
func handoffStore(home, rootSHA string) string { return abcdhome.Path(home, handoffDirName, rootSHA) }

// handoffDir is the folder of run's copy made under alias, in store: the run's
// id and the alias, and nothing of the copy's task or cell.
func handoffDir(store, run, alias string) string { return filepath.Join(store, run, alias) }

// The files of a copy's folder in the hand-off store.
const (
	launchBriefName   = "brief.md"
	launchClaimName   = "launch.claim"
	launchReceiptName = "receipt.json"
	launchDocName     = "document.md"
)

// The run's own files in the hand-off store: the lock one launch of the run
// holds for as long as it runs, and who holds it.
const (
	launchLockName   = "launch.lock"
	launchHolderName = "launch.holder"
)

// launchLockWait is how long a launch waits for its run's lock before it is
// refused: long enough for a reader that only looked at the lock to let it go.
const launchLockWait = time.Second

// ErrLaunchHeld is a launch refused because another launch of its run holds
// the run's lock: one launch per run at a time (exit 3).
var ErrLaunchHeld = errors.New("another launch of the run holds it")

// launchLockPath is run's launch lock in the hand-off store.
func launchLockPath(store, run string) string { return filepath.Join(store, run, launchLockName) }

// runLockHeld reports whether a launch holds the lock at lock now: free where
// its folder was never made or the lock is taken and given back at once, held
// where it is taken, and held where it cannot be judged.
func runLockHeld(lock string) bool {
	if !fsutil.IsRealDir(filepath.Dir(lock)) {
		return false
	}
	err := fsutil.WithFileLock(lock, 0, func() error { return nil })
	return err != nil && !errors.Is(err, fs.ErrNotExist)
}

// receiptName is the file a copy of kind hands its answer back in: a writing
// copy's document, and every other copy's receipt or findings.
func receiptName(kind TaskKind) string {
	if kind == KindWriting {
		return launchDocName
	}
	return launchReceiptName
}

// launchTools are what a launched copy's runner grants without a prompt: the
// tools the build loop's implementer is granted.
var launchTools = []string{"Read", "Edit", "Write", "Bash", "Grep", "Glob"}

// maxLaunchReceiptBytes bounds what a launched copy hands back: a writing
// copy's document or a check copy's findings, or a code copy's receipt.
const maxLaunchReceiptBytes = maxCheckAnswerBytes

// outageWaitStep is the wait before a copy's second try on its pin while its
// provider is down; each later try waits as many steps as tries before it.
const outageWaitStep = 30 * time.Second

// LaunchedCopy is how one copy's launch came out: its hand-off's state once
// the launch ended, its trial's status where one is written, and why.
type LaunchedCopy struct {
	Key   TrialKey     `json:"key"`
	State HandOffState `json:"state"`
	// Status is the copy's trial's status, "" where no trial is written: a
	// code copy awaiting its hidden tests' result, a copy the launch did not
	// start, and one it did not finish.
	Status TrialStatus `json:"status,omitempty"`
	Reason string      `json:"reason"`
}

// LaunchResult is one launch of a run's runner cells' copies.
type LaunchResult struct {
	Run    string         `json:"run"`
	Copies []LaunchedCopy `json:"copies"`
}

// cellDispatch runs one try of a launched copy's request in its cell: the
// runner's dispatch in cell mode, its answer checked by check.Validate and its
// transcript kept by check.
type cellDispatch func(ctx context.Context, req runner.Request, check *copyCheck) (runner.Outcome, error)

// copyLauncher launches a run's runner cells' copies from the checkout at
// repoRoot, under the account's home at home.
type copyLauncher struct {
	repoRoot, home, labID string
	// configure is cell pin's dispatch for one copy; it refuses a cell that
	// cannot run here as its run recorded it. It is called once for each cell
	// before anything is launched, and once for each copy.
	configure func(pin CellPin) (cellDispatch, error)
	// wait is the wait before try n+1 of a copy whose provider is down.
	wait func(ctx context.Context, n int) error
	// now is the launch's clock.
	now func() time.Time
}

// LaunchCopies launches the copies of run, in the lab labID of the checkout at
// repoRoot, that wait for their cell's runner, at most as many at once as the
// run's declared concurrency, each in its own clone under the account's home
// at home; with only, that one copy alone. It refuses, launching nothing: a
// copy the run did not hand off, a host cell's copy, a refused cell's, one
// already launched, and a cell whose route or pin cannot run here as the run
// recorded it. A copy the spending cap's remainder does not cover is left
// waiting and named. A stop (ctx ending) ends the launch with its error; the
// copies done before it keep their trials.
func LaunchCopies(ctx context.Context, repoRoot, home, labID, run string, only *TrialKey) (LaunchResult, error) {
	l := &copyLauncher{repoRoot: repoRoot, home: home, labID: labID, wait: outageWait, now: func() time.Time { return time.Now().UTC() }}
	return l.launch(ctx, run, only, func(rec RunRecord, d Declaration) func(CellPin) (cellDispatch, error) {
		return personsCells(repoRoot, d)
	})
}

// outageWait waits n steps, or until ctx ends.
func outageWait(ctx context.Context, n int) error {
	t := time.NewTimer(time.Duration(n) * outageWaitStep)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// personsCells is a cell's dispatch on this machine, as the person's own
// layers configure it: the cell's route is resolved again and held to the
// source and pin the run recorded, then placed as the implementer's route for
// this launch alone.
func personsCells(repoRoot string, d Declaration) func(CellPin) (cellDispatch, error) {
	return func(pin CellPin) (cellDispatch, error) {
		roots, _ := layered.RootsFor(repoRoot)
		var dc *roleCell
		for _, c := range d.cells() {
			if c.ID == pin.Cell {
				dc = &c
			}
		}
		if dc == nil {
			return nil, fmt.Errorf("the run declares no cell %s", pin.Cell)
		}
		api, err := oracle.LoadAPI(roots)
		if err != nil {
			return nil, err
		}
		check := func(fr oracle.FlagRoute) error {
			rc, err := runner.Load(roots)
			if err != nil {
				return err
			}
			fr.Agent = loopRole[Role(fr.Agent)]
			return rc.Override([]oracle.FlagRoute{fr})
		}
		pc, err := resolveCell(*dc, api, check)
		if err != nil {
			return nil, fmt.Errorf("cell %s cannot run here as the run recorded it: %w", pin.Cell, err)
		}
		if pc.Source != pin.Source || pc.Runner != pin.Runner || pc.Model != pin.Model ||
			!reflect.DeepEqual(pc.ProviderPin, pin.ProviderPin) || !reflect.DeepEqual(pc.LocalPin, pin.LocalPin) {
			return nil, fmt.Errorf("cell %s's source or pin on this machine is not the one the run recorded and the person consented to; "+
				"start a new run of the declaration to launch it on what is configured now", pin.Cell)
		}
		routes, err := oracle.ParseRunnerRoutes([]string{pin.Route}, []string{string(pin.Role)})
		if err != nil {
			return nil, declaredRoute(pin.Route, err)
		}
		fr := routes[0]
		fr.Agent = loopRole[pin.Role]
		cfg, err := runner.Load(roots)
		if err != nil {
			return nil, err
		}
		if err := cfg.Override([]oracle.FlagRoute{fr}); err != nil {
			return nil, err
		}
		return dispatchOn(cfg), nil
	}
}

// dispatchOn is the cell dispatch through the runner configuration cfg: the
// runners' cell mode, with no host session, so nothing stands in for the
// cell's runner and no fallback receipt is written.
func dispatchOn(cfg *runner.Config) cellDispatch {
	return func(ctx context.Context, req runner.Request, check *copyCheck) (runner.Outcome, error) {
		d := runner.Dispatcher{Config: cfg, Cell: true, Validate: check.validate, Transcripts: check,
			Record: func(runner.FallbackReceipt) error {
				return errors.New("a launched copy's cell falls back on nothing, so no fallback receipt is written")
			}}
		return d.Dispatch(ctx, req)
	}
}

// launchPick is one copy the launch makes.
type launchPick struct {
	h     HandOff
	pin   CellPin
	task  Task
	alias string
}

// launch is LaunchCopies with the cells' dispatch made by cells, run while
// it holds its run's launch lock: one launch per run at a time, so what the
// copies in flight may spend and how many run at once are reckoned by one
// process. A second is refused (ErrLaunchHeld), naming who holds the lock.
func (l *copyLauncher) launch(ctx context.Context, run string, only *TrialKey, cells func(RunRecord, Declaration) func(CellPin) (cellDispatch, error)) (LaunchResult, error) {
	if !runIDRe.MatchString(run) {
		return LaunchResult{}, fmt.Errorf("%q is not a run id", clip(run))
	}
	if only != nil {
		if err := only.validate(); err != nil {
			return LaunchResult{}, err
		}
	}
	proc, err := lab.OpenProcedure(l.repoRoot, l.labID)
	if err != nil {
		return LaunchResult{}, err
	}
	rootSHA := proc.Entry().RootSHA
	proc.Close()
	store := handoffStore(l.home, rootSHA)
	if err := runFolder(store, run); err != nil {
		return LaunchResult{}, err
	}
	holder := filepath.Join(store, run, launchHolderName)
	var out LaunchResult
	locked := false
	err = fsutil.WithFileLock(launchLockPath(store, run), launchLockWait, func() error {
		locked = true
		who := fmt.Sprintf("pid %d, since %s", os.Getpid(), l.now().Format(time.RFC3339))
		if err := fsutil.WriteFileAtomic(holder, []byte(who+"\n"), 0o600); err != nil {
			return err
		}
		defer os.Remove(holder)
		var lerr error
		out, lerr = l.launchLocked(ctx, run, rootSHA, only, cells)
		return lerr
	})
	if !locked && errors.Is(err, fsutil.ErrLockContention) {
		who := "who holds it is not recorded"
		if raw, rerr := fsutil.ReadGuarded(holder, 256); rerr == nil {
			who = "held by " + termsafe.SanitizeLine(strings.TrimSpace(string(raw)))
		}
		return LaunchResult{}, fmt.Errorf("%w (%s): a run is launched by one launch at a time, so wait for that one to end", ErrLaunchHeld, who)
	}
	return out, err
}

// listingDenied is the mode a run's folder in the hand-off store and in the
// clone store holds while its copies run: its owner may make, remove and pass
// through the entries it names, and cannot list them. A copy runs as the same
// account, so 0711 would not do: it keeps the owner's read.
const listingDenied fs.FileMode = 0o300

// cloneLstated is called once removeClone has read what dir is, before it
// opens it: a test replaces it to swap the clone in that window.
var cloneLstated = func(dir string) {}

// runFolders is run's folder in the hand-off store and in the clone store,
// which a launch closes to listing while its copies run.
func runFolders(home, rootSHA, run string) []string {
	return []string{filepath.Join(handoffStore(home, rootSHA), run), filepath.Join(cloneStore(home, rootSHA), run)}
}

// reopenListing gives each of dirs that exists, a real directory, and holds
// no read for its owner the mode 0700 back: a launch killed outright (SIGKILL,
// a power loss) runs no deferred restore, and leaves the run's folders closed
// to listing (denyListing). Its caller holds the run's launch lock, which
// proves no launch of the run is at work.
func reopenListing(dirs ...string) error {
	var errs []error
	for _, d := range dirs {
		fi, err := os.Lstat(d)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			continue
		case err != nil:
			errs = append(errs, err)
		case fi.IsDir() && fi.Mode().Perm()&0o400 == 0:
			if err := os.Chmod(d, 0o700); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}

// reopenIdle is reopenListing of run's folders, for a reader that holds no
// lock of its own: it takes the run's launch lock for that moment alone, and
// where a launch holds it leaves the folders as they are, closed for that
// launch's copies; what the reader cannot list it then reads as such
// (globWalker.read), never as a folder that holds nothing.
func reopenIdle(home, rootSHA, run string) error {
	lock := launchLockPath(handoffStore(home, rootSHA), run)
	if !fsutil.IsRealDir(filepath.Dir(lock)) {
		return nil
	}
	err := fsutil.WithFileLock(lock, 0, func() error { return reopenListing(runFolders(home, rootSHA, run)...) })
	if errors.Is(err, fsutil.ErrLockContention) {
		return nil
	}
	return err
}

// denyListing sets each of dirs that exists, a real directory, to
// listingDenied, and returns what gives each the mode it held, or 0700 where
// it held no read for its owner. Nothing lists these folders while they are
// closed: review, judging and the report read the copies from the run's
// records, and own_copy_only's walk of a pattern reads them as a launched
// copy's own shell did, matching nothing in them, and as unexamined for any
// other copy (complianceInputs.RanClosed).
func denyListing(dirs ...string) (func() error, error) {
	type held struct {
		dir  string
		mode fs.FileMode
	}
	var done []held
	restore := func() error {
		var errs []error
		for _, h := range done {
			if err := os.Chmod(h.dir, h.mode); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	}
	for _, d := range dirs {
		fi, err := os.Lstat(d)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err == nil && !fi.IsDir() {
			err = fmt.Errorf("%s is not a real directory", filepath.Base(d))
		}
		if err != nil {
			return nil, errors.Join(err, restore())
		}
		mode := fi.Mode().Perm()
		if mode&0o400 == 0 {
			mode = 0o700
		}
		if err := os.Chmod(d, listingDenied); err != nil {
			return nil, errors.Join(err, restore())
		}
		done = append(done, held{dir: d, mode: mode})
	}
	return restore, nil
}

// removeClone deletes the copy's clone dir. Its run's folder may be closed to
// listing (denyListing), and os.RemoveAll opens the folder that holds what it
// deletes, so the clone is opened once as a root, held to be the folder its
// path named when it was read (a link swapped into its place is refused), and
// its entries are deleted through that root, so no entry is found again by its
// path; the emptied clone is then removed from its folder by name, which owner
// write and traverse allow. A clone that is not a real directory is removed as
// it is, never followed.
func removeClone(dir string) error {
	fi, err := os.Lstat(dir)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil
	case err != nil:
		return err
	case !fi.IsDir():
		return os.Remove(dir)
	}
	cloneLstated(dir)
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer root.Close()
	if at, err := root.Stat("."); err != nil {
		return err
	} else if !os.SameFile(fi, at) {
		return fmt.Errorf("the clone %s was replaced while it was deleted, so nothing in it was deleted", filepath.Base(dir))
	}
	f, err := root.Open(".")
	if err != nil {
		return err
	}
	ents, err := f.ReadDir(-1)
	f.Close()
	if err != nil {
		return err
	}
	var errs []error
	for _, e := range ents {
		errs = append(errs, root.RemoveAll(e.Name()))
	}
	if err := errors.Join(errs...); err != nil {
		return err
	}
	return os.Remove(dir)
}

// runFolder makes run's folder in the hand-off store, a real directory
// outside every working tree.
func runFolder(store, run string) error {
	if !filepath.IsAbs(store) || !runIDRe.MatchString(run) {
		return errors.New("a run's hand-off folder is named by an absolute store and a run id")
	}
	if err := os.MkdirAll(store, 0o700); err != nil {
		return fmt.Errorf("the hand-off store could not be made: %w", err)
	}
	if err := outsideWorkingTrees(store); err != nil {
		return err
	}
	if err := fsutil.EnsureRealDirAll(store, run, 0o700); err != nil {
		return fmt.Errorf("the run's hand-off folder could not be made: %w", err)
	}
	return nil
}

// launchLocked is launch once it holds its run's lock. It first gives back
// the modes a launch killed outright left closed (reopenListing), before it
// reads anything, whether or not it then finds a copy to launch. While its
// copies run, the run's folders in the hand-off store and the clone store,
// which hold every copy's folder, are closed to listing (denyListing), and
// their modes are given back however the launch ends short of a kill: a copy
// that keeps their modes finds no sibling's folder by listing, and names one
// only by its alias, which own_copy_only reads.
func (l *copyLauncher) launchLocked(ctx context.Context, run, rootSHA string, only *TrialKey, cells func(RunRecord, Declaration) func(CellPin) (cellDispatch, error)) (res LaunchResult, retErr error) {
	folders := runFolders(l.home, rootSHA, run)
	if err := reopenListing(folders...); err != nil {
		return LaunchResult{}, fmt.Errorf("the run's folders, left closed to listing by a launch that was killed, could not be opened again: %w", err)
	}
	picks, rec, d, trials, handed, err := l.pick(run, only)
	if err != nil {
		return LaunchResult{}, err
	}
	if rec.RootSHA != rootSHA {
		return LaunchResult{}, fmt.Errorf("the run %s is keyed on another repository's root commit than its lab", run)
	}
	out := LaunchResult{Run: run, Copies: []LaunchedCopy{}}
	if len(picks) == 0 {
		return out, nil
	}
	if l.configure == nil {
		l.configure = cells(rec, d)
	}
	checked := map[string]bool{}
	for _, p := range picks {
		if checked[p.pin.Cell] {
			continue
		}
		if _, err := l.configure(p.pin); err != nil {
			return LaunchResult{}, err
		}
		checked[p.pin.Cell] = true
	}
	restore, err := denyListing(folders...)
	if err != nil {
		return LaunchResult{}, fmt.Errorf("the run's folders could not be closed to listing while its copies run: %w", err)
	}
	defer func() {
		if err := restore(); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("the run's folders could not be opened to listing again: %w", err))
		}
	}()
	budget := newSpendBudget(rec, d, trials, handed)
	slots := make(chan struct{}, max(1, rec.Concurrency))
	results := make([]*LaunchedCopy, len(picks))
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		runErr  error
		claimed bool
	)
	fail := func(err error) {
		mu.Lock()
		defer mu.Unlock()
		runErr = errors.Join(runErr, err)
	}
	for i, p := range picks {
		select {
		case slots <- struct{}{}:
		case <-ctx.Done():
		}
		if ctx.Err() != nil {
			break
		}
		mu.Lock()
		stopped := runErr != nil
		mu.Unlock()
		if stopped {
			<-slots
			break
		}
		need, err := budget.reserve(p.pin)
		if err != nil {
			<-slots
			results[i] = &LaunchedCopy{Key: p.h.Key, State: p.h.State, Reason: termsafe.SanitizeLine(err.Error())}
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-slots }()
			lc, spent, took, err := l.one(ctx, rec, p)
			budget.settle(p.h.Key, p.pin, need, spent, took && lc.Status != StatusNotStarted)
			mu.Lock()
			results[i] = &lc
			claimed = claimed || took
			mu.Unlock()
			if err != nil {
				fail(fmt.Errorf("copy %s: %w", p.h.Key, err))
			}
		}()
	}
	wg.Wait()
	for _, r := range results {
		if r != nil {
			out.Copies = append(out.Copies, *r)
		}
	}
	if err := errors.Join(runErr, ctx.Err()); err != nil {
		err = fmt.Errorf("the launch ended before every copy was made: %w", err)
		if claimed {
			return out, afterWrite(err)
		}
		return out, err
	}
	return out, nil
}

// pick reads the run and chooses the copies a launch makes: every copy of a
// runner cell awaiting its runner, in key order, or, with only, that one,
// refused where it is not one to launch.
func (l *copyLauncher) pick(run string, only *TrialKey) ([]launchPick, RunRecord, Declaration, []TrialRecord, []HandOff, error) {
	fail := func(err error) ([]launchPick, RunRecord, Declaration, []TrialRecord, []HandOff, error) {
		return nil, RunRecord{}, Declaration{}, nil, nil, err
	}
	proc, err := lab.OpenProcedure(l.repoRoot, l.labID)
	if err != nil {
		return fail(err)
	}
	defer proc.Close()
	rec, d, err := readRun(proc, run)
	if err != nil {
		return fail(err)
	}
	handed, err := readHandOffs(proc, run)
	if err != nil {
		return fail(err)
	}
	trials, err := readTrials(proc, run, run+"/trials")
	if err != nil {
		return fail(err)
	}
	tried := map[TrialKey]bool{}
	for _, t := range trials {
		tried[t.Key] = true
	}
	var picks []launchPick
	found := false
	for _, h := range handed {
		if only != nil && h.Key != *only {
			continue
		}
		found = true
		task, pin, err := taskAndPin(rec, d, h.Key)
		if err != nil {
			return fail(err)
		}
		var why string
		switch {
		case pin.Source == SourceHost || h.State == HandOffWaiting:
			why = fmt.Sprintf("copy %s is a host cell's: the host makes it in its clone and hands it back with abcd lab matrix copy; a launch starts only a runner cell's copy", h.Key)
		case h.State == HandOffRefused:
			why = fmt.Sprintf("copy %s's cell was refused before it started (%s), so it is never launched", h.Key, h.Reason)
		case h.State != HandOffAwaitingRunner || tried[h.Key]:
			why = fmt.Sprintf("copy %s is already launched (%s); a copy is launched once", h.Key, h.State)
		}
		if why != "" {
			if only != nil {
				return fail(errors.New(why))
			}
			continue
		}
		alias, err := readAlias(proc, run, h.Key)
		if err != nil {
			return fail(err)
		}
		picks = append(picks, launchPick{h: h, pin: pin, task: task, alias: alias})
	}
	if only != nil && !found {
		return fail(fmt.Errorf("the run %s handed off no copy %s", run, only))
	}
	return picks, rec, d, trials, handed, nil
}

// one launches one copy, and reports how it came out, what it spent (nil
// where nothing was reported) and whether it claimed the copy. An error is
// the run's, never the copy's score: a stop, a dispatch the runner refused
// whole, or a record that could not be written.
func (l *copyLauncher) one(ctx context.Context, rec RunRecord, p launchPick) (LaunchedCopy, *oracle.Cost, bool, error) {
	key, h := p.h.Key, p.h
	lc := LaunchedCopy{Key: key, State: h.State}
	store := cloneStore(l.home, rec.RootSHA)
	if want := cloneDir(store, rec.RunID, p.alias); h.Clone != want || !fsutil.IsRealDir(want) {
		lc.Reason = "the copy's hand-off does not name its clone in the clone store, so it is not launched"
		return lc, nil, false, nil
	}
	dir, root, err := claimCopy(handoffStore(l.home, rec.RootSHA), rec.RunID, p.alias)
	if errors.Is(err, fs.ErrExist) {
		lc.Reason = "another launch claimed the copy first; a copy is launched once"
		return lc, nil, false, nil
	}
	if err != nil {
		return lc, nil, false, err
	}
	defer root.Close()
	setState := func(st HandOffState, reason string) error {
		next := h
		next.State, next.Reason = st, clipTo(termsafe.SanitizeLine(reason), maxReasonBytes)
		if err := l.writeHandOff(next); err != nil {
			return err
		}
		h = next
		lc.State, lc.Reason = h.State, h.Reason
		return nil
	}
	if err := setState(HandOffLaunching, "its launch began at "+l.now().Format(time.RFC3339)); err != nil {
		// Nothing of the launch is recorded, so the copy still waits for its
		// runner: its claim is given back for a later launch.
		if rerr := root.Remove(launchClaimName); rerr != nil {
			err = errors.Join(err, fmt.Errorf("and its claim could not be given back: %w", rerr))
		}
		return lc, nil, false, err
	}
	// The clone is scanned again just before its runner starts, as it was
	// when it was handed off: nothing in it is sent to a provider unscanned.
	if err := admitClone(l.repoRoot, h.Clone, p.pin); err != nil {
		if rerr := removeClone(h.Clone); rerr != nil {
			return lc, nil, true, errors.Join(err, rerr, setState(HandOffRefused, err.Error()+"; its clone could not be removed"))
		}
		h.Clone = ""
		return lc, nil, true, setState(HandOffRefused, err.Error())
	}
	brief := launchBrief(p.task)
	if err := fsutil.WriteFileAtomicInRoot(root, launchBriefName, []byte(brief), 0o600); err != nil {
		return lc, nil, true, errors.Join(err, setState(HandOffInterrupted, "its brief could not be written: "+err.Error()))
	}
	// Whatever sits where the answer goes was not written by this launch.
	if err := root.Remove(receiptName(p.task.Kind)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return lc, nil, true, errors.Join(err, setState(HandOffInterrupted, "what sat where its answer goes could not be removed: "+err.Error()))
	}
	dispatch, err := l.configure(p.pin)
	if err != nil {
		return lc, nil, true, errors.Join(err, setState(HandOffInterrupted, "its cell could not be dispatched: "+err.Error()))
	}
	req := runner.Request{Role: loopRole[RoleBuilder], Brief: filepath.Join(dir, launchBriefName),
		Receipt: filepath.Join(dir, receiptName(p.task.Kind)), Dir: h.Clone, Tools: slices.Clone(launchTools),
		Base: rec.FrozenCommit, SessionID: p.alias, Work: runner.LaunchWork{Run: rec.RunID}}
	check := &copyCheck{kind: p.task.Kind, root: root, name: receiptName(p.task.Kind)}
	var started, finished time.Time
	res, err := runCopy(ctx, p.pin, outageTries, l.wait, func(ctx context.Context, _ CellPin) (runner.Outcome, error) {
		check.reset()
		started = l.now()
		out, err := dispatch(ctx, req, check)
		finished = l.now()
		return out, err
	})
	if err != nil {
		// The run's error: the copy is not scored, and not launched again.
		werr := setState(HandOffInterrupted, "its launch was stopped before the copy was made ("+err.Error()+
			"); this is the run's error, not the copy's score, and the copy is not launched again, since its clone holds what its runner left")
		return lc, nil, true, errors.Join(err, werr)
	}
	t, spent := launchedTrial(rec, p, res, started, finished)
	back := launchBack{transcript: check.transcript()}
	if res.Made() {
		answer, err := fsutil.ReadGuardedInRoot(root, receiptName(p.task.Kind), maxLaunchReceiptBytes)
		if err != nil {
			return lc, spent, true, errors.Join(fmt.Errorf("its answer, read when its runner had answered, cannot be read again: %w", err),
				setState(HandOffInterrupted, "its answer could not be read once its runner had answered"))
		}
		back.answer = answer
		t = madeTrial(t, p.task, answer)
	}
	t, awaiting, err := l.finish(rec, p, t, back)
	if err != nil {
		return lc, spent, true, errors.Join(err, setState(HandOffInterrupted, "its trial could not be written: "+err.Error()))
	}
	if awaiting {
		h.Trial = &t
		return lc, spent, true, setState(HandOffLaunched, "its runner made the copy, which waits for its hidden tests' result: "+
			"the host hands it back with abcd lab matrix copy --receipt")
	}
	lc.Status = t.Status
	if err := setState(HandOffLaunched, t.Reason); err != nil {
		return lc, spent, true, afterWrite(err)
	}
	return lc, spent, true, nil
}

// writeHandOff records copy h's launch at the stage its state names, once,
// beside its hand-off (launchRel), which reads as h from then on.
func (l *copyLauncher) writeHandOff(h HandOff) error {
	stage := launchStage(h.State)
	if stage == "" {
		return fmt.Errorf("copy %s's launch leaves no copy %s", h.Key, h.State)
	}
	data, err := encodeCheck(h)
	if err != nil {
		return err
	}
	proc, err := lab.OpenProcedure(l.repoRoot, l.labID)
	if err != nil {
		return err
	}
	defer proc.Close()
	return proc.WriteState(launchRel(h.Run, h.Key, stage), data)
}

// claimCopy makes copy alias's folder of run in the hand-off store, a real
// directory outside every working tree, and claims the copy by creating its
// claim file exclusively in it: fs.ErrExist when another launch holds it. The
// folder is returned opened as a root, so nothing written or read through it
// leaves it.
func claimCopy(store, run, alias string) (string, *os.Root, error) {
	if !filepath.IsAbs(store) || !aliasRe.MatchString(alias) || !runIDRe.MatchString(run) {
		return "", nil, errors.New("a copy's hand-off folder is named by an absolute store, a run id and an alias")
	}
	if err := runFolder(store, run); err != nil {
		return "", nil, err
	}
	if err := fsutil.EnsureRealDirAll(store, run+"/"+alias, 0o700); err != nil {
		return "", nil, fmt.Errorf("the copy's hand-off folder could not be made: %w", err)
	}
	dir := handoffDir(store, run, alias)
	if !fsutil.IsRealDir(dir) {
		return "", nil, errors.New("the copy's hand-off folder is not a real directory")
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return "", nil, err
	}
	f, err := root.OpenFile(launchClaimName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		root.Close()
		return "", nil, err
	}
	if err := f.Close(); err != nil {
		root.Close()
		return "", nil, err
	}
	return dir, root, nil
}

// launchBrief is what a launched copy of task is asked: the host's blind brief
// (copyBrief), and where and in what shape it hands its answer back. It names
// no task id, cell, route or model.
func launchBrief(task Task) string {
	var out string
	switch task.Kind {
	case KindCode:
		out = "When you are done, write to the Receipt path exactly one JSON object: " +
			`{"schema_version": 1, "outcome": "made"} once the change is built, or ` +
			`{"schema_version": 1, "outcome": "failed", "reason": "<why, in one line>"} if you could not build it. ` +
			"Write nothing else there."
	case KindWriting:
		out = "Write the document's text, as Markdown and nothing else, to the Receipt path."
	default:
		out = "Write the check's findings, exactly as the check writes them and nothing else, to the Receipt path."
	}
	return copyBrief(task) + "\n\nThe one file you write outside the current directory is the Receipt the prompt names. " + out + "\n"
}

// launchReceipt is a launched code copy's own word on whether it built the
// change: nothing else, since a copy never scores itself.
type launchReceipt struct {
	SchemaVersion int    `json:"schema_version"`
	Outcome       string `json:"outcome"`
	Reason        string `json:"reason,omitempty"`
}

// parseLaunchReceipt reads a launched code copy's receipt strictly.
func parseLaunchReceipt(data []byte) (launchReceipt, error) {
	var rc launchReceipt
	if err := jsonstrict.Decode(data, &rc); err != nil {
		return launchReceipt{}, fmt.Errorf("the copy's receipt does not parse as {schema_version, outcome, reason}: %w", err)
	}
	switch {
	case rc.SchemaVersion != HandOffSchema:
		return launchReceipt{}, fmt.Errorf("the copy's receipt has schema_version %d; this abcd reads %d", rc.SchemaVersion, HandOffSchema)
	case rc.Outcome != OutcomeMade && rc.Outcome != OutcomeFailed:
		return launchReceipt{}, fmt.Errorf("the copy's receipt names the outcome %q; a copy is made or failed", clip(rc.Outcome))
	case rc.Outcome == OutcomeFailed && strings.TrimSpace(rc.Reason) == "":
		return launchReceipt{}, errors.New("the copy's receipt says it failed and names no reason")
	}
	return rc, line("reason", rc.Reason, maxReasonBytes)
}

// copyCheck is one launched copy's answer check and transcript store: the
// contract's validator the runner's answer passes, reading what the copy
// wrote at the Receipt path through its folder's root, and the transcript the
// runner's harness wrote, kept for the compliance check.
type copyCheck struct {
	kind TaskKind
	root *os.Root
	name string
	mu   sync.Mutex
	raw  []byte
}

// validate holds the copy's answer to its kind's shape: a code copy's receipt,
// a writing copy's readable document, a check copy's findings within their
// bound. One that fails is the runner's invalid answer, the cell's failure.
func (c *copyCheck) validate(_ string, _ runner.Request, _ runner.Answer) error {
	data, err := fsutil.ReadGuardedInRoot(c.root, c.name, maxLaunchReceiptBytes)
	if err != nil {
		return errors.New("the copy wrote no answer at the Receipt path, or one past its bound or not a plain file")
	}
	switch c.kind {
	case KindCode:
		_, err = parseLaunchReceipt(data)
	case KindWriting:
		err = readable("the copy's document", string(data))
	default:
		if len(strings.TrimSpace(string(data))) == 0 {
			err = errors.New("the copy's findings are empty")
		}
	}
	return err
}

// Store keeps the transcript of the try that ran, the last one's, whole: the
// runner bounds it already, and one past the compliance check's own bound is
// left unread there, never cut into a transcript that reads as whole.
func (c *copyCheck) Store(_ string, _ runner.Request, _ runner.Answer, raw []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.raw = slices.Clone(raw)
	return nil
}

func (c *copyCheck) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.raw = nil
}

func (c *copyCheck) transcript() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.raw
}

// launchBack is what a launched copy handed back: its transcript, nil where
// its harness wrote none, and its answer, for a made copy.
type launchBack struct {
	transcript []byte
	answer     []byte
}

// launchedTrial is the trial of copy p as its launch came out, before its
// answer is read and its compliance checked, and what the launch spent. Its
// served side is the runner's role receipt, which abcd's launch read from the
// harness's own events, so nothing in it is the host's word: the fields the
// runner could not pin are listed as it lists them, and a copy whose runner
// gave no answer lists every served field as not reported.
func launchedTrial(rec RunRecord, p launchPick, res copyResult, started, finished time.Time) (TrialRecord, *oracle.Cost) {
	pin := p.pin
	// The role's receipt names the brief and the receipt where they sit,
	// under the account's home: the trial names them from the home folder,
	// so no record carries the home path.
	role := res.Outcome.Receipt
	role.Brief = abcdhome.Display(handoffDirName, rec.RootSHA, rec.RunID, p.alias, launchBriefName)
	role.Contract = abcdhome.Display(handoffDirName, rec.RootSHA, rec.RunID, p.alias, receiptName(p.task.Kind))
	t := TrialRecord{SchemaVersion: TrialSchema, Run: rec.RunID, Key: p.h.Key, Kind: p.task.Kind, FrozenCommit: rec.FrozenCommit,
		Source: pin.Source, Runner: pin.Runner, ModelAsked: pin.Model, Settings: pin.Settings,
		StartedAt: started, FinishedAt: finished, Roles: []runner.RoleReceipt{role},
		Calls: []oracle.ReceiptRoute{}, Fallbacks: res.Fallbacks}
	if t.Fallbacks == nil {
		t.Fallbacks = []runner.FallbackReceipt{}
	}
	if p.task.Kind == KindCode {
		t.Intent = p.task.Intent
	}
	if !finished.IsZero() && finished.After(started) {
		t.ModelTimeMS = min(finished.Sub(started).Milliseconds(), maxMS)
	}
	route := res.Outcome.Receipt.Route
	var spent *oracle.Cost
	if res.Outcome.Answer != nil {
		t.ModelServed, t.ServedProvider, t.HarnessVersion = route.Model, route.ServedProvider, route.HarnessVersion
		if route.Usage != nil {
			t.Usage = *route.Usage
		}
		t.Cost = route.Cost
		spent = route.Cost
		t.Unpinned = slices.Clone(route.Unpinned)
	} else {
		var un oracle.Unpin
		for _, f := range oracle.ServedFields {
			un.Add(f, oracle.NotReported)
		}
		t.Unpinned = un.List()
		if f := res.Outcome.Failure; f != nil && f.Spent != nil {
			spent = f.Spent.Cost
		}
	}
	if t.Unpinned == nil {
		t.Unpinned = []oracle.Unpinned{}
	}
	switch res.Status {
	case StatusNotStarted:
		// A copy that never started spent nothing a trial carries.
		t.Usage, t.Cost, t.ModelTimeMS = oracle.Usage{}, nil, 0
		t.Status, t.Reason = StatusNotStarted, res.Reason
	case StatusExcluded, StatusFailed:
		t.Status, t.Reason = res.Status, clipTo(termsafe.SanitizeLine(res.Reason), maxReasonBytes)
	}
	if t.FinishedAt.IsZero() {
		t.FinishedAt = started
	}
	return t, spent
}

// madeTrial is t once its runner made the copy and its answer reads as its
// kind's: a code copy waits for its hidden tests' result (status ""), unless
// its receipt says it failed; a writing or a check copy is made, and judged
// after, never by its trial.
func madeTrial(t TrialRecord, task Task, answer []byte) TrialRecord {
	switch task.Kind {
	case KindCode:
		if rc, err := parseLaunchReceipt(answer); err == nil && rc.Outcome == OutcomeFailed {
			t.Status, t.Reason = StatusFailed, clipTo("the copy's runner reports it failed: "+termsafe.SanitizeLine(rc.Reason), maxReasonBytes)
		}
	case KindWriting:
		t.Status, t.Reason = StatusPassed, "the copy was made; people judge it in the run's judging session, never by its trial"
	default:
		t.Status, t.Reason = StatusPassed, "the copy was made; its findings are judged in the run's two-phase comparison, never by its trial"
	}
	return t
}

// finish checks launched copy p for compliance, reading its clone as data and
// its transcript for reads outside it, and records it as a host copy's
// hand-back does: a writing copy's text kept, a check copy's findings read
// before its clone is deleted, and the trial written through the secret
// screen. A made code copy the check left ranked is not written: it is
// returned awaiting its hidden tests' result.
func (l *copyLauncher) finish(rec RunRecord, p launchPick, t TrialRecord, back launchBack) (TrialRecord, bool, error) {
	key := p.h.Key
	store := cloneStore(l.home, rec.RootSHA)
	if t.Status != StatusNotStarted {
		c, err := checkCompliance(complianceInputs{RepoRoot: l.repoRoot, Store: store, Run: rec.RunID, Key: key, Alias: p.alias,
			Frozen: rec.FrozenCommit, Source: p.pin.Source, Transcript: back.transcript, State: labStore(l.home, rec.RootSHA),
			Handoff: handoffStore(l.home, rec.RootSHA), RanClosed: true})
		if err != nil {
			return t, false, err
		}
		t = c.apply(t)
	}
	if t.Status == "" {
		// Held to every bound but its status, which its score gives.
		probe := t
		probe.Status, probe.Reason = StatusFailed, "awaiting its hidden tests' result"
		if err := probe.validate(); err != nil {
			return t, false, fmt.Errorf("the trial is not kept: %w", err)
		}
		if _, err := screenTrial(l.repoRoot, probe); err != nil {
			return t, false, err
		}
		return t, true, nil
	}
	if err := t.validate(); err != nil {
		return t, false, fmt.Errorf("the trial is not written: %w", err)
	}
	made := back.answer != nil
	switch p.task.Kind {
	case KindWriting:
		if _, err := screenTrial(l.repoRoot, t); err != nil {
			return t, false, err
		}
		if made && t.Status != StatusFailed {
			text, _ := newRedactor(reviewInputs{RepoRoot: l.repoRoot, Store: store, Dir: cloneDir(store, rec.RunID, p.alias), Run: rec.RunID}).redact(string(back.answer))
			if err := keepWriting(l.repoRoot, l.labID, rec.RunID, key, text); err != nil {
				return t, false, err
			}
		}
		return t, false, writeTrial(l.repoRoot, l.labID, t)
	case KindCheck:
		if t.Status == StatusNotStarted {
			return t, false, writeTrial(l.repoRoot, l.labID, t)
		}
		if _, err := screenTrial(l.repoRoot, t); err != nil {
			return t, false, err
		}
		in := checkCopyInputs{RepoRoot: l.repoRoot, Store: store, Run: rec.RunID, Frozen: rec.FrozenCommit, Key: key,
			Form: answerForm(back.answer), Answer: back.answer, Gates: p.h.Gates}
		if !made {
			in.Failure, in.Answer = clipTo(t.Reason, maxReasonBytes), nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), complianceDeadline)
		material, err := readFrozenMaterial(ctx, l.repoRoot, rec.FrozenCommit, p.task.Material)
		cancel()
		if err != nil {
			return t, false, fmt.Errorf("its frozen material could not be read: %w", err)
		}
		in.Material = material
		cc, clone, err := recordCheckCopy(l.labID, in)
		if err != nil {
			return t, false, err
		}
		if !cc.Answered && t.Status == StatusPassed {
			t.Status, t.Reason = StatusFailed, cc.Reason
		}
		if err := writeTrial(l.repoRoot, l.labID, t); err != nil {
			return t, false, err
		}
		if err := removeClone(clone); err != nil {
			return t, false, afterWrite(fmt.Errorf("its trial is written, and its clone could not be deleted: %w", err))
		}
		return t, false, nil
	}
	return t, false, writeTrial(l.repoRoot, l.labID, t)
}

// answerForm is the form a launched check copy's findings are read in: the
// consistency pass's when they decode as its findings, and the doc-fidelity
// review's verdict otherwise, whose decoder then says what is wrong with them.
func answerForm(answer []byte) CheckForm {
	var ca consistencyAnswer
	if jsonstrict.Decode(answer, &ca) == nil {
		return CheckConsistency
	}
	return CheckDocFidelity
}

// scoreLaunched takes a launched code copy's hidden tests' result back from
// the host and writes its trial: the trial its launch kept, its compliance
// already checked, with the score marked as the host's report, since abcd
// neither placed nor ran the tests. It refuses, writing nothing: a copy with
// no trial kept (its launch wrote its trial already), a receipt that is not a
// made copy's with a score of at least one test, and one that names anything
// the launch recorded itself (the served model, the harness version, the
// times), or carries a transcript or an answer, which the launch read.
func scoreLaunched(repoRoot, labID string, h HandOff, rc CopyReceipt, back CopyBack) (TrialRecord, error) {
	key := h.Key
	if h.Trial == nil || h.Kind != KindCode {
		return TrialRecord{}, fmt.Errorf("copy %s was launched and its launch wrote its trial; a copy's trial is written once", key)
	}
	switch {
	case back.Transcript != nil || back.Answer != nil || back.Form != "":
		return TrialRecord{}, fmt.Errorf("copy %s was launched, and its launch read its transcript; hand it back with its hidden tests' result alone", key)
	case rc.Outcome != OutcomeMade || rc.Score == nil || rc.Score.Total < 1:
		return TrialRecord{}, fmt.Errorf("copy %s was made by its runner, and is handed back with its hidden tests' result (outcome made and a score of at least one test)", key)
	case rc.HarnessVersion != "" || rc.ModelServed != "" || !rc.StartedAt.IsZero() || !rc.FinishedAt.IsZero() || rc.ModelTimeMS != 0 || rc.GateWaitMS != 0:
		return TrialRecord{}, fmt.Errorf("copy %s's served model, harness version and times are its launch's own record, never the host's word; hand back its score alone", key)
	}
	t := *h.Trial
	if t.Key != key || t.Run != h.Run || t.Kind != KindCode || t.Status != "" {
		return TrialRecord{}, fmt.Errorf("copy %s's kept trial does not read as its own", key)
	}
	s := rc.Score
	t.Score = &Score{Passed: s.Passed, Total: s.Total, Detail: hostScoreDetail}
	t.Status, t.Reason = StatusFailed, fmt.Sprintf("its hidden tests: %d of %d passed", s.Passed, s.Total)
	if s.Passed == s.Total {
		t.Status = StatusPassed
	}
	t.Unpinned = append(slices.Clone(t.Unpinned), oracle.Unpinned{Field: TrialScore, Reason: oracle.SelfReported})
	if err := t.validate(); err != nil {
		return TrialRecord{}, fmt.Errorf("the trial is not written: %w", err)
	}
	return t, writeTrial(repoRoot, labID, t)
}

// spendBudget holds a launch's metered copies to the run's spending cap: what
// the run's trials spent, and what the copies in flight may spend, by their
// cells' declared cost per copy. A cost a runner reports in OpenRouter's
// credits is set against the cap at face value, an over-count where a credit
// is worth less than a dollar and never an under-count.
type spendBudget struct {
	mu       sync.Mutex
	cap      *Money
	perCopy  map[string]*Money
	spent    float64
	reserved float64
	// unknown names the first metered copy whose spend can be reckoned from
	// neither its record nor its cell's declared cost per copy; while one
	// stands, no metered copy is launched.
	unknown string
}

// newSpendBudget is run rec's budget, counted from its records, so it holds
// across launches as within one: each trial's spend, and each metered copy an
// earlier launch began that holds no trial yet (one launching, one made and
// awaiting its score, one interrupted), at the cost its kept trial reports,
// else its cell's declared cost per copy, else unknown.
func newSpendBudget(rec RunRecord, d Declaration, trials []TrialRecord, handed []HandOff) *spendBudget {
	b := &spendBudget{cap: rec.SpendCap, perCopy: map[string]*Money{}}
	for _, c := range d.Builders {
		b.perCopy[c.ID] = c.CostPerCopy
	}
	tried := map[TrialKey]bool{}
	for _, t := range trials {
		tried[t.Key] = true
		if !t.Source.Metered() || t.Status == StatusNotStarted {
			continue
		}
		if !b.add(t.Key.Cell, t.Cost) && b.unknown == "" {
			b.unknown = t.Key.String()
		}
	}
	for _, h := range handed {
		if tried[h.Key] || !h.Source.Metered() {
			continue
		}
		switch h.State {
		case HandOffLaunching, HandOffLaunched, HandOffInterrupted:
		default:
			continue
		}
		var c *oracle.Cost
		if h.Trial != nil {
			c = h.Trial.Cost
		}
		if !b.add(h.Key.Cell, c) && b.unknown == "" {
			b.unknown = h.Key.String()
		}
	}
	return b
}

// add counts a metered copy's spend: its reported cost, else its cell's
// declared cost per copy; false where it has neither.
func (b *spendBudget) add(cell string, c *oracle.Cost) bool {
	switch {
	case c != nil:
		b.spent += c.Amount
	case b.perCopy[cell] != nil:
		b.spent += b.perCopy[cell].Amount
	default:
		return false
	}
	return true
}

// reserve holds back what a copy on pin may spend before it is launched, and
// refuses one the cap's remainder does not cover.
func (b *spendBudget) reserve(pin CellPin) (float64, error) {
	if !pin.Source.Metered() {
		return 0, nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.cap == nil {
		return 0, fmt.Errorf("cell %s spends, and the run names no spending cap, so its copy is not launched", pin.Cell)
	}
	if b.unknown != "" {
		return 0, fmt.Errorf("copy %s's spend is neither recorded nor declared, so the spending cap's remainder cannot be reckoned and no metered copy is launched; "+
			"declare the cell's cost_per_copy in a new run", b.unknown)
	}
	need := 0.0
	if c := b.perCopy[pin.Cell]; c != nil {
		need = c.Amount
	}
	left := b.cap.Amount - b.spent - b.reserved
	if left <= 0 || need > left {
		return 0, fmt.Errorf("the spending cap's remainder is %.2f %s, with %.2f spent and %.2f held for the copies in flight, which does not cover a copy of cell %s (%.2f declared), so it waits",
			max(0, left), b.cap.Unit, b.spent, b.reserved, pin.Cell, need)
	}
	b.reserved += need
	return need, nil
}

// settle releases copy key's reserve and counts what it spent: its reported
// cost, else, where its runner ran, its declared cost per copy; a copy that ran
// with neither leaves the remainder unknown.
func (b *spendBudget) settle(key TrialKey, pin CellPin, need float64, spent *oracle.Cost, ran bool) {
	if !pin.Source.Metered() {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.reserved -= need
	switch {
	case spent != nil:
		b.spent += spent.Amount
	case !ran:
	case need > 0:
		b.spent += need
	case b.unknown == "":
		b.unknown = key.String()
	}
}
