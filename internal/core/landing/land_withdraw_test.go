package landing

// Before land's push moves a pull request's head, land withdraws the arming at
// that head, reads it back and says so. Where it cannot, it refuses and pushes
// nothing.

import (
	"errors"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// armingBy is one pull request from the branch as ArmingsQuery answers it,
// armed (auto) or queued (queued) by the account by; "" attributes it to none.
func armingBy(head string, auto, queued bool, by string) string {
	who := "null"
	if by != "" {
		who = `{"login":"` + by + `"}`
	}
	arm, entry := "null", "null"
	if auto {
		arm = `{"mergeMethod":"MERGE","enabledBy":` + who + `}`
	}
	q := "false"
	if queued {
		q = "true"
		entry = `{"enqueuer":` + who + `}`
	}
	return `[{"number":7,"id":"PR_7","headRefOid":"` + head + `","isCrossRepository":false,"isInMergeQueue":` + q + `,"mergeQueueEntry":` + entry + `,"autoMergeRequest":` + arm + `}]`
}

var dequeue7 = []string{"api", "graphql", "-f", "query=" + claim.DequeueMutation, "-f", "id=PR_7"}

// moving is a fixture in auto mode whose branch is on origin at old, with one
// local commit past it: a push moves the pull request's head from old. The
// listing shows the local head, as the forge does after the push.
func moving(t *testing.T) (f *fixture, old string) {
	t.Helper()
	f = newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": autoConfig})
	old = f.head()
	f.repo.Git("push", "-q", "origin", "feat/x")
	f.repo.Write("more.txt", "more\n")
	f.repo.Commit("more")
	f.forge.prs = f.prFor(f.head())
	f.forge.views = []string{viewBlocked, viewQueued}
	return f, old
}

// reviewedAt writes the review receipt for sha.
func (f *fixture) reviewedAt(t *testing.T, sha string) {
	t.Helper()
	head := f.head()
	f.repo.Git("checkout", "-q", sha)
	f.reviewed(t)
	f.repo.Git("checkout", "-q", "feat/x")
	if f.head() != head {
		t.Fatalf("head moved from %s to %s", head, f.head())
	}
}

// withdrawnAtPush records how many calls beginning with prefix the forge had
// seen when the landing pushed.
func withdrawnAtPush(f *fixture, prefix ...string) *int {
	seen := -1
	f.beforeGit = func(args []string) {
		if len(args) > 0 && args[0] == "push" {
			seen = f.forge.called(prefix...)
		}
	}
	return &seen
}

func noWithdrawal(t *testing.T, f *fixture) {
	t.Helper()
	if f.forge.called("pr", "merge", "7", "--disable-auto") != 0 || f.forge.called(dequeue7...) != 0 {
		t.Fatalf("calls %v: want the arming left as it is", f.forge.calls)
	}
}

func TestLandWithdrawsItsOwnArmingAtTheHeadItsPushMovesAndArmsTheNewOne(t *testing.T) {
	f, old := moving(t)
	f.reviewed(t)
	f.forge.armings = []string{armingBy(old, true, false, "alice"), "[]"}
	seen := withdrawnAtPush(f, "pr", "merge", "7", "--disable-auto")
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	if *seen != 1 || f.onOrigin("feat/x") != f.head() {
		t.Fatalf("withdrawals at the push %d, origin %q; calls %v", *seen, f.onOrigin("feat/x"), f.forge.calls)
	}
	s := step(t, res, "withdraw")
	for _, want := range []string{"#7", short(old), short(f.head()), "withdrew"} {
		if !strings.Contains(s.Note, want) || s.Status != StatusDone {
			t.Fatalf("withdraw step %+v does not say %q", s, want)
		}
	}
	if f.forge.called("pr", "merge", "7", "--auto", "--merge", "--match-head-commit="+f.head()) != 1 {
		t.Fatalf("calls %v: want the reviewed local head armed, pinned", f.forge.calls)
	}
}

func TestLandTakesItsOwnQueueEntryAtTheHeadItsPushMovesOutOfTheQueue(t *testing.T) {
	f, old := moving(t)
	f.reviewed(t)
	f.forge.armings = []string{armingBy(old, false, true, "alice"), "[]"}
	seen := withdrawnAtPush(f, dequeue7...)
	if _, err := f.land(f.options()); err != nil {
		t.Fatal(err)
	}
	if *seen != 1 || f.forge.called("pr", "merge", "7", "--disable-auto") != 0 {
		t.Fatalf("dequeues at the push %d; calls %v", *seen, f.forge.calls)
	}
}

func TestLandTurnsAutoMergeOffBeforeItDequeues(t *testing.T) {
	f, old := moving(t)
	f.reviewed(t)
	f.forge.armings = []string{armingBy(old, true, true, "alice"), "[]"}
	if _, err := f.land(f.options()); err != nil {
		t.Fatal(err)
	}
	off, out := -1, -1
	for i, c := range f.forge.calls {
		switch strings.Join(c, " ") {
		case "pr merge 7 --disable-auto":
			off = i
		case strings.Join(dequeue7, " "):
			out = i
		}
	}
	if off < 0 || out < off {
		t.Fatalf("auto-merge off at call %d, out of the queue at %d: %v", off, out, f.forge.calls)
	}
}

// The enqueue step reports an arming already at the local head.
func TestLandReportsAnArmingAtTheLocalHead(t *testing.T) {
	cases := []struct {
		name, config, by string
		set              func(f *fixture)
		says             string
	}{
		{"its own, with no receipt", autoConfig, "alice", nil, "no review receipt names"},
		{"another account's, unapproved in human mode", humanConfig, "someone-else", func(f *fixture) { f.forge.reviews = "[]" }, "waits for the technical facilitator's approval"},
		{"the author's permission read fails", autoConfig, "alice", func(f *fixture) {
			f.forge.roles = map[string]string{"author-a": "!gh: HTTP 502: Bad Gateway"}
		}, "unverified"},
		{"the pull request read fails", humanConfig, "alice", func(f *fixture) {
			f.forge.pullErr = errors.New("gh: HTTP 502: Bad Gateway")
		}, "unverified"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": c.config})
			f.forge.prs = f.prFor(f.head())
			f.forge.armings = []string{armingBy(f.head(), true, false, c.by)}
			if c.set != nil {
				c.set(f)
			}
			res, err := f.land(f.options())
			if err != nil {
				t.Fatal(err)
			}
			noWithdrawal(t, f)
			s := step(t, res, "enqueue")
			if !strings.Contains(s.Note, c.says) || !strings.Contains(s.Note, "is left as it is") {
				t.Fatalf("enqueue step %+v: want %q and the arming said left", s, c.says)
			}
		})
	}
}

