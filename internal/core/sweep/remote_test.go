package sweep

import (
	"cmp"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/gittest"
)

// pullRequest is one pull request as the fake forge lists it, in gh's field
// names.
type pullRequest struct {
	Number      int    `json:"number"`
	HeadRefName string `json:"headRefName"`
	HeadRefOid  string `json:"headRefOid"`
	BaseRefName string `json:"baseRefName"`
	State       string `json:"state"`
}

// fakeForge answers `gh pr list` from prs, filtered by the call's --state and
// --head as gh filters them; a per-branch listing (--head, the last check
// before a delete) answers from recheck instead once it is set. It answers
// `gh repo view` with view, or hostedView when that is empty.
//
// onRecheck, when set, runs at the first per-branch listing: the moment the
// last check begins, before it reads the repository's own state again.
//
// mu guards the fake's state while a call runs: the sweep runs each forge
// call in a goroutine and abandons one that outlives its bound, so an
// abandoned call may still be inside the fake when the next one begins. A
// test sets the fields before the run starts.
type fakeForge struct {
	mu           sync.Mutex
	prs, recheck []pullRequest
	view         string
	calls        [][]string
	onRecheck    func()
}

func (f *fakeForge) gh(_ string, args ...string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, args)
	if isRepoView(args) {
		return cmp.Or(f.view, hostedView), nil
	}
	value := func(flag string) string {
		for i, a := range args {
			if v, ok := strings.CutPrefix(a, flag+"="); ok {
				return v
			}
			if a == flag && i+1 < len(args) {
				return args[i+1]
			}
		}
		return ""
	}
	state, head, base := value("--state"), value("--head"), value("--base")
	if head != "" && f.onRecheck != nil {
		f.onRecheck()
		f.onRecheck = nil
	}
	list := f.prs
	if head != "" && f.recheck != nil {
		list = f.recheck
	}
	out := []pullRequest{}
	for _, pr := range list {
		if (state == "all" || strings.EqualFold(state, pr.State)) && (head == "" || head == pr.HeadRefName) && (base == "" || base == pr.BaseRefName) {
			out = append(out, pr)
		}
	}
	b, err := json.Marshal(out)
	return string(b), err
}

// originRepo is sweepRepo with an origin that holds main, kept under HOME so
// a path git prints of it reads home-redacted, and fetched. The sweep reads
// origin's address as the forge's (hostedOrigin), while git reaches the
// local repository.
func originRepo(t *testing.T) (*gittest.Repo, string) {
	t.Helper()
	repo, home := sweepRepo(t)
	origin := filepath.Join(home, "origin.git")
	gittest.InitBare(t, origin)
	repo.Git("remote", "add", "origin", origin)
	publish(repo)
	hostedOrigin(t, origin)
	return repo, origin
}

// mergedOnOrigin cuts branch from main with one commit at rel, pushes it to
// origin, fast-forwards main over it and publishes main: a pull request merged
// at the branch's tip. It answers the tip.
func mergedOnOrigin(repo *gittest.Repo, branch, rel string) string {
	branchWith(repo, branch, rel)
	repo.Git("push", "-q", "origin", branch)
	repo.Git("merge", "-q", "--ff-only", branch)
	publish(repo)
	return repo.Git("rev-parse", branch)
}

// squashedOnOrigin cuts branch from main with one commit at rel, pushes it to
// origin, merges it into main by a squash and publishes main: merged by
// content, which git alone shows. It answers the branch's tip.
func squashedOnOrigin(repo *gittest.Repo, branch, rel string) string {
	branchWith(repo, branch, rel)
	repo.Git("push", "-q", "origin", branch)
	repo.Git("merge", "-q", "--squash", branch)
	repo.Git("commit", "-q", "-m", "squash "+branch)
	publish(repo)
	return repo.Git("rev-parse", branch)
}

// mergedPR is the forge's record of a pull request from branch merged at tip.
func mergedPR(n int, branch, tip string) pullRequest {
	return pullRequest{Number: n, HeadRefName: branch, HeadRefOid: tip, BaseRefName: "main", State: "MERGED"}
}

// onOrigin is origin's tip of branch, "" when origin does not hold it.
func onOrigin(repo *gittest.Repo, origin, branch string) string {
	return strings.TrimSpace(repo.Git("--git-dir", origin, "for-each-ref", "--format=%(objectname)", "refs/heads/"+branch))
}

func remoteOpts(f *fakeForge) Options {
	return Options{Probe: idleProbe(), Forge: f.gh}
}

