package sweep

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	store "github.com/intentdriven/abcd/internal/core/ahoy/worktree"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/report"
	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// managed answers the repository as one abcd manages, asking nothing.
func managed(context.Context, string) (bool, error) { return true, nil }

// autoOpts is o as an automatic sweep of repo runs it: managed, its setting
// read at the repository and the test's HOME.
func autoOpts(t *testing.T, repo *gittest.Repo, o Options) AutoOptions {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	return AutoOptions{Options: o, Roots: &layered.Roots{Repo: repo.Root(), Home: home}, Managed: managed}
}

// TestTheAutomaticSweepClearsTheSpentAndStartsNothing: in a managed
// repository the automatic sweep removes a merged clean worktree of the store
// and deletes its merged branch, with apply's own proofs, keeps the merged
// branches abcd did not make for the applying sweep, and neither dispatches
// the reservation workflow nor files the waiting inbox report. Its report
// counts what it did and shows the rows in the sweep's own wording, naming
// where the full list is.
func TestTheAutomaticSweepClearsTheSpentAndStartsNothing(t *testing.T) {
	repo, o, wf := everyKind(t)

	a := Automatic(context.Background(), repo.Root(), autoOpts(t, repo, o))
	if !a.Ran || a.Fault != "" {
		t.Fatalf("ran %v, fault %q", a.Ran, a.Fault)
	}
	acted := map[RowKind]Outcome{}
	for _, r := range a.Result.Rows {
		if r.Kind == RowReservation || r.Kind == RowInbox {
			t.Fatalf("the automatic sweep judged a %s: %+v", r.Kind, r)
		}
		if r.Outcome != OutcomeKept {
			acted[r.Kind] = r.Outcome
		}
	}
	for k, want := range map[RowKind]Outcome{RowWorktree: OutcomeRemoved, RowBranch: OutcomeDeleted} {
		if acted[k] != want {
			t.Fatalf("no %s row was %s: %+v", k, want, a.Result.Rows)
		}
	}
	if _, ok := acted[RowRemoteBranch]; ok {
		t.Fatalf("a branch on origin abcd did not make was deleted: %+v", a.Result.Rows)
	}
	if list := repo.Git("worktree", "list", "--porcelain"); strings.Contains(list, "branch refs/heads/wt\n") {
		t.Fatalf("git still lists the merged clean worktree:\n%s", list)
	}
	if out := repo.Git("branch", "--list", "wt"); out != "" {
		t.Fatalf("the merged store worktree's branch stands: %q", out)
	}
	if out := repo.Git("branch", "--list", "loc"); out == "" {
		t.Fatal("the merged local branch abcd did not make was deleted")
	}
	if len(wf.dispatches()) != 0 {
		t.Fatalf("the automatic sweep dispatched the reservation workflow %d times", len(wf.dispatches()))
	}
	if waiting, err := report.List(); err != nil || len(waiting) != 1 {
		t.Fatalf("the inbox report is left waiting: %d, %v", len(waiting), err)
	}
	text := a.Text()
	for _, want := range []string{"abcd swept on its own: 2 removed or deleted", "[removed  worktree  wt", "[deleted  branch  wt", "[kept  branch  feat", "`abcd sweep` lists every row"} {
		if !strings.Contains(text, want) {
			t.Fatalf("the report %q lacks %q", text, want)
		}
	}
}

// TestTheAutomaticSweepKeepsADirtyOrLiveWorktree: a dirty worktree is kept and
// its work saved as a patch, as apply does; a merged clean worktree a live
// process is in is kept, naming the process.
func TestTheAutomaticSweepKeepsADirtyOrLiveWorktree(t *testing.T) {
	repo, home := sweepRepo(t)
	wip := dirtyWip(t, repo, home)
	busy := mergedAdd(t, repo, "busy")
	o := sweepOpts()
	o.Probe.Processes = cwds(busy)

	a := Automatic(context.Background(), repo.Root(), autoOpts(t, repo, o))
	if !a.Ran || a.Fault != "" {
		t.Fatalf("ran %v, fault %q", a.Ran, a.Fault)
	}
	if r := rowFor(t, a.Result, "wip"); r.Outcome != OutcomeSaved || !exists(wip) {
		t.Fatalf("the dirty worktree: %+v, standing %v; want its work saved and it kept", r, exists(wip))
	}
	if r := rowFor(t, a.Result, "busy"); r.Outcome != OutcomeKept || !strings.Contains(r.Reason, "in use: process") || !exists(busy) {
		t.Fatalf("the worktree a process is in: %+v; want it kept, naming the process", r)
	}
	if text := a.Text(); !strings.Contains(text, "1 saved as a patch and kept") || !strings.Contains(text, "[saved  worktree  wip") {
		t.Fatalf("the report %q names neither the save nor its row", text)
	}
}