func TestLandLeavesAnArmingAtAReviewedLocalHead(t *testing.T) {
	f := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": autoConfig})
	f.reviewed(t)
	f.forge.prs = f.prFor(f.head())
	f.forge.armings = []string{armingBy(f.head(), true, false, "alice")}
	f.forge.views = []string{viewQueued}
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	noWithdrawal(t, f)
	for _, s := range res.Steps {
		if s.Name == "withdraw" {
			t.Fatalf("a withdraw step reported for nothing withdrawn: %+v", s)
		}
	}
}

func TestLandLeavesAForksPullRequestFromABranchOfTheSameName(t *testing.T) {
	f, _ := moving(t)
	f.reviewed(t)
	f.forge.armings = []string{`[{"number":8,"id":"PR_8","headRefOid":"` + strings.Repeat("c", 40) + `","isCrossRepository":true,"isInMergeQueue":false,"mergeQueueEntry":null,"autoMergeRequest":{"mergeMethod":"MERGE","enabledBy":{"login":"alice"}}}]`}
	if _, err := f.land(f.options()); err != nil {
		t.Fatal(err)
	}
	if f.forge.called("pr", "merge", "8") != 0 {
		t.Fatalf("calls %v: a fork's pull request is not this branch's", f.forge.calls)
	}
}