func sweepRemoteOK(t *testing.T, repo *gittest.Repo, o Options, apply bool) Result {
	t.Helper()
	res, err := sweepRemote(context.Background(), repo.Root(), o, apply)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func remoteRowFor(t *testing.T, res Result, name string) Row {
	t.Helper()
	for _, r := range res.Rows {
		if r.Kind == RowRemoteBranch && r.Name == name {
			return r
		}
	}
	t.Fatalf("no remote-branch row named %q in %+v", name, res.Rows)
	return Row{}
}

// TestAMergedBranchOnOriginIsDeletedUnderALeaseAndNamed is criterion 8's
// first half: a branch whose pull request the forge records merged at its
// tip, and one merged by a squash with no record at all, are each deleted on
// origin and named; the dry run before it changed nothing on either side.
func TestAMergedBranchOnOriginIsDeletedUnderALeaseAndNamed(t *testing.T) {
	repo, origin := originRepo(t)
	tip := mergedOnOrigin(repo, "feat", "feat.txt")
	branchWith(repo, "squash", "sq.txt")
	repo.Git("push", "-q", "origin", "squash")
	repo.Git("merge", "-q", "--squash", "squash")
	repo.Git("commit", "-q", "-m", "squash")
	publish(repo)
	f := &fakeForge{prs: []pullRequest{mergedPR(4, "feat", tip)}}

	roots := []string{repo.Root(), origin}
	before := treeHash(t, roots...)
	dry := sweepRemoteOK(t, repo, remoteOpts(f), false)
	after := treeHash(t, roots...)
	for p, h := range before {
		if after[p] != h {
			t.Errorf("the dry run changed %s", p)
		}
	}
	for p := range after {
		if _, ok := before[p]; !ok {
			t.Errorf("the dry run made %s", p)
		}
	}
	for name, by := range map[string]string{"feat": "judged by forge", "squash": "judged by content"} {
		r := remoteRowFor(t, dry, name)
		if r.Action != ActionDelete || !strings.Contains(r.Reason, by) || !strings.Contains(r.Reason, "no open pull request heads it") {
			t.Fatalf("%s in the dry run: %+v; want it deleted, %s", name, r, by)
		}
	}
	kept(t, remoteRowFor(t, dry, "main"), "never deleted: the default branch")
	if dry.ExitCode() != ExitActed {
		t.Fatalf("a dry run that would delete exits %d", dry.ExitCode())
	}

	res := sweepRemoteOK(t, repo, remoteOpts(f), true)
	for _, name := range []string{"feat", "squash"} {
		r := remoteRowFor(t, res, name)
		if r.Outcome != OutcomeDeleted || !strings.Contains(r.Reason, "deleted on origin under a lease on its judged tip") || r.Name != name {
			t.Fatalf("%s: %+v; want it deleted on origin and named", name, r)
		}
		if got := onOrigin(repo, origin, name); got != "" {
			t.Fatalf("origin still holds %s at %s", name, got)
		}
	}
	if !strings.Contains(remoteRowFor(t, res, "feat").Reason, short(tip)) {
		t.Fatalf("the delete names the tip it leased: %+v", remoteRowFor(t, res, "feat"))
	}
	if onOrigin(repo, origin, "main") == "" {
		t.Fatal("origin's main is gone")
	}
	if res.ExitCode() != ExitActed {
		t.Fatalf("an applied run that deleted exits %d", res.ExitCode())
	}
}

// TestABranchPushedToAfterTheLastCheckSurvivesTheLease: a push to the branch
// between the last check and the delete is refused by the lease on the tip
// that was judged; the branch stands at its new tip, kept and no fault.
func TestABranchPushedToAfterTheLastCheckSurvivesTheLease(t *testing.T) {
	repo, origin := originRepo(t)
	tip := mergedOnOrigin(repo, "feat", "feat.txt")
	f := &fakeForge{prs: []pullRequest{mergedPR(4, "feat", tip)}}
	repo.Git("switch", "-q", "feat")
	repo.Write("late.txt", "late\n")
	repo.Git("add", "-A")
	repo.Git("commit", "-q", "-m", "late")
	repo.Git("switch", "-q", "main")
	late := repo.Git("rev-parse", "feat")
	beforeRemoteDelete = func(string) { repo.Git("push", "-q", "--force", "origin", "feat") }
	t.Cleanup(func() { beforeRemoteDelete = nil })

	res := sweepRemoteOK(t, repo, remoteOpts(f), true)
	r := remoteRowFor(t, res, "feat")
	if r.Outcome != OutcomeKept || !strings.Contains(r.Reason, "the lease refused the delete") {
		t.Fatalf("the branch pushed to: %+v; want it kept by the lease", r)
	}
	if got := onOrigin(repo, origin, "feat"); got != late {
		t.Fatalf("origin's feat is %q; want it standing at %s", got, late)
	}
	if res.ExitCode() != ExitNothing {
		t.Fatalf("a run that deleted nothing exits %d", res.ExitCode())
	}
}

// TestAMergedBranchSomethingStillNeedsIsKeptNamingIt is criterion 8's second
// half: a merged branch held by an open pull request from it or targeting it,
// by a reservation (its
// record in the name, or its footprint over a path a worktree tracking the
// branch changes), or by a live session in a worktree tracking it is kept,
// naming what holds it, and stands on origin after an applied run.
func TestAMergedBranchSomethingStillNeedsIsKeptNamingIt(t *testing.T) {
	now := stale()()
	live := func(record string, footprint ...string) claim.Reservation {
		return claim.Reservation{Record: record, Holder: "bob", State: claim.StateLive, Footprint: footprint, EffectiveExpiry: now.Add(time.Hour)}
	}
	cases := []struct {
		name, branch, rel string
		setup             func(*gittest.Repo, *fakeForge, *Options, string)
		want              string
	}{
		{"an open pull request", "feat", "feat.txt", func(_ *gittest.Repo, f *fakeForge, _ *Options, tip string) {
			f.prs = append(f.prs, pullRequest{Number: 12, HeadRefName: "feat", HeadRefOid: tip, BaseRefName: "main", State: "OPEN"})
		}, "pull request #12 is open on it"},
		{"an open pull request targeting it", "feat", "feat.txt", func(_ *gittest.Repo, f *fakeForge, _ *Options, _ string) {
			f.prs = append(f.prs, pullRequest{Number: 13, HeadRefName: "follow-up", HeadRefOid: strings.Repeat("a", 40), BaseRefName: "feat", State: "OPEN"})
		}, "open pull request #13 targets it as its base"},
		{"a reservation naming its record", "fix/iss-2610230000000001-x", "fix.txt", func(_ *gittest.Repo, _ *fakeForge, o *Options, _ string) {
			o.Probe.Reservations.Reservations = []claim.Reservation{live("iss-2610230000000001")}
		}, "reserved as iss-2610230000000001 by bob"},
		{"a reservation's footprint over a worktree tracking it", "feat", "feat.txt", func(repo *gittest.Repo, _ *fakeForge, o *Options, _ string) {
			wt := filepath.Join(o.Cwd, "tracking")
			repo.Git("worktree", "add", "-q", "--track", "-b", "local-feat", wt, "origin/feat")
			if err := os.MkdirAll(filepath.Join(wt, "pkg"), 0o755); err != nil {
				t.Fatal(err)
			}
			write(t, filepath.Join(wt, "pkg", "new.txt"), "new\n")
			o.Probe.Reservations.Reservations = []claim.Reservation{live("itd-2610230000000002", "pkg")}
		}, "its footprint covering pkg/new.txt"},
		{"a live session", "feat", "feat.txt", func(repo *gittest.Repo, _ *fakeForge, o *Options, _ string) {
			wt := filepath.Join(o.Cwd, "tracking")
			repo.Git("worktree", "add", "-q", "--track", "-b", "local-feat", wt, "origin/feat")
			o.Probe.Processes = cwds(wt)
		}, "in use: process 100 works in it"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			repo, origin := originRepo(t)
			branchWith(repo, c.branch, c.rel)
			repo.Git("push", "-q", "origin", c.branch)
			repo.Git("merge", "-q", "--ff-only", c.branch)
			publish(repo)
			tip := repo.Git("rev-parse", c.branch)
			f := &fakeForge{prs: []pullRequest{mergedPR(4, c.branch, tip)}}
			o := remoteOpts(f)
			o.Cwd = t.TempDir()
			c.setup(repo, f, &o, tip)

			res := sweepRemoteOK(t, repo, o, true)
			r := remoteRowFor(t, res, c.branch)
			if r.Action != ActionKeep || r.Outcome != OutcomeKept || !strings.Contains(r.Reason, "still needed") || !strings.Contains(r.Reason, c.want) {
				t.Fatalf("%+v; want it kept, naming %q", r, c.want)
			}
			if onOrigin(repo, origin, c.branch) != tip {
				t.Fatal("origin no longer holds the branch something needs")
			}
		})
	}
}

