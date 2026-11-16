package sweep

import (
	"context"
	"errors"
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
// and deletes a merged lane branch, with apply's own proofs, keeps the merged
// branches abcd did not make for the applying sweep, the store worktree's own
// branch among them (it has no lane's shape), and neither dispatches
// the reservation workflow nor files the waiting inbox report. Its report
// counts what it did and shows the rows in the sweep's own wording, naming
// where the full list is.
func TestTheAutomaticSweepClearsTheSpentAndStartsNothing(t *testing.T) {
	repo, o, wf := everyKind(t)
	branchWith(repo, testLane, "lane.txt")
	repo.Git("merge", "-q", "--ff-only", testLane)
	publish(repo)

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
	if out := repo.Git("branch", "--list", testLane); out != "" {
		t.Fatalf("the merged lane branch stands: %q: %+v", out, a.Result.Rows)
	}
	if out := repo.Git("branch", "--list", "wt"); out == "" {
		t.Fatal("the store worktree's branch, which has no lane's shape, was deleted unattended")
	}
	for _, r := range a.Result.Rows {
		if r.Kind == RowBranch && r.Name == "wt" && !strings.HasPrefix(r.Reason, "left for `abcd sweep --apply`") {
			t.Fatalf("the store worktree's branch is not listed for a person: %+v", r)
		}
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
	for _, want := range []string{"abcd swept on its own: 2 removed or deleted", "[removed  worktree  wt", "[deleted  branch  " + testLane, "[kept  branch  feat", "`abcd sweep` lists every row"} {
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

// testLane is a branch with the exact shape the build loop gives a lane.
const testLane = "build/run-2611160000000001-lane-1"

// TestTheAutomaticSweepDeletesOnlyTheBranchesAbcdMade: unattended, a merged
// lane branch is deleted locally and on origin, while an unprotected develop
// at the default branch, locally and on origin, and a peer's branch on origin
// with no commits past the default branch are kept and listed for the
// applying sweep; that sweep, run by a person, still deletes them
// (dcn-2611161459048806).
func TestTheAutomaticSweepDeletesOnlyTheBranchesAbcdMade(t *testing.T) {
	repo, origin := originRepo(t)
	const lane = testLane
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

// TestTheAutomaticSweepKeepsAPersonsBranchTheStoreCheckedOut: a person's
// existing develop, checked out into the worktree store (`ahoy worktree add`
// with an existing branch), merged and its worktree spent: unattended, the
// worktree is removed, and develop is kept locally and on origin, listed for
// the applying sweep, since a store worktree's branch is no proof abcd made
// it; that sweep, run by a person, still deletes develop.
func TestTheAutomaticSweepKeepsAPersonsBranchTheStoreCheckedOut(t *testing.T) {
	repo, origin := originRepo(t)
	dev := mergedOnOrigin(repo, "develop", "develop.txt")
	got, err := store.Add(repo.Root(), store.AddOptions{Name: "dev", Branch: "develop", Existing: true,
		Git: func(dir string, args ...string) (string, error) { return gitutil.Run(dir, args...) }})
	if err != nil {
		t.Fatal(err)
	}
	o := sweepOpts()
	o.Forge = (&fakeForge{prs: []pullRequest{mergedPR(5, "develop", dev)}}).gh

	a := Automatic(context.Background(), repo.Root(), autoOpts(t, repo, o))
	if !a.Ran || a.Fault != "" {
		t.Fatalf("ran %v, fault %q: %+v", a.Ran, a.Fault, a.Result.Rows)
	}
	if r := rowFor(t, a.Result, "dev"); r.Outcome != OutcomeRemoved || exists(got.Path) {
		t.Fatalf("the spent store worktree: %+v, standing %v; want it removed", r, exists(got.Path))
	}
	for _, r := range a.Result.Rows {
		if r.Name == "develop" && (r.Outcome != OutcomeKept || !strings.HasPrefix(r.Reason, "left for `abcd sweep --apply`")) {
			t.Fatalf("the person's develop was not kept for a person: %+v", r)
		}
	}
	if repo.Git("branch", "--list", "develop") == "" {
		t.Fatalf("the local develop was deleted unattended: %+v", a.Result.Rows)
	}
	if onOrigin(repo, origin, "develop") == "" {
		t.Fatalf("origin's develop was deleted unattended: %+v", a.Result.Rows)
	}

	res, err := runSweep(context.Background(), repo.Root(), o, true)
	if err != nil {
		t.Fatalf("%v: %+v", err, res.Rows)
	}
	if repo.Git("branch", "--list", "develop") != "" || onOrigin(repo, origin, "develop") != "" {
		t.Fatalf("the applying sweep kept develop: %+v", res.Rows)
	}
}

// TestTheAutomaticSweepKeepsABranchThatOnlyLooksLikeALane: a person's merged
// build/ci-tweak, and a build/ branch whose run id is not one the loop mints,
// are kept unattended, locally and on origin; the lane-shaped branch beside
// them is deleted.
func TestTheAutomaticSweepKeepsABranchThatOnlyLooksLikeALane(t *testing.T) {
	repo, origin := originRepo(t)
	var prs []pullRequest
	for i, b := range []string{"build/ci-tweak", "build/run-1-lane-1", testLane} {
		prs = append(prs, mergedPR(10+i, b, mergedOnOrigin(repo, b, strings.ReplaceAll(b, "/", "-")+".txt")))
	}
	o := sweepOpts()
	o.Forge = (&fakeForge{prs: prs}).gh

	a := Automatic(context.Background(), repo.Root(), autoOpts(t, repo, o))
	if !a.Ran || a.Fault != "" {
		t.Fatalf("ran %v, fault %q: %+v", a.Ran, a.Fault, a.Result.Rows)
	}
	for _, b := range []string{"build/ci-tweak", "build/run-1-lane-1"} {
		if repo.Git("branch", "--list", b) == "" || onOrigin(repo, origin, b) == "" {
			t.Fatalf("%s, a person's, was deleted unattended: %+v", b, a.Result.Rows)
		}
	}
	if repo.Git("branch", "--list", testLane) != "" || onOrigin(repo, origin, testLane) != "" {
		t.Fatalf("the lane's branch stands: %+v", a.Result.Rows)
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

// spentWithEnv is a merged, clean store worktree named wt holding an ignored
// .env, the ignore rule set by how: the repository's .gitignore, the global
// ignore file git reads by default under XDG_CONFIG_HOME, or a
// core.excludesFile a global git config names. Every file lives under the
// test's own HOME; the config git reads is the test's, never the machine's.
func spentWithEnv(t *testing.T, how string) (*gittest.Repo, string, string) {
	t.Helper()
	repo, home := sweepRepo(t)
	config := filepath.Join(home, ".config")
	t.Setenv("XDG_CONFIG_HOME", config)
	switch how {
	case "repository":
		repo.Write(".gitignore", ".abcd/.work.local/\n.env\n")
		repo.Commit("ignore .env")
	case "global ignore file":
		if err := os.MkdirAll(filepath.Join(config, "git"), 0o755); err != nil {
			t.Fatal(err)
		}
		write(t, filepath.Join(config, "git", "ignore"), ".env\n")
	case "global core.excludesFile":
		write(t, filepath.Join(home, "personal-ignore"), ".env\n")
		write(t, filepath.Join(home, ".gitconfig"), "[core]\n\texcludesFile = "+filepath.Join(home, "personal-ignore")+"\n")
	default:
		t.Fatalf("no ignore rule %q", how)
	}
	wt := mergedAdd(t, repo, "wt")
	env := filepath.Join(wt, ".env")
	write(t, env, "TOKEN=a person's own\n")
	return repo, wt, env
}

// TestTheAutomaticSweepKeepsAStoreWorktreeHoldingIgnoredFiles
// (iss-2611161722120909, dcn-2611161654034047's ruling extended): a spent
// store worktree holding an ignored .env is kept by the sweep abcd runs on its
// own, its row naming .env and the applying form, and the .env still there;
// the applying sweep a person runs removes the same worktree. The .env is
// counted however git ignores it in the removal abcd runs: by the
// repository's .gitignore, or by the global ignore file git reads by default,
// which a listing with core.excludesFile turned off misses. One a
// core.excludesFile in a global config ignores is no ignored file to the
// removal abcd runs, which reads no global config: the worktree reads as
// dirty, and is kept with its work saved.
func TestTheAutomaticSweepKeepsAStoreWorktreeHoldingIgnoredFiles(t *testing.T) {
	for _, how := range []string{"repository", "global ignore file"} {
		t.Run(how, func(t *testing.T) {
			repo, wt, env := spentWithEnv(t, how)
			a := Automatic(context.Background(), repo.Root(), autoOpts(t, repo, sweepOpts()))
			if !a.Ran || a.Fault != "" {
				t.Fatalf("ran %v, fault %q", a.Ran, a.Fault)
			}
			r := rowFor(t, a.Result, "wt")
			if r.Outcome != OutcomeKept || !exists(env) {
				t.Fatalf("the worktree holding an ignored .env: %+v, .env standing %v; want it kept", r, exists(env))
			}
			for _, want := range []string{"left for `abcd sweep --apply`", "1 ignored path", ".env"} {
				if !strings.Contains(r.Reason, want) {
					t.Fatalf("the row's reason %q lacks %q", r.Reason, want)
				}
			}
			if !hasEvidence(r, KindIgnored, Proven) {
				t.Fatalf("the row carries no proof of the ignored file: %+v", r.Evidence)
			}
			if text := a.Text(); !strings.Contains(text, "[kept  worktree  wt") || !strings.Contains(text, ".env") {
				t.Fatalf("the report %q does not name the kept worktree's .env", text)
			}

			if got := rowFor(t, apply(t, repo, sweepOpts()), "wt"); got.Outcome != OutcomeRemoved || exists(wt) {
				t.Fatalf("the applying sweep: %+v, standing %v; want the worktree removed", got, exists(wt))
			}
		})
	}
	t.Run("global core.excludesFile", func(t *testing.T) {
		repo, _, env := spentWithEnv(t, "global core.excludesFile")
		a := Automatic(context.Background(), repo.Root(), autoOpts(t, repo, sweepOpts()))
		if r := rowFor(t, a.Result, "wt"); r.Outcome == OutcomeRemoved || !exists(env) {
			t.Fatalf("the worktree whose .env only a global core.excludesFile ignores: %+v, .env standing %v; want it kept", r, exists(env))
		}
	})
}

// TestTheAutomaticSweepRemovesAWorktreeIgnoringOnlyItsLocalTier: ignored
// files inside the local tier alone keep nothing, since the removal archives
// the tier first.
func TestTheAutomaticSweepRemovesAWorktreeIgnoringOnlyItsLocalTier(t *testing.T) {
	repo, _ := sweepRepo(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	wt := mergedAdd(t, repo, "wt")
	if err := os.MkdirAll(filepath.Join(wt, ".abcd", ".work.local", "scratch"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(wt, ".abcd", ".work.local", "NEXT.md"), "handover\n")
	write(t, filepath.Join(wt, ".abcd", ".work.local", "scratch", "x.txt"), "x\n")
	a := Automatic(context.Background(), repo.Root(), autoOpts(t, repo, sweepOpts()))
	if r := rowFor(t, a.Result, "wt"); r.Outcome != OutcomeRemoved || exists(wt) {
		t.Fatalf("the worktree ignoring only its local tier: %+v, standing %v; want it removed", r, exists(wt))
	}
}

// TestTheAutomaticSweepKeepsAWorktreeWhoseIgnoredFilesCannotBeListed: a
// listing that fails, or whose answer cannot be trusted, keeps the worktree
// on the automatic path, its row saying so.
func TestTheAutomaticSweepKeepsAWorktreeWhoseIgnoredFilesCannotBeListed(t *testing.T) {
	for name, c := range map[string]struct {
		list func(string) (store.IgnoredState, error)
		want string
	}{
		"fails": {func(string) (store.IgnoredState, error) {
			return store.IgnoredState{}, errors.New("git refused the listing")
		}, "git refused the listing"},
		"untrusted": {func(string) (store.IgnoredState, error) { return store.IgnoredState{Unread: true}, nil }, "could not be listed"},
	} {
		t.Run(name, func(t *testing.T) {
			repo, _ := sweepRepo(t)
			wt := mergedAdd(t, repo, "wt")
			o := sweepOpts()
			o.ignored = c.list
			a := Automatic(context.Background(), repo.Root(), autoOpts(t, repo, o))
			r := rowFor(t, a.Result, "wt")
			if r.Outcome != OutcomeKept || !exists(wt) {
				t.Fatalf("the worktree whose ignored files could not be listed: %+v; want it kept", r)
			}
			for _, want := range []string{"left for `abcd sweep --apply`", c.want} {
				if !strings.Contains(r.Reason, want) {
					t.Fatalf("the row's reason %q lacks %q", r.Reason, want)
				}
			}
		})
	}
}

// TestAPersonsSweepNamesTheIgnoredFilesARemovalTakes: the sweep a person runs,
// its bare listing and its applying form, names on a remove row the ignored
// files outside the local tier that go with the worktree, as information,
// and still removes it.
func TestAPersonsSweepNamesTheIgnoredFilesARemovalTakes(t *testing.T) {
	repo, wt, env := spentWithEnv(t, "global ignore file")
	r := rowFor(t, dryRun(t, repo, sweepOpts()), "wt")
	if r.Action != ActionRemove {
		t.Fatalf("the bare listing kept the worktree: %+v", r)
	}
	if r.Ignored == nil || r.Ignored.Count != 1 || r.Ignored.Paths[0] != ".env" {
		t.Fatalf("the bare listing's row does not carry the ignored .env: %+v", r.Ignored)
	}
	for _, want := range []string{"1 ignored path(s) outside its local tier go with it, unarchived: .env"} {
		if !strings.Contains(r.Reason, want) {
			t.Fatalf("the bare listing's reason %q lacks %q", r.Reason, want)
		}
	}
	if !hasEvidence(r, KindIgnored, Proven) {
		t.Fatalf("the row carries no proof of the ignored file: %+v", r.Evidence)
	}
	got := rowFor(t, apply(t, repo, sweepOpts()), "wt")
	if got.Outcome != OutcomeRemoved || exists(wt) || exists(env) {
		t.Fatalf("the applying sweep: %+v; want the worktree removed", got)
	}
	if !strings.Contains(got.Reason, ".env") {
		t.Fatalf("the applying form's row %q does not name the ignored .env", got.Reason)
	}
}

// TestAPersonsSweepSaysWhenTheIgnoredFilesCouldNotBeListed: a failed listing
// does not keep a worktree a person's sweep removes, but its row says so.
func TestAPersonsSweepSaysWhenTheIgnoredFilesCouldNotBeListed(t *testing.T) {
	repo, _ := sweepRepo(t)
	mergedAdd(t, repo, "wt")
	o := sweepOpts()
	o.ignored = func(string) (store.IgnoredState, error) {
		return store.IgnoredState{}, errors.New("git refused the listing")
	}
	r := rowFor(t, dryRun(t, repo, o), "wt")
	if r.Action != ActionRemove {
		t.Fatalf("a failed listing kept the worktree from a person's sweep: %+v", r)
	}
	if !strings.Contains(r.Reason, "its ignored files could not be listed (git refused the listing)") {
		t.Fatalf("the row %q does not say its ignored files could not be listed", r.Reason)
	}
	if r.Ignored == nil || !r.Ignored.Unread {
		t.Fatalf("the row's ignored state: %+v; want it unread", r.Ignored)
	}
}

// TestTheAutomaticLineKeepsTheIgnoredWarningAtRealPathLengths: the bounded
// line the automatic sweep prints names a kept worktree's ignored count and
// first path even where its store path is as long as a real one.
func TestTheAutomaticLineKeepsTheIgnoredWarningAtRealPathLengths(t *testing.T) {
	repo, home := sweepRepo(t)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	repo.Write(".gitignore", ".abcd/.work.local/\n*.local\n")
	repo.Commit("ignore local files")
	name := "lane-bd-sweep-keeps-ignored-files"
	wt := mergedAdd(t, repo, name)
	write(t, filepath.Join(wt, ".env.production.local"), "TOKEN=a person's own\n")
	a := Automatic(context.Background(), repo.Root(), autoOpts(t, repo, sweepOpts()))
	r := rowFor(t, a.Result, name)
	if want := len(abcdhome.Display("worktrees")+"/") + 40 + 1 + len(name); len(r.Path) < want {
		t.Fatalf("the row's path %q is shorter than a real store path (%d bytes)", r.Path, want)
	}
	text := a.Text()
	for _, want := range []string{"1 ignored path", ".env.production.local"} {
		if !strings.Contains(text, want) {
			t.Fatalf("the automatic line lacks %q:\n%s", want, text)
		}
	}
}

// TestTheAutomaticSweepReadsTheIgnoredFilesOnceMoreBeforeRemoving: an
// ignored file written after the judgement and before the removal is seen by
// the last check before `git worktree remove`, which keeps the worktree.
func TestTheAutomaticSweepReadsTheIgnoredFilesOnceMoreBeforeRemoving(t *testing.T) {
	repo, home := sweepRepo(t)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	repo.Write(".gitignore", ".abcd/.work.local/\n.env\n")
	repo.Commit("ignore .env")
	wt := mergedAdd(t, repo, "wt")
	env := filepath.Join(wt, ".env")
	o := sweepOpts()
	reads := 0
	o.ignored = func(dir string) (store.IgnoredState, error) {
		reads++
		s, err := store.IgnoredForRemoval(dir)
		if reads == 1 {
			// Judged with nothing ignored; the file arrives after.
			write(t, env, "TOKEN=written after the judgement\n")
		}
		return s, err
	}
	a := Automatic(context.Background(), repo.Root(), autoOpts(t, repo, o))
	r := rowFor(t, a.Result, "wt")
	if r.Outcome != OutcomeKept || !exists(env) {
		t.Fatalf("the worktree whose .env arrived before the removal: %+v, .env standing %v; want it kept", r, exists(env))
	}
	if !strings.Contains(r.Reason, ".env") || !hasEvidence(r, KindIgnored, Proven) {
		t.Fatalf("the row does not name the .env the last check found: %+v", r)
	}
	if reads < 2 {
		t.Fatalf("the ignored files were read %d time(s); want a last read before the removal", reads)
	}
}