// A local head the forge's head is not an ancestor of would have its push
// refused: the arming is left, and nothing is pushed.
func TestLandLeavesAnArmingAtAHeadItsLocalHeadDoesNotDescendFrom(t *testing.T) {
	t.Run("a head this checkout does not hold", func(t *testing.T) {
		f, _ := moving(t)
		f.reviewed(t)
		f.forge.armings = []string{armingBy(strings.Repeat("b", 40), true, false, "alice")}
		_, err := f.land(f.options())
		wantRefusal(t, err, "not an ancestor")
		wantRefusal(t, err, "nothing was pushed")
		noWithdrawal(t, f)
		if len(f.pushes()) != 0 {
			t.Fatalf("pushes %v", f.pushes())
		}
	})
	t.Run("a head beside the local one", func(t *testing.T) {
		f, old := moving(t)
		f.reviewed(t)
		tree := f.repo.Git("rev-parse", old+"^{tree}")
		beside := f.repo.Git("commit-tree", tree, "-p", old, "-m", "beside")
		f.forge.armings = []string{armingBy(beside, true, false, "alice")}
		_, err := f.land(f.options())
		wantRefusal(t, err, "not an ancestor")
		noWithdrawal(t, f)
		if len(f.pushes()) != 0 {
			t.Fatalf("pushes %v", f.pushes())
		}
	})
	t.Run("origin's branch is elsewhere", func(t *testing.T) {
		f, old := moving(t)
		f.reviewed(t)
		tree := f.repo.Git("rev-parse", old+"^{tree}")
		beside := f.repo.Git("commit-tree", tree, "-p", old, "-m", "beside")
		f.repo.Git("push", "-q", "origin", beside+":refs/heads/feat/x")
		f.forge.armings = []string{armingBy(old, true, false, "alice")}
		_, err := f.land(f.options())
		wantRefusal(t, err, "nothing was pushed")
		noWithdrawal(t, f)
		if len(f.pushes()) != 0 {
			t.Fatalf("pushes %v", f.pushes())
		}
	})
}

// The forge's reading can lag origin's: a push that changes nothing moves no
// head, so a good arming is not taken off for it.
func TestLandLeavesAnArmingWhenOriginAlreadyHoldsTheLocalHead(t *testing.T) {
	f, old := moving(t)
	f.reviewed(t)
	f.repo.Git("push", "-q", "origin", "feat/x")
	f.forge.prs = f.prFor(old)
	f.forge.armings = []string{armingBy(old, true, false, "alice")}
	res, err := f.land(f.options())
	if err != nil {
		t.Fatal(err)
	}
	noWithdrawal(t, f)
	if s := step(t, res, "withdraw"); s.Status != StatusLeft || !strings.Contains(s.Note, "left as it is") {
		t.Fatalf("withdraw step %+v", s)
	}
}

func TestLandLeavesAnArmingWhenOriginCannotSayWhereTheBranchIs(t *testing.T) {
	f, old := moving(t)
	f.reviewed(t)
	f.forge.armings = []string{armingBy(old, true, false, "alice")}
	o := f.options()
	inner := o.Git
	o.Git = func(dir string, args ...string) (string, error) {
		if len(args) > 0 && args[0] == "ls-remote" && args[len(args)-1] == "refs/heads/feat/x" {
			return "", errors.New("the remote hung up")
		}
		return inner(dir, args...)
	}
	_, err := f.land(o)
	wantRefusal(t, err, "the remote hung up")
	wantRefusal(t, err, "nothing was pushed")
	noWithdrawal(t, f)
	if len(f.pushes()) != 0 {
		t.Fatalf("pushes %v", f.pushes())
	}
}

// With the push off the head does not move, so nothing comes off.
func TestLandWithThePushOffLeavesAnArmingAtAnotherHead(t *testing.T) {
	f, old := moving(t)
	f.reviewed(t)
	f.forge.armings = []string{armingBy(old, true, false, "alice")}
	f.forge.prs = f.prFor(old)
	o := f.options()
	o.NoPush = true
	res, err := f.land(o)
	if err != nil {
		t.Fatal(err)
	}
	noWithdrawal(t, f)
	if s := step(t, res, "withdraw"); s.Status != StatusLeft || !strings.Contains(s.Note, "left as it is") {
		t.Fatalf("withdraw step %+v", s)
	}
}