// TestTheWorktreeTrackingABranchHoldsItOnlyWhileLive: the same worktree with
// no process in it holds nothing, so the branch it tracks is deleted; the
// liveness proof, not the worktree, is what keeps it.
func TestTheWorktreeTrackingABranchHoldsItOnlyWhileLive(t *testing.T) {
	repo, origin := originRepo(t)
	tip := mergedOnOrigin(repo, "feat", "feat.txt")
	repo.Git("worktree", "add", "-q", "--track", "-b", "local-feat", filepath.Join(t.TempDir(), "tracking"), "origin/feat")
	res := sweepRemoteOK(t, repo, remoteOpts(&fakeForge{prs: []pullRequest{mergedPR(4, "feat", tip)}}), true)
	if r := remoteRowFor(t, res, "feat"); r.Outcome != OutcomeDeleted {
		t.Fatalf("%+v; want a branch only an idle worktree tracks deleted", r)
	}
	if onOrigin(repo, origin, "feat") != "" {
		t.Fatal("origin still holds feat")
	}
}

// TestTheLastCheckBeforeTheDeleteAsksTheForgeAgain: a pull request opened
// after the run's listing, seen only at the last check, keeps the branch.
func TestTheLastCheckBeforeTheDeleteAsksTheForgeAgain(t *testing.T) {
	repo, origin := originRepo(t)
	tip := mergedOnOrigin(repo, "feat", "feat.txt")
	f := &fakeForge{prs: []pullRequest{mergedPR(4, "feat", tip)}}
	f.recheck = append(f.prs, pullRequest{Number: 9, HeadRefName: "feat", HeadRefOid: tip, BaseRefName: "main", State: "OPEN"})
	res := sweepRemoteOK(t, repo, remoteOpts(f), true)
	r := remoteRowFor(t, res, "feat")
	if r.Outcome != OutcomeKept || !strings.Contains(r.Reason, "held at the last check before the delete") || !strings.Contains(r.Reason, "pull request #9 is open on it") {
		t.Fatalf("%+v; want it kept at the last check, naming the pull request", r)
	}
	if onOrigin(repo, origin, "feat") != tip {
		t.Fatal("origin no longer holds feat")
	}
}

// TestTheNeverDeleteList: the default branch, a release branch, a reservation
// branch, a status branch and a branch the sweep cannot show merged are kept
// even when the forge records each merged at its tip, and an applied run
// leaves every one of them, and a tag, on origin. With no forge, no branch is
// deleted, since no open pull request can be shown absent.
func TestTheNeverDeleteList(t *testing.T) {
	repo, origin := originRepo(t)
	var prs []pullRequest
	for i, b := range []string{"release/v1.0", "abcd-reserved/iss-2610230000000003", "abcd-status/bob"} {
		prs = append(prs, mergedPR(i+1, b, mergedOnOrigin(repo, b, strings.ReplaceAll(b, "/", "-")+".txt")))
	}
	branchWith(repo, "open-work", "w.txt")
	repo.Git("push", "-q", "origin", "open-work")
	repo.Git("tag", "v1.0", "main")
	repo.Git("push", "-q", "origin", "v1.0")
	repo.Git("fetch", "-q", "origin")
	f := &fakeForge{prs: prs}
	before := repo.Git("--git-dir", origin, "for-each-ref")

	res := sweepRemoteOK(t, repo, remoteOpts(f), true)
	for name, want := range map[string]string{
		"main":                               "never deleted: the default branch",
		"release/v1.0":                       "never deleted: a release branch",
		"abcd-reserved/iss-2610230000000003": "never deleted: a reservation branch",
		"abcd-status/bob":                    "never deleted: a status branch",
		"open-work":                          "never deleted: the sweep cannot show it is merged",
	} {
		r := remoteRowFor(t, res, name)
		kept(t, r, want)
		if r.Outcome != OutcomeKept {
			t.Errorf("%s: %+v; want it kept", name, r)
		}
	}
	if len(res.Rows) != 5 {
		t.Fatalf("one row per branch on origin, and none for a tag or origin/HEAD: %+v", res.Rows)
	}
	if after := repo.Git("--git-dir", origin, "for-each-ref"); after != before {
		t.Fatalf("origin's refs changed:\n%s\n->\n%s", before, after)
	}

	branchWith(repo, "feat", "feat.txt")
	repo.Git("push", "-q", "origin", "feat")
	repo.Git("merge", "-q", "--squash", "feat")
	repo.Git("commit", "-q", "-m", "squash feat")
	publish(repo)
	res = sweepRemoteOK(t, repo, Options{Probe: idleProbe()}, true)
	r := remoteRowFor(t, res, "feat")
	kept(t, r, "no open pull request can be shown absent")
	if !strings.Contains(res.ForgeNote, "no branch on origin is deleted") || onOrigin(repo, origin, "feat") == "" {
		t.Fatalf("with no forge nothing is deleted, and the run says so once: %q, %+v", res.ForgeNote, r)
	}
}