// TestTheAutomaticSweepIsOffWhereTheSettingSaysSo: sweep.automatic false in
// the repository's file or the machine's removes nothing and says which layer
// turned it off; a value that is not a bool reads as off and says so; outside
// a managed repository nothing is removed and nothing is said.
func TestTheAutomaticSweepIsOffWhereTheSettingSaysSo(t *testing.T) {
	cases := []struct {
		name, repoFile, homeFile, want string
		unmanaged                      bool
	}{
		{name: "repository", repoFile: `{"sweep": {"automatic": false}}`, want: "sweep.automatic is off (repo layer, .abcd/config.json)"},
		{name: "machine", homeFile: `{"sweep": {"automatic": false}}`, repoFile: `{"sweep": {"automatic": true}}`, want: "sweep.automatic is off (machine layer"},
		{name: "not a bool", repoFile: `{"sweep": {"automatic": "yes"}}`, want: "it takes true or false"},
		{name: "misspelt", repoFile: `{"sweep": {"automatc": true}}`, want: "could not be read"},
		{name: "unmanaged", unmanaged: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			repo, home := sweepRepo(t)
			p := mergedAdd(t, repo, "done")
			if c.repoFile != "" {
				repo.Write(".abcd/config.json", c.repoFile)
			}
			if c.homeFile != "" {
				if err := os.MkdirAll(abcdhome.Path(home), 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(abcdhome.Path(home, "config.json"), []byte(c.homeFile), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			o := sweepOpts()
			asked := recordPrunes(&o)
			ao := autoOpts(t, repo, o)
			if c.unmanaged {
				ao.Managed = func(context.Context, string) (bool, error) { return false, nil }
			}

			a := Automatic(context.Background(), repo.Root(), ao)
			if a.Ran || len(*asked) != 0 || !exists(p) {
				t.Fatalf("ran %v, prunes %q, worktree standing %v; want nothing removed", a.Ran, *asked, exists(p))
			}
			text := a.Text()
			if c.unmanaged {
				if text != "" {
					t.Fatalf("outside a managed repository the sweep says %q; want nothing", text)
				}
				return
			}
			if !strings.HasPrefix(text, "abcd did not sweep on its own: ") || !strings.Contains(text, c.want) {
				t.Fatalf("the report %q; want it to name %q", text, c.want)
			}
		})
	}
}

// TestAnAutomaticSweepsFaultIsCarriedNotRaised: a prune that panics, or a
// reader of the build loop's holds that fails, is the answer's fault, said
// in its report; Automatic neither panics nor returns an error.
func TestAnAutomaticSweepsFaultIsCarriedNotRaised(t *testing.T) {
	repo, _ := sweepRepo(t)
	mergedAdd(t, repo, "done")

	o := sweepOpts()
	o.pruneOne = func(string, string, store.PruneOptions) (store.Pruned, error) { panic("the prune fell over") }
	a := Automatic(context.Background(), repo.Root(), autoOpts(t, repo, o))
	if !strings.Contains(a.Fault, "the prune fell over") || !strings.Contains(a.Text(), "refused: ") {
		t.Fatalf("fault %q, report %q", a.Fault, a.Text())
	}

	o = sweepOpts()
	o.Held = nil
	a = Automatic(context.Background(), repo.Root(), autoOpts(t, repo, o))
	if a.Fault == "" || !strings.Contains(a.Text(), "the act it followed stands") {
		t.Fatalf("fault %q, report %q", a.Fault, a.Text())
	}
}

// TestTheAutomaticSweepsReportIsBounded: past the first few rows the report
// counts the rest and names where the full list is.
func TestTheAutomaticSweepsReportIsBounded(t *testing.T) {
	var rows []Row
	for _, n := range []string{"a", "b", "c", "d", "e", "f", "g"} {
		rows = append(rows, Row{Kind: RowWorktree, Name: n, Action: ActionKeep, Outcome: OutcomeKept, Reason: "unmerged"})
	}
	text := Auto{Setting: Setting{On: true}, Ran: true, Result: Result{Rows: rows, Applied: true}}.Text()
	if strings.Count(text, "[kept  worktree") != autoRows || !strings.Contains(text, "2 more; `abcd sweep` lists every row") {
		t.Fatalf("the report %q", text)
	}
}

// TestTheAutomaticSweepDeletesOnlyTheBranchesAbcdMade: unattended, a merged
// lane branch is deleted locally and on origin, while an unprotected develop
// at the default branch, locally and on origin, and a peer's branch on origin
// with no commits past the default branch are kept and listed for the
// applying sweep; that sweep, run by a person, still deletes them
// (dcn-2611161459048806).
func TestTheAutomaticSweepDeletesOnlyTheBranchesAbcdMade(t *testing.T) {
	repo, origin := originRepo(t)
	const lane = "build/run-1-lane-1"
	tip := mergedOnOrigin(repo, lane, "lane.txt")
	dev := mergedOnOrigin(repo, "develop", "develop.txt")
	peer := mergedOnOrigin(repo, "peer-work", "peer.txt")
	repo.Git("branch", "-q", "-d", "peer-work")
	o := sweepOpts()
	o.Forge = (&fakeForge{prs: []pullRequest{mergedPR(4, lane, tip), mergedPR(5, "develop", dev), mergedPR(6, "peer-work", peer)}}).gh

	a := Automatic(context.Background(), repo.Root(), autoOpts(t, repo, o))
	if !a.Ran || a.Fault != "" {
		t.Fatalf("ran %v, fault %q: %+v", a.Ran, a.Fault, a.Result.Rows)
	}
	for _, r := range a.Result.Rows {
		if r.Name == lane && r.Outcome != OutcomeDeleted {
			t.Fatalf("the lane's branch was not deleted: %+v", r)
		}
		if (r.Name == "develop" || r.Name == "peer-work") && (r.Outcome != OutcomeKept || !strings.HasPrefix(r.Reason, "left for `abcd sweep --apply`: abcd deletes on its own only the branches it made")) {
			t.Fatalf("a branch abcd did not make was not kept for a person: %+v", r)
		}
	}
	if onOrigin(repo, origin, lane) != "" || repo.Git("branch", "--list", lane) != "" {
		t.Fatalf("the lane's branch stands")
	}
	for _, b := range []string{"develop", "peer-work"} {
		if onOrigin(repo, origin, b) == "" {
			t.Fatalf("origin's %s was deleted unattended", b)
		}
	}
	if repo.Git("branch", "--list", "develop") == "" {
		t.Fatalf("the local develop was deleted unattended")
	}
	if text := a.Text(); !strings.Contains(text, "left for `abcd sweep --apply`") {
		t.Fatalf("the report %q does not list what it left for a person", text)
	}

	res, err := runSweep(context.Background(), repo.Root(), o, true)
	if err != nil {
		t.Fatalf("%v: %+v", err, res.Rows)
	}
	for _, b := range []string{"develop", "peer-work"} {
		if onOrigin(repo, origin, b) != "" {
			t.Fatalf("the applying sweep kept origin's %s: %+v", b, res.Rows)
		}
	}
	if repo.Git("branch", "--list", "develop") != "" {
		t.Fatalf("the applying sweep kept the local develop: %+v", res.Rows)
	}
}

// TestAnAppliedSweepStopsBetweenRemovalsWhenItsContextEnds: a run whose
// context ends after one removal asks for no other: the next merged clean
// worktree is kept, saying the run was stopped, and the run carries that.
func TestAnAppliedSweepStopsBetweenRemovalsWhenItsContextEnds(t *testing.T) {
	repo, _ := sweepRepo(t)
	first := mergedAdd(t, repo, "first")
	second := mergedAdd(t, repo, "second")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	o := sweepOpts()
	var asked []string
	o.pruneOne = func(root, name string, po store.PruneOptions) (store.Pruned, error) {
		asked = append(asked, name)
		defer cancel()
		return store.PruneOne(root, name, po)
	}

	res, err := runApply(ctx, repo.Root(), o)
	if err == nil || len(asked) != 1 {
		t.Fatalf("err %v, prunes asked %q; want one, then the run refused as stopped", err, asked)
	}
	left := first
	if asked[0] == "first" {
		left = second
	}
	r := rowFor(t, res, filepath.Base(left))
	if r.Outcome != OutcomeKept || !strings.Contains(r.Reason, "the run was stopped before this worktree was acted on") || !exists(left) {
		t.Fatalf("the worktree after the stop: %+v, standing %v", r, exists(left))
	}
}

// TestTheSweepsFetchEndsWithItsContext: the fetch the sweep makes before it
// judges ends with the run's context, so a run whose bound ends there judges
// nothing.
func TestTheSweepsFetchEndsWithItsContext(t *testing.T) {
	repo, _ := originRepo(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	clear, err := gitutil.SetBeforeHookedGitAt(func(_ string, args []string) error {
		if gitutil.Subcommand(args) == "fetch" {
			cancel()
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer clear()
	if _, err := prepare(ctx, repo.Root(), sweepOpts()); err == nil || !strings.Contains(err.Error(), "nothing was judged") {
		t.Fatalf("a fetch whose context ended answered %v; want the run refused before judging", err)
	}
}

// TestADeletesPushOnOriginEndsWithItsContext: the push that deletes a merged
// branch on origin is bound to the run's context: a context that ends as the
// push starts leaves the branch on origin and says the run was stopped.
func TestADeletesPushOnOriginEndsWithItsContext(t *testing.T) {
	repo, origin := originRepo(t)
	tip := mergedOnOrigin(repo, "build/run-1-lane-1", "lane.txt")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	clear, err := gitutil.SetBeforeHookedGitAt(func(_ string, args []string) error {
		if gitutil.Subcommand(args) == "push" {
			cancel()
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer clear()
	res, err := sweepRemote(ctx, repo.Root(), remoteOpts(&fakeForge{prs: []pullRequest{mergedPR(4, "build/run-1-lane-1", tip)}}), true)
	if err == nil {
		t.Fatalf("a push whose context ended was no fault: %+v", res.Rows)
	}
	if onOrigin(repo, origin, "build/run-1-lane-1") == "" {
		t.Fatalf("the branch was deleted on origin after the run's context ended")
	}
	if r := rowFor(t, res, "build/run-1-lane-1"); !strings.Contains(r.Reason, "the run was stopped while the delete was pushed") {
		t.Fatalf("the row: %+v", r)
	}
}

// TestTheAutomaticSweepsReportReadsRightWhenRefusedOrHomeless: a refusal
// before anything was judged says so rather than counting nothing removed,
// with its text bounded; a home that cannot be resolved reads the machine
// layer as unreadable, so the sweep is off and says why.
func TestTheAutomaticSweepsReportReadsRightWhenRefusedOrHomeless(t *testing.T) {
	long := strings.Repeat("x", 4*autoFaultCap)
	text := Auto{Setting: Setting{On: true}, Ran: true, Result: Result{Rows: []Row{}}, Fault: long, unjudged: true}.Text()
	if !strings.HasPrefix(text, "abcd's own sweep was refused before it judged anything: ") || strings.Contains(text, "0 removed") || len(text) > 2*autoFaultCap {
		t.Fatalf("the report %q", text)
	}
	s := readAutomatic(layered.Roots{Repo: t.TempDir()}, true)
	if s.On || !s.Said || !strings.Contains(s.Why, "the home directory is unresolved") {
		t.Fatalf("a home that cannot be resolved read as %+v", s)
	}
}