func TestLandWithTheEnqueueOffStillWithdrawsAtTheHeadItsPushMoves(t *testing.T) {
	f, old := moving(t)
	f.reviewed(t)
	f.forge.armings = []string{armingBy(old, true, false, "alice"), "[]"}
	seen := withdrawnAtPush(f, "pr", "merge", "7", "--disable-auto")
	o := f.options()
	o.NoEnqueue = true
	if _, err := f.land(o); err != nil {
		t.Fatal(err)
	}
	if *seen != 1 || f.forge.called("pr", "merge", "7", "--auto") != 0 {
		t.Fatalf("withdrawals at the push %d, calls %v", *seen, f.forge.calls)
	}
}

// Another person's arming, or one the forge attributes to no one, comes off
// only where a receipt here names the head the push moves past. Otherwise it is
// left, and so that it does not stay in force over a new head, nothing is pushed.
func TestAnotherPersonsArmingComesOffOnlyPastAHeadAReceiptNames(t *testing.T) {
	for _, c := range []struct {
		name         string
		auto, queued bool
		by           string
	}{
		{"another account's arming", true, false, "someone-else"},
		{"another account's queue entry", false, true, "someone-else"},
		{"an arming the forge attributes to no one", true, false, ""},
	} {
		t.Run(c.name+", no receipt", func(t *testing.T) {
			f, old := moving(t)
			f.reviewed(t)
			f.forge.armings = []string{armingBy(old, c.auto, c.queued, c.by)}
			_, err := f.land(f.options())
			wantRefusal(t, err, "nothing was pushed")
			wantRefusal(t, err, short(old))
			noWithdrawal(t, f)
			if len(f.pushes()) != 0 {
				t.Fatalf("pushes %v", f.pushes())
			}
		})
		t.Run(c.name+", a receipt names its head", func(t *testing.T) {
			f, old := moving(t)
			f.reviewed(t)
			f.reviewedAt(t, old)
			f.forge.armings = []string{armingBy(old, c.auto, c.queued, c.by), "[]"}
			if _, err := f.land(f.options()); err != nil {
				t.Fatal(err)
			}
			if f.forge.called("pr", "merge", "7", "--disable-auto")+f.forge.called(dequeue7...) == 0 || f.onOrigin("feat/x") != f.head() {
				t.Fatalf("calls %v, origin %q", f.forge.calls, f.onOrigin("feat/x"))
			}
		})
	}
}

func TestAWithdrawalThatFailsStopsThePush(t *testing.T) {
	cases := []struct {
		name string
		set  func(f *fixture, old string)
		says string
	}{
		{"the forge refuses to turn auto-merge off", func(f *fixture, old string) {
			f.forge.armings = []string{armingBy(old, true, false, "alice")}
			f.forge.failDisarm = true
		}, "the disarm failed"},
		{"the forge refuses the dequeue", func(f *fixture, old string) {
			f.forge.armings = []string{armingBy(old, false, true, "alice")}
			f.forge.dequeueErr = errors.New("fake forge: the dequeue failed")
		}, "the dequeue failed"},
		{"the dequeue answers with errors", func(f *fixture, old string) {
			f.forge.armings = []string{armingBy(old, false, true, "alice")}
			f.forge.dequeueAnswer = `{"data":null,"errors":[{"message":"could not dequeue"}]}`
		}, "could not dequeue"},
		{"the forge still reports it armed", func(f *fixture, old string) {
			f.forge.armings = []string{armingBy(old, true, false, "alice")}
		}, "still reports"},
		{"the forge cannot be asked", func(f *fixture, _ string) {
			f.forge.armingsErr = errors.New("fake forge: the arming read failed")
		}, "the arming read failed"},
		{"its answer does not parse", func(f *fixture, _ string) {
			f.forge.armingsAnswer = "not json"
		}, "does not parse"},
		{"its answer carries errors", func(f *fixture, _ string) {
			f.forge.armingsAnswer = `{"data":null,"errors":[{"message":"no such field"}]}`
		}, "no such field"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f, old := moving(t)
			f.reviewed(t)
			c.set(f, old)
			_, err := f.land(f.options())
			wantRefusal(t, err, c.says)
			wantRefusal(t, err, "nothing was pushed")
			if len(f.pushes()) != 0 || f.onOrigin("feat/x") != old {
				t.Fatalf("pushes %v, origin %q: a failed withdrawal must stop the push", f.pushes(), f.onOrigin("feat/x"))
			}
		})
	}
}