// TestAFailedDeleteOnOriginIsTheRunsFault: origin refusing the delete for a
// reason other than the lease keeps the branch, names origin's refusal with
// the home path redacted, and exits as a fault.
func TestAFailedDeleteOnOriginIsTheRunsFault(t *testing.T) {
	repo, origin := originRepo(t)
	tip := mergedOnOrigin(repo, "feat", "feat.txt")
	repo.Git("--git-dir", origin, "config", "receive.denyDeletes", "true")
	res, err := sweepRemote(context.Background(), repo.Root(), remoteOpts(&fakeForge{prs: []pullRequest{mergedPR(4, "feat", tip)}}), true)
	if err == nil || res.ExitCode() != ExitRefused || res.Refusal == "" {
		t.Fatalf("a refused delete is the run's fault: err %v, exit %d", err, res.ExitCode())
	}
	r := remoteRowFor(t, res, "feat")
	if r.Outcome != OutcomeKept || !strings.Contains(r.Reason, "the delete on origin failed") {
		t.Fatalf("%+v; want it kept, naming origin's refusal", r)
	}
	home := filepath.Dir(origin)
	if strings.Contains(r.Reason, home) || strings.Contains(res.Refusal, home) {
		t.Fatalf("origin's refusal names the home path unredacted: %q / %q", r.Reason, res.Refusal)
	}
	if onOrigin(repo, origin, "feat") != tip {
		t.Fatal("origin no longer holds feat")
	}
}

// TestTheDeleteGoesThroughThePushTheRepositorysOwnHooksSee: a pre-push hook
// of the checkout that refuses deleting the branch is honoured: the branch
// stands on origin, its row carries the refusal, and the run is a fault, as
// for any other failed push.
func TestTheDeleteGoesThroughThePushTheRepositorysOwnHooksSee(t *testing.T) {
	repo, origin := originRepo(t)
	tip := mergedOnOrigin(repo, "feat", "feat.txt")
	hooks := repo.Git("rev-parse", "--git-path", "hooks")
	if !filepath.IsAbs(hooks) {
		hooks = filepath.Join(repo.Root(), hooks)
	}
	if err := os.MkdirAll(hooks, 0o755); err != nil {
		t.Fatal(err)
	}
	hook := "#!/bin/sh\nwhile read local lsha remote rsha; do\n" +
		"  case \"$lsha\" in 0000000000000000000000000000000000000000) echo \"pre-push: this checkout keeps $remote\" >&2; exit 1;; esac\n" +
		"done\nexit 0\n"
	if err := os.WriteFile(filepath.Join(hooks, "pre-push"), []byte(hook), 0o755); err != nil {
		t.Fatal(err)
	}
	res, err := sweepRemote(context.Background(), repo.Root(), remoteOpts(&fakeForge{prs: []pullRequest{mergedPR(4, "feat", tip)}}), true)
	if err == nil || res.ExitCode() != ExitRefused {
		t.Fatalf("a delete the hook refuses is the run's fault: err %v, exit %d", err, res.ExitCode())
	}
	r := remoteRowFor(t, res, "feat")
	if r.Outcome != OutcomeKept || !strings.Contains(r.Reason, "the delete on origin failed") || !strings.Contains(r.Reason, "this checkout keeps refs/heads/feat") {
		t.Fatalf("%+v; want it kept, naming the hook's refusal", r)
	}
	if onOrigin(repo, origin, "feat") != tip {
		t.Fatal("origin no longer holds feat")
	}
}

// TestOriginsAnswerIsScrubbedBeforeItIsShown: when origin's URL carries a
// user and a password and the last check's ls-remote fails, the row's reason
// and the tip reader's error name neither.
func TestOriginsAnswerIsScrubbedBeforeItIsShown(t *testing.T) {
	repo, _ := originRepo(t)
	tip := mergedOnOrigin(repo, "feat", "feat.txt")
	// Joined by hand: filepath.Join would fold the URL's "//" into one, and
	// git quotes the path back with the URL whole.
	url := "file://" + t.TempDir() + "/https://user:s3cret@example.invalid/none.git"
	repo.Git("remote", "set-url", "origin", url)
	hostedOrigin(t, url)

	_, _, err := lsRemoteTipContext(context.Background(), repo.Root(), "origin", "feat")
	if err == nil || strings.Contains(err.Error(), "s3cret") || strings.Contains(err.Error(), "user:") {
		t.Fatalf("the tip reader's error: %v; want a failure naming neither the user nor the password", err)
	}
	res := sweepRemoteOK(t, repo, remoteOpts(&fakeForge{prs: []pullRequest{mergedPR(4, "feat", tip)}}), true)
	r := remoteRowFor(t, res, "feat")
	if r.Outcome != OutcomeKept || !strings.Contains(r.Reason, "held at the last check before the delete") {
		t.Fatalf("%+v; want it kept at the last check", r)
	}
	if strings.Contains(r.Reason, "s3cret") || strings.Contains(r.Reason, "user:") {
		t.Fatalf("the row's reason names origin's user or password: %q", r.Reason)
	}
}

// TestADefaultBranchThatMovedOnOriginHoldsEveryDelete: when origin's default
// branch has moved past the tip this checkout last fetched, the last check
// holds the delete and asks for a fetch, though the branch's name matches.
func TestADefaultBranchThatMovedOnOriginHoldsEveryDelete(t *testing.T) {
	repo, origin := originRepo(t)
	tip := mergedOnOrigin(repo, "feat", "feat.txt")
	fetched := repo.Git("rev-parse", "refs/remotes/origin/main")
	// Another clone moves origin's main, as another writer would. A push from
	// this checkout cannot stand in for it, even one to origin's path rather
	// than its name: since git 2.56 a push to a URL that matches one
	// configured remote updates that remote's tracking ref too, which moves
	// origin/main here with origin's main.
	other := filepath.Join(t.TempDir(), "other")
	repo.Git("clone", "-q", origin, other)
	write(t, filepath.Join(other, "later.txt"), "later\n")
	repo.Git("-C", other, "add", "-A")
	repo.Git("-C", other, "commit", "-q", "-m", "later")
	repo.Git("-C", other, "push", "-q", "origin", "main")
	if moved := onOrigin(repo, origin, "main"); moved == fetched || repo.Git("rev-parse", "refs/remotes/origin/main") != fetched {
		t.Fatalf("the fixture: origin's main is at %s and origin/main here at %s; want origin moved past %s and origin/main left there",
			moved, repo.Git("rev-parse", "refs/remotes/origin/main"), fetched)
	}

	res := sweepRemoteOK(t, repo, remoteOpts(&fakeForge{prs: []pullRequest{mergedPR(4, "feat", tip)}}), true)
	r := remoteRowFor(t, res, "feat")
	moved := onOrigin(repo, origin, "main")
	// The reason names the tip origin's default branch moved to, which a
	// renamed default branch's reason does not.
	if r.Outcome != OutcomeKept || !strings.Contains(r.Reason, "fetch origin and run the sweep again") ||
		!strings.Contains(r.Reason, "origin's default branch is at "+short(moved)+", but it was judged against origin/main at "+short(fetched)) {
		t.Fatalf("%+v; want it held until a fetch, naming origin's main at %s against %s", r, short(moved), short(fetched))
	}
	if onOrigin(repo, origin, "feat") != tip {
		t.Fatal("origin no longer holds feat")
	}
}

// TestADefaultBranchThatMovesOnOriginBetweenDeletesHoldsTheLaterOne: the last
// check before each delete asks origin for its default branch again, so when
// another clone moves origin's main after the first of two deletes passed its
// last check, the first is deleted and the second is held until a fetch. The
// question to origin after the first delete sees the move too, so the first
// row notes how to restore that branch; that is no fault.
func TestADefaultBranchThatMovesOnOriginBetweenDeletesHoldsTheLaterOne(t *testing.T) {
	repo, origin := originRepo(t)
	first := mergedOnOrigin(repo, "first", "first.txt")
	second := mergedOnOrigin(repo, "second", "second.txt")
	fetched := repo.Git("rev-parse", "refs/remotes/origin/main")
	// Another clone moves origin's main, pushing by the remote's name; see
	// TestADefaultBranchThatMovedOnOriginHoldsEveryDelete for why a push from
	// this checkout cannot stand in for it.
	other := filepath.Join(t.TempDir(), "other")
	repo.Git("clone", "-q", origin, other)
	write(t, filepath.Join(other, "later.txt"), "later\n")
	repo.Git("-C", other, "add", "-A")
	repo.Git("-C", other, "commit", "-q", "-m", "later")
	var hooked []string
	beforeRemoteDelete = func(ref string) {
		hooked = append(hooked, ref)
		if ref == "refs/heads/first" {
			repo.Git("-C", other, "push", "-q", "origin", "main")
		}
	}
	t.Cleanup(func() { beforeRemoteDelete = nil })
	f := &fakeForge{prs: []pullRequest{mergedPR(4, "first", first), mergedPR(5, "second", second)}}

	res, err := sweepRemote(context.Background(), repo.Root(), remoteOpts(f), true)
	moved := onOrigin(repo, origin, "main")
	if moved == fetched || repo.Git("rev-parse", "refs/remotes/origin/main") != fetched {
		t.Fatalf("the fixture: origin's main is at %s and origin/main here at %s; want origin moved past %s and origin/main left there",
			moved, repo.Git("rev-parse", "refs/remotes/origin/main"), fetched)
	}
	if len(hooked) == 0 || hooked[0] != "refs/heads/first" {
		t.Fatalf("the deletes reached the push in the order %v; want first's first", hooked)
	}
	restore := "git push origin " + first + ":refs/heads/first"
	if r := remoteRowFor(t, res, "first"); r.Outcome != OutcomeDeleted || !strings.Contains(r.Reason, "`"+restore+"` restores it") {
		t.Fatalf("first: %+v; want it deleted as origin's main moved, naming %q", r, restore)
	}
	if got := onOrigin(repo, origin, "first"); got != "" {
		t.Fatalf("origin still holds first at %s", got)
	}
	r := remoteRowFor(t, res, "second")
	if r.Action != ActionDelete || r.Outcome != OutcomeKept || !strings.Contains(r.Reason, "held at the last check before the delete") ||
		!strings.Contains(r.Reason, "origin's default branch is at "+short(moved)+", but it was judged against origin/main at "+short(fetched)) ||
		!strings.Contains(r.Reason, "fetch origin and run the sweep again") {
		t.Fatalf("second: %+v; want it judged deletable, then held until a fetch at the last check", r)
	}
	if onOrigin(repo, origin, "second") != second {
		t.Fatal("origin no longer holds second")
	}
	if err != nil || res.Refusal != "" || res.ExitCode() != ExitActed {
		t.Fatalf("a delete origin's main moved under is a note, no fault: err %v, exit %d, refusal %q", err, res.ExitCode(), res.Refusal)
	}
}