// A listing with more pull requests behind it could hide an armed one among
// them, so it is no answer.
func TestAListingWithMoreBehindItStopsThePush(t *testing.T) {
	f, old := moving(t)
	f.reviewed(t)
	f.forge.armingsAnswer = `{"data":{"viewer":{"login":"alice"},"repository":{"pullRequests":{"pageInfo":{"hasNextPage":true},"nodes":[]}}}}`
	_, err := f.land(f.options())
	wantRefusal(t, err, "more than 100")
	wantRefusal(t, err, "nothing was pushed")
	if len(f.pushes()) != 0 || f.onOrigin("feat/x") != old {
		t.Fatalf("pushes %v, origin %q", f.pushes(), f.onOrigin("feat/x"))
	}
	noWithdrawal(t, f)
}

// DecideMerge tells a refusal of the head from an answer it could not read.
func TestTheMergeGateMarksAnAnswerItCouldNotRead(t *testing.T) {
	for _, c := range []struct {
		name, config string
		set          func(f *fixture)
		unread       bool
	}{
		{"auto, no receipt", autoConfig, nil, false},
		{"human, no approval", humanConfig, func(f *fixture) { f.forge.reviews = "[]" }, false},
		{"contributor, no approval", autoConfig, func(f *fixture) {
			f.forge.roles = map[string]string{"author-a": `{"permission":"read","role_name":"read"}`}
			f.forge.reviews = "[]"
		}, false},
		{"the author's permission fails", autoConfig, func(f *fixture) {
			f.forge.roles = map[string]string{"author-a": "!gh: HTTP 502: Bad Gateway"}
		}, true},
		{"the pull request read fails", autoConfig, func(f *fixture) { f.forge.pullErr = errors.New("gh: HTTP 502") }, true},
		{"the reviews read fails", humanConfig, func(f *fixture) { f.forge.reviewsErr = errors.New("gh: HTTP 502") }, true},
		{"the reviews do not parse", humanConfig, func(f *fixture) { f.forge.reviews = "not json" }, true},
		{"a reviewer's permission fails", humanConfig, func(f *fixture) {
			f.forge.roles = map[string]string{"author-a": `{"permission":"write","role_name":"write"}`, "maintainer-b": "!gh: HTTP 502"}
		}, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": c.config})
			f.forge.head = f.head()
			if c.set != nil {
				c.set(f)
			}
			tip, err := gitutil.Run(f.repo.Root(), "rev-parse", "origin/main")
			if err != nil {
				t.Fatal(err)
			}
			d, err := DecideMerge(MergeQuery{RepoRoot: f.repo.Root(), Roots: f.options().Roots, Tip: tip, TipName: "origin/main", Default: "main",
				Repo: "o/r", PR: "7", Head: f.head(), Forge: func(args ...string) (string, error) { return f.forge.run("", args...) }})
			if err != nil {
				t.Fatal(err)
			}
			if d.Arm || d.Unread != c.unread {
				t.Fatalf("decision %+v: want no arming, Unread %v", d, c.unread)
			}
		})
	}
}

// The gate every re-arm is held to is the merge gate at the default branch as
// last fetched, asked of the head recorded.
func TestTheGateARearmAsksIsTheMergeGate(t *testing.T) {
	f := newFixtureFiles(t, queueRuleset, map[string]string{".abcd/config.json": autoConfig})
	f.forge.head = f.head()
	ask := func(args ...string) (string, error) { return f.forge.run(f.repo.Root(), args...) }
	arm, why, err := rearmGate(ask, f.repo.Root(), "o/r", "7", f.head())
	if err != nil || arm || !strings.Contains(why, "no review receipt names "+f.head()) {
		t.Fatalf("arm %v, why %q, err %v: want no arming at a head no receipt names", arm, why, err)
	}
	f.reviewed(t)
	if arm, why, err := rearmGate(ask, f.repo.Root(), "o/r", "7", f.head()); err != nil || !arm {
		t.Fatalf("arm %v, why %q, err %v: want the arming at a reviewed head", arm, why, err)
	}
	if !claim.ArmGateWired() {
		t.Fatal("linking this package leaves no merge gate wired for a re-arm")
	}
}