// TestADefaultBranchOriginCannotNameAtTheLastCheckHoldsTheDelete: when origin
// answers the run's first question about its default branch but not the last
// check's, because its HEAD came to name a branch it does not hold, the delete
// is held and the branch stands on origin.
func TestADefaultBranchOriginCannotNameAtTheLastCheckHoldsTheDelete(t *testing.T) {
	repo, origin := originRepo(t)
	tip := mergedOnOrigin(repo, "feat", "feat.txt")
	f := &fakeForge{prs: []pullRequest{mergedPR(4, "feat", tip)}}
	// The forge's per-branch listing comes after the run's first question to
	// origin and before the last check's.
	f.onRecheck = func() { repo.Git("--git-dir", origin, "symbolic-ref", "HEAD", "refs/heads/gone") }

	res := sweepRemoteOK(t, repo, remoteOpts(f), true)
	r := remoteRowFor(t, res, "feat")
	if r.Action != ActionDelete || r.Outcome != OutcomeKept || !strings.Contains(r.Reason, "held at the last check before the delete") ||
		!strings.Contains(r.Reason, "origin's default branch could not be asked") {
		t.Fatalf("%+v; want it judged deletable, then held because origin's default branch could not be asked", r)
	}
	if onOrigin(repo, origin, "feat") != tip {
		t.Fatal("origin no longer holds feat")
	}
}

// TestAWorktreeThatBeginsTrackingTheBranchBeforeTheDeleteHoldsIt: a worktree
// that starts tracking the branch under another local name after the
// judgement, with a process working in it, is seen at the last check, and the
// branch is kept, naming that process.
func TestAWorktreeThatBeginsTrackingTheBranchBeforeTheDeleteHoldsIt(t *testing.T) {
	repo, origin := originRepo(t)
	tip := mergedOnOrigin(repo, "feat", "feat.txt")
	wt := filepath.Join(t.TempDir(), "late")
	f := &fakeForge{prs: []pullRequest{mergedPR(4, "feat", tip)}}
	f.onRecheck = func() { repo.Git("worktree", "add", "-q", "--track", "-b", "late-feat", wt, "origin/feat") }
	o := remoteOpts(f)
	o.Probe.Processes = cwds(wt)

	res := sweepRemoteOK(t, repo, o, true)
	r := remoteRowFor(t, res, "feat")
	if r.Action != ActionDelete || r.Outcome != OutcomeKept || !strings.Contains(r.Reason, "held at the last check before the delete") ||
		!strings.Contains(r.Reason, "in use: process 100 works in it") {
		t.Fatalf("%+v; want it judged deletable, then held at the last check by the new worktree's process", r)
	}
	if onOrigin(repo, origin, "feat") != tip {
		t.Fatal("origin no longer holds feat")
	}
}

// TestAPullRequestTargetingTheBranchSeenOnlyAtTheLastCheckHoldsIt: an open
// pull request that comes to target the branch after the run's listing is
// seen by the last check, and the branch is kept.
func TestAPullRequestTargetingTheBranchSeenOnlyAtTheLastCheckHoldsIt(t *testing.T) {
	repo, origin := originRepo(t)
	tip := mergedOnOrigin(repo, "feat", "feat.txt")
	f := &fakeForge{prs: []pullRequest{mergedPR(4, "feat", tip)}}
	f.recheck = append(f.prs, pullRequest{Number: 21, HeadRefName: "follow-up", HeadRefOid: strings.Repeat("b", 40), BaseRefName: "feat", State: "OPEN"})
	f.onRecheck = func() { f.prs = f.recheck }
	res := sweepRemoteOK(t, repo, remoteOpts(f), true)
	r := remoteRowFor(t, res, "feat")
	if r.Outcome != OutcomeKept || !strings.Contains(r.Reason, "held at the last check before the delete") || !strings.Contains(r.Reason, "open pull request #21 targets it") {
		t.Fatalf("%+v; want it held at the last check, naming the pull request", r)
	}
	if onOrigin(repo, origin, "feat") != tip {
		t.Fatal("origin no longer holds feat")
	}
}

// TestARunWhoseContextEndsStopsJudging: a context that ends while the run is
// under way refuses the run before any branch is judged deletable.
func TestARunWhoseContextEndsStopsJudging(t *testing.T) {
	repo, origin := originRepo(t)
	tip := mergedOnOrigin(repo, "feat", "feat.txt")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := &fakeForge{prs: []pullRequest{mergedPR(4, "feat", tip)}}
	o := remoteOpts(f)
	o.Forge = func(dir string, args ...string) (string, error) {
		cancel()
		return f.gh(dir, args...)
	}
	res, err := sweepRemote(ctx, repo.Root(), o, true)
	if err == nil || res.ExitCode() != ExitRefused || len(res.Rows) != 0 {
		t.Fatalf("a stopped run refuses: err %v, exit %d, rows %+v", err, res.ExitCode(), res.Rows)
	}
	if onOrigin(repo, origin, "feat") != tip {
		t.Fatal("origin no longer holds feat")
	}
}

// TestAPushAddressOtherThanOriginsFetchAddressHoldsEveryDelete: with a push
// URL configured for origin that names another repository, the push would
// change a repository other than the one the sweep reads, so origin is not
// one repository: the forge is asked nothing, every branch is kept, naming
// why, in the dry run and under apply, and neither repository loses the
// branch.
func TestAPushAddressOtherThanOriginsFetchAddressHoldsEveryDelete(t *testing.T) {
	repo, origin := originRepo(t)
	tip := squashedOnOrigin(repo, "feat", "feat.txt")
	mirror := filepath.Join(t.TempDir(), "mirror.git")
	repo.Git("clone", "-q", "--bare", origin, mirror)
	repo.Git("config", "remote.origin.pushurl", mirror)
	f := &fakeForge{}

	for _, apply := range []bool{false, true} {
		res := sweepRemoteOK(t, repo, remoteOpts(f), apply)
		r := remoteRowFor(t, res, "feat")
		if r.Action != ActionKeep || !strings.Contains(r.Reason, "the forge's repository could not be matched to origin") ||
			!strings.Contains(r.Reason, "origin is read from one repository and pushed to another") {
			t.Fatalf("apply %v: %+v; want it kept because origin's fetch and push addresses name two repositories", apply, r)
		}
		if strings.Contains(r.Reason, mirror) || strings.Contains(r.Reason, origin) {
			t.Fatalf("the reason shows an address: %q", r.Reason)
		}
		if res.Refusal != "" || res.ExitCode() != ExitNothing {
			t.Fatalf("a kept branch is no fault: refusal %q, exit %d", res.Refusal, res.ExitCode())
		}
	}
	if len(f.calls) != 0 {
		t.Fatalf("the forge was asked %v", f.calls)
	}
	if onOrigin(repo, origin, "feat") != tip || onOrigin(repo, mirror, "feat") != tip {
		t.Fatal("a repository lost feat")
	}
}

// TestARunStoppedWhileOriginsTipIsReadIsReportedStopped: a run whose context
// ends while the last check reads origin's tip of the branch is reported as
// stopped, the run's fault, not as a branch whose tip could not be read; the
// branch stands on origin.
func TestARunStoppedWhileOriginsTipIsReadIsReportedStopped(t *testing.T) {
	repo, origin := originRepo(t)
	tip := mergedOnOrigin(repo, "feat", "feat.txt")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	readRemoteTip = func(rctx context.Context, dir, remote, branch string) (string, bool, error) {
		cancel()
		return lsRemoteTipContext(rctx, dir, remote, branch)
	}
	t.Cleanup(func() { readRemoteTip = lsRemoteTipContext })
	f := &fakeForge{prs: []pullRequest{mergedPR(4, "feat", tip)}}

	res, err := sweepRemote(ctx, repo.Root(), remoteOpts(f), true)
	r := remoteRowFor(t, res, "feat")
	if err == nil || res.ExitCode() != ExitRefused || !strings.Contains(res.Refusal, "the run was stopped before every delete was made") {
		t.Fatalf("a run stopped at the last check is the run's fault: err %v, exit %d, refusal %q", err, res.ExitCode(), res.Refusal)
	}
	if r.Outcome != OutcomeKept || r.Reason != "the run was stopped before the delete, so the branch is kept" {
		t.Fatalf("%+v; want it kept as stopped, not held on an unread tip", r)
	}
	if onOrigin(repo, origin, "feat") != tip {
		t.Fatal("origin no longer holds feat")
	}
}

// TestADefaultBranchThatMovesAfterADeleteNamesTheRestore: when another clone
// moves origin's main after origin took a delete, as an ordinary merge does,
// the question to origin after the delete sees it; the outcome stays deleted,
// and the row notes the judged tip and the exact command that restores the
// branch, with no fault. The command, run as printed, puts the branch back.
func TestADefaultBranchThatMovesAfterADeleteNamesTheRestore(t *testing.T) {
	repo, origin := originRepo(t)
	tip := mergedOnOrigin(repo, "feat", "feat.txt")
	fetched := repo.Git("rev-parse", "refs/remotes/origin/main")
	// Another clone moves origin's main, pushing by the remote's name; see
	// TestADefaultBranchThatMovedOnOriginHoldsEveryDelete for why a push from
	// this checkout cannot stand in for it.
	other := filepath.Join(t.TempDir(), "other")
	repo.Git("clone", "-q", origin, other)
	write(t, filepath.Join(other, "later.txt"), "later\n")
	repo.Git("-C", other, "add", "-A")
	repo.Git("-C", other, "commit", "-q", "-m", "later")
	afterRemoteDelete = func(string) { repo.Git("-C", other, "push", "-q", "origin", "main") }
	t.Cleanup(func() { afterRemoteDelete = nil })
	f := &fakeForge{prs: []pullRequest{mergedPR(4, "feat", tip)}}

	res, err := sweepRemote(context.Background(), repo.Root(), remoteOpts(f), true)
	moved := onOrigin(repo, origin, "main")
	if moved == fetched {
		t.Fatal("the fixture: origin's main did not move")
	}
	r := remoteRowFor(t, res, "feat")
	restore := "git push origin " + tip + ":refs/heads/feat"
	if r.Outcome != OutcomeDeleted ||
		!strings.Contains(r.Reason, "; note: origin's default branch is main at "+short(moved)+" after the delete, but the branch was judged against origin/main at "+short(fetched)) ||
		!strings.Contains(r.Reason, "in case the branch's work is no longer on main, its judged tip was "+tip+", and `"+restore+"` restores it") {
		t.Fatalf("%+v; want it deleted, noting its judged tip and %q", r, restore)
	}
	if err != nil || res.Refusal != "" || res.ExitCode() != ExitActed {
		t.Fatalf("a delete origin's default branch moved after is a note, no fault: err %v, exit %d, refusal %q", err, res.ExitCode(), res.Refusal)
	}
	if onOrigin(repo, origin, "feat") != "" {
		t.Fatal("origin still holds feat")
	}
	repo.Git(strings.Fields(restore)[1:]...)
	if onOrigin(repo, origin, "feat") != tip {
		t.Fatalf("%q did not put feat back at %s", restore, tip)
	}
}

// TestAnUnansweredQuestionAfterADeleteIsANoteNotAFault: when origin cannot
// name its default branch after it took a delete, the row says origin could
// not be asked after the delete, prints no restore line, and the run does
// not fault.
func TestAnUnansweredQuestionAfterADeleteIsANoteNotAFault(t *testing.T) {
	repo, origin := originRepo(t)
	tip := mergedOnOrigin(repo, "feat", "feat.txt")
	afterRemoteDelete = func(string) { repo.Git("--git-dir", origin, "symbolic-ref", "HEAD", "refs/heads/gone") }
	t.Cleanup(func() { afterRemoteDelete = nil })
	f := &fakeForge{prs: []pullRequest{mergedPR(4, "feat", tip)}}

	res, err := sweepRemote(context.Background(), repo.Root(), remoteOpts(f), true)
	r := remoteRowFor(t, res, "feat")
	if r.Outcome != OutcomeDeleted || !strings.Contains(r.Reason, "origin could not be asked for its default branch after the delete") ||
		strings.Contains(r.Reason, "git push") {
		t.Fatalf("%+v; want it deleted, with a note that origin could not be asked and no restore line", r)
	}
	if err != nil || res.Refusal != "" || res.ExitCode() != ExitActed {
		t.Fatalf("an unanswered question after a delete is no fault: err %v, refusal %q, exit %d", err, res.Refusal, res.ExitCode())
	}
	if onOrigin(repo, origin, "feat") != "" {
		t.Fatal("origin still holds feat")
	}
}

// shellWords splits a printed command as a POSIX shell reads its words:
// blanks separate words, single quotes hold everything literally, and a
// backslash outside them takes the next character literally. Any other
// character a shell acts on unquoted (; | & < > ( ) $ ` " and the like) is
// refused, so a line this reads is words and nothing else.
func shellWords(t *testing.T, line string) []string {
	t.Helper()
	var (
		words []string
		cur   strings.Builder
		in    bool
	)
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case c == '\'':
			end := strings.IndexByte(line[i+1:], '\'')
			if end < 0 {
				t.Fatalf("an unclosed quote in %q", line)
			}
			cur.WriteString(line[i+1 : i+1+end])
			i += end + 1
			in = true
		case c == '\\' && i+1 < len(line):
			cur.WriteByte(line[i+1])
			i++
			in = true
		case c == ' ':
			if in {
				words = append(words, cur.String())
				cur.Reset()
				in = false
			}
		case strings.IndexByte(";|&<>()$`\"!{}#*?[]~\t\n", c) >= 0:
			t.Fatalf("%q holds %q unquoted", line, c)
		default:
			cur.WriteByte(c)
			in = true
		}
	}
	if in {
		words = append(words, cur.String())
	}
	return words
}

// TestTheRestoreCommandIsOneWordPerArgument: whatever a branch name holds
// that git allows, the printed restore command reads back as exactly
// `git push origin <tip>:refs/heads/<name>`, the refspec one shell word; a
// name holding a backquote or a character outside printable ASCII gets no
// command.
func TestTheRestoreCommandIsOneWordPerArgument(t *testing.T) {
	sha := strings.Repeat("a", 40)
	for _, name := range []string{"plain/name-1.2", "a;b", "a|b", "$(touch x)", "a&b", "it's", `say"hi"`, "a!b", "{x,y}", "a#b", "x>y<z", "end'"} {
		ref := "refs/heads/" + name
		cmd, ok := remoteRestoreCommand(ref, sha)
		if !ok {
			t.Fatalf("%q: no command", name)
		}
		got := shellWords(t, cmd)
		want := []string{"git", "push", "origin", sha + ":" + ref}
		if !slices.Equal(got, want) {
			t.Fatalf("%q: %q reads as %q; want %q", name, cmd, got, want)
		}
	}
	if cmd, _ := remoteRestoreCommand("refs/heads/plain/name-1.2", sha); cmd != "git push origin "+sha+":refs/heads/plain/name-1.2" {
		t.Fatalf("a plain name is quoted: %q", cmd)
	}
	for _, name := range []string{"with`tick", "café", "bell\x07"} {
		if cmd, ok := remoteRestoreCommand("refs/heads/"+name, sha); ok || cmd != "" {
			t.Fatalf("%q: printed %q; want no command", name, cmd)
		}
	}
}

// TestARestoreNoteForANameNoCommandCarries: when origin's default branch
// moves after the delete of a branch whose name holds a backquote, the note
// gives the judged tip and says to quote the name by hand, with no command.
func TestARestoreNoteForANameNoCommandCarries(t *testing.T) {
	repo, origin := originRepo(t)
	name := "odd`name"
	tip := mergedOnOrigin(repo, name, "odd.txt")
	other := filepath.Join(t.TempDir(), "other")
	repo.Git("clone", "-q", origin, other)
	write(t, filepath.Join(other, "later.txt"), "later\n")
	repo.Git("-C", other, "add", "-A")
	repo.Git("-C", other, "commit", "-q", "-m", "later")
	afterRemoteDelete = func(string) { repo.Git("-C", other, "push", "-q", "origin", "main") }
	t.Cleanup(func() { afterRemoteDelete = nil })

	res := sweepRemoteOK(t, repo, remoteOpts(&fakeForge{prs: []pullRequest{mergedPR(4, name, tip)}}), true)
	r := remoteRowFor(t, res, name)
	if r.Outcome != OutcomeDeleted || !strings.Contains(r.Reason, "its judged tip was "+tip+"; its name holds a character no printed command carries safely, so quote the name by hand") ||
		strings.Contains(r.Reason, "git push") {
		t.Fatalf("%+v; want the judged tip and no command", r)
	}
}
