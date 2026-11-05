package loop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The watch over an armed lane queues or toggles only a head the merge gate
// still arms. It takes the arming off only when it is sure: the gate refused the
// head, or origin's branch has moved past the head the loop pushed. An answer it
// could not read leaves the arming as it is, and the wait says so.

// armedThenApprovalGone is an armed lane CLEAN outside the queue whose head no
// approval counts for any more, and the forge's log before the next step.
func armedThenApprovalGone(t *testing.T) (*landFixture, string) {
	t.Helper()
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.landedToArmed(t)
	f.queueAnswer(t, stubQueue{head: l.Landing.Pushed, enabledAt: "2026-10-15T07:00:00Z", status: "CLEAN"})
	if err := os.Remove(filepath.Join(f.gh, "approved")); err != nil {
		t.Fatal(err)
	}
	return f, f.ghLog(t)
}

func TestTheWatchWithdrawsAnArmingTheGateRefuses(t *testing.T) {
	f, before := armedThenApprovalGone(t)
	f.step(t)
	log := strings.TrimPrefix(f.ghLog(t), before)
	if strings.Contains(log, "enqueuePullRequest") || strings.Contains(log, "pr merge 7 --auto") {
		t.Fatalf("a head the gate does not arm was queued or armed:\n%s", log)
	}
	if !strings.Contains(log, "pr merge 7 --disable-auto") {
		t.Fatalf("the arming stays in force at a head the gate refuses:\n%s", log)
	}
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	if ld := st.Lanes[0].Landing; ld.Armed || !strings.HasPrefix(ld.Merge, "left open") {
		t.Fatalf("the lane is left open, for the gate to be asked again: %+v", ld)
	}
	if note := f.lastNote(t); !strings.Contains(note, "withdrew") {
		t.Fatalf("the record says the arming was withdrawn: %q", note)
	}
}

func TestTheWatchStopsWhenItCannotWithdrawAnArmingTheGateRefuses(t *testing.T) {
	f, before := armedThenApprovalGone(t)
	if err := os.WriteFile(filepath.Join(f.gh, "refuse-disarm"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
	if r := mustRefusal(t, err); r.Contention {
		t.Fatalf("a withdrawal the forge refused is not a wait: %+v", r)
	}
	if log := strings.TrimPrefix(f.ghLog(t), before); strings.Contains(log, "enqueuePullRequest") {
		t.Fatalf("a head the gate refuses was queued:\n%s", log)
	}
}

// A forge read that fails is no refusal of the head: the arming stays, nothing
// is queued or toggled, and the wait says why.
func TestTheWatchLeavesAnArmingWhenItCannotReadTheGate(t *testing.T) {
	for _, c := range []struct{ name, file, body string }{
		{"the author's permission", "role-author-a", "Bad Gateway"},
		{"the reviews", "reviews.json", "Bad Gateway"},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := newLandFixture(t, queueRuleset("MERGE"))
			l := f.landedToArmed(t)
			f.queueAnswer(t, stubQueue{head: l.Landing.Pushed, enabledAt: "2026-10-15T07:00:00Z", status: "CLEAN"})
			if err := os.WriteFile(filepath.Join(f.gh, c.file), []byte(c.body), 0o644); err != nil {
				t.Fatal(err)
			}
			before := f.ghLog(t)
			_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
			r := mustRefusal(t, err)
			if !r.Contention || !strings.Contains(r.Reason, "left as it is") {
				t.Fatalf("an unread gate is a wait that says the arming stays: %+v", r)
			}
			log := strings.TrimPrefix(f.ghLog(t), before)
			if strings.Contains(log, "--disable-auto") || strings.Contains(log, "enqueuePullRequest") || strings.Contains(log, "pr merge 7 --auto") {
				t.Fatalf("an unread gate touched the arming:\n%s", log)
			}
		})
	}
}

// movedOnOrigin puts a commit past the lane's pushed head on origin's branch,
// and returns it.
func movedOnOrigin(t *testing.T, f *landFixture, l Lane) string {
	t.Helper()
	late := laneCommit(t, f.repo, l, "late.txt")
	f.repo.Git("push", "-q", "origin", late+":refs/heads/"+l.Branch)
	return late
}

func TestTheWatchWithdrawsAnArmingWhoseHeadMoved(t *testing.T) {
	for _, c := range []struct {
		name string
		q    stubQueue
		want []string
	}{
		{"armed", stubQueue{enabledAt: "2026-10-15T07:00:00Z", status: "CLEAN"}, []string{"pr merge 7 --disable-auto"}},
		{"queued", stubQueue{queued: true, status: "CLEAN"}, []string{"dequeuePullRequest", "id=PR_kwDOstub7"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := newLandFixture(t, queueRuleset("MERGE"))
			l := f.landedToArmed(t)
			c.q.head = movedOnOrigin(t, f, l)
			f.queueAnswer(t, c.q)
			before := f.ghLog(t)
			_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
			r := mustRefusal(t, err)
			if r.Contention || !r.outOfQueue || !strings.Contains(r.Reason, "withdrew") {
				t.Fatalf("a moved head is reported, its arming withdrawn: %+v", r)
			}
			log := strings.TrimPrefix(f.ghLog(t), before)
			for _, want := range c.want {
				if !strings.Contains(log, want) {
					t.Fatalf("the forge was not asked %q:\n%s", want, log)
				}
			}
		})
	}
}

// The forge's reading can lag: origin still at the pushed head is no move.
func TestTheWatchLeavesAnArmingWhileOriginHoldsThePushedHead(t *testing.T) {
	for _, q := range []stubQueue{
		{head: strings.Repeat("c", 40), enabledAt: "2026-10-15T07:00:00Z", status: "BLOCKED"},
		{head: strings.Repeat("c", 40), queued: true, status: "CLEAN"},
	} {
		f := newLandFixture(t, queueRuleset("MERGE"))
		f.landedToArmed(t)
		f.queueAnswer(t, q)
		before := f.ghLog(t)
		_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
		if r := mustRefusal(t, err); !r.Contention || !strings.Contains(r.Reason, "left as it is") {
			t.Fatalf("a lagging reading is a wait: %+v", r)
		}
		log := strings.TrimPrefix(f.ghLog(t), before)
		if strings.Contains(log, "--disable-auto") || strings.Contains(log, "dequeuePullRequest") {
			t.Fatalf("a lagging reading took the arming off:\n%s", log)
		}
	}
}

func TestTheWatchSaysWhenItCouldNotWithdrawAnArmingWhoseHeadMoved(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.landedToArmed(t)
	f.queueAnswer(t, stubQueue{head: movedOnOrigin(t, f, l), enabledAt: "2026-10-15T07:00:00Z", status: "CLEAN"})
	if err := os.WriteFile(filepath.Join(f.gh, "refuse-disarm"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
	r := mustRefusal(t, err)
	if !strings.Contains(r.Reason, "could not withdraw") || !strings.Contains(r.Remedy, "gh pr merge 7 --disable-auto") {
		t.Fatalf("the refusal says the arming stands and how to take it off: %+v", r)
	}
}

func TestTheWaitWithdrawsAnAutoMergeWhoseHeadMovedWithNoQueue(t *testing.T) {
	f := newLandFixture(t, noQueueRuleset)
	f.approve(t)
	l := f.landedToArmed(t)
	f.queueAnswer(t, stubQueue{head: movedOnOrigin(t, f, l), enabledAt: "2026-10-15T07:00:00Z", status: "CLEAN"})
	before := f.ghLog(t)
	_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
	if r := mustRefusal(t, err); r.Contention || !strings.Contains(r.Reason, "withdrew") {
		t.Fatalf("a moved head is reported, its arming withdrawn: %+v", r)
	}
	if log := strings.TrimPrefix(f.ghLog(t), before); !strings.Contains(log, "pr merge 7 --disable-auto") {
		t.Fatalf("the arming stays in force over the moved head:\n%s", log)
	}
}

// stillArmed makes the stub forge go on reporting the pull request armed.
func stillArmed(t *testing.T, f *landFixture) {
	t.Helper()
	body := `{"data":{"viewer":{"login":"author-a"},"repository":{"pullRequests":{"pageInfo":{"hasNextPage":false},"nodes":[` +
		`{"number":7,"id":"PR_kwDOstub7","headRefOid":"` + strings.Repeat("c", 40) + `","isCrossRepository":false,"isInMergeQueue":false,"mergeQueueEntry":null,"autoMergeRequest":{"mergeMethod":"MERGE","enabledBy":{"login":"author-a"}}}]}}}}`
	if err := os.WriteFile(filepath.Join(f.gh, "armings.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// gh can exit 0 without withdrawing, so the watch reads the arming back as land
// does, and a withdrawal it cannot confirm is not reported as made.
func TestTheWatchReadsBackAnArmingItWithdrew(t *testing.T) {
	gateRefused := func(t *testing.T) *landFixture {
		f, _ := armedThenApprovalGone(t)
		return f
	}
	moved := func(t *testing.T) *landFixture {
		f := newLandFixture(t, queueRuleset("MERGE"))
		l := f.landedToArmed(t)
		f.queueAnswer(t, stubQueue{head: movedOnOrigin(t, f, l), enabledAt: "2026-10-15T07:00:00Z", status: "CLEAN"})
		return f
	}
	for _, c := range []struct {
		name string
		make func(t *testing.T) *landFixture
	}{{"a head the gate refuses", gateRefused}, {"a moved head", moved}} {
		t.Run(c.name+", still armed", func(t *testing.T) {
			f := c.make(t)
			stillArmed(t, f)
			_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
			r := mustRefusal(t, err)
			if r.Contention || !strings.Contains(r.Reason, "still reports") || r.outOfQueue {
				t.Fatalf("an arming the forge still reports is refused, and not as out of the queue: %+v", r)
			}
			st, err := ReadState(f.repo.Root(), f.runID)
			if err != nil {
				t.Fatal(err)
			}
			if !st.Lanes[0].Landing.Armed {
				t.Fatalf("the lane no longer records the arming the forge still reports: %+v", st.Lanes[0].Landing)
			}
		})
	}
	t.Run("a moved head, the read fails", func(t *testing.T) {
		f := moved(t)
		if err := os.WriteFile(filepath.Join(f.gh, "refuse-armings"), nil, 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
		r := mustRefusal(t, err)
		if r.Contention || !strings.Contains(r.Reason, "could not read") {
			t.Fatalf("a read-back that fails is refused, naming it: %+v", r)
		}
	})
}

// A withdrawal the forge took, at a head the gate refuses, stays recorded when
// its read-back fails: the lane is left open, and the next step asks the gate
// again instead of reading the pull request as dropped from the queue.
func TestAWithdrawalAtARefusedHeadIsRecordedWhenItsReadBackFails(t *testing.T) {
	f, _ := armedThenApprovalGone(t)
	if err := os.WriteFile(filepath.Join(f.gh, "refuse-armings"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := advance(f.repo.Root(), f.runID, f.stages, Options{}); err != nil {
		t.Fatalf("the withdrawal is recorded, not refused: %v", err)
	}
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	if ld := st.Lanes[0].Landing; ld.Armed || !strings.HasPrefix(ld.Merge, "left open") {
		t.Fatalf("the lane is left open: %+v", ld)
	}
	if note := f.lastNote(t); !strings.Contains(note, "withdrew") || !strings.Contains(note, "could not read") {
		t.Fatalf("the record says what was and was not read: %q", note)
	}
	if err := os.Remove(filepath.Join(f.gh, "refuse-armings")); err != nil {
		t.Fatal(err)
	}
	f.queueAnswer(t, stubQueue{head: st.Lanes[0].Landing.Pushed, status: "CLEAN"})
	before := f.ghLog(t)
	_, err = advance(f.repo.Root(), f.runID, f.stages, Options{})
	r := mustRefusal(t, err)
	if !r.Contention || r.outOfQueue || strings.Contains(r.Reason, "left main's merge queue") || strings.Contains(r.Remedy, "--match-head-commit") {
		t.Fatalf("the next step waits on the gate, with no drop and no re-arming advised: %+v", r)
	}
	if log := strings.TrimPrefix(f.ghLog(t), before); strings.Contains(log, "pr merge 7 --auto") {
		t.Fatalf("a head the gate refuses was armed:\n%s", log)
	}
}

// An enqueue the forge goes on refusing is toggled a bounded number of times,
// and then the wait says the loop has stopped.
func TestTheWatchStopsTogglingAfterItsBound(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.landedToArmed(t)
	f.queueAnswer(t, stubQueue{head: l.Landing.Pushed, enabledAt: "2026-10-15T07:00:00Z", status: "CLEAN"})
	if err := os.WriteFile(filepath.Join(f.gh, "refuse-enqueue"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	before := f.ghLog(t)
	late := Options{Now: func() time.Time { return time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC) }}
	var last error
	for range maxToggles + 3 {
		_, last = advance(f.repo.Root(), f.runID, f.stages, late)
	}
	log := strings.TrimPrefix(f.ghLog(t), before)
	if got := strings.Count(log, "pr merge 7 --disable-auto"); got != maxToggles {
		t.Fatalf("toggled %d times, want %d:\n%s", got, maxToggles, log)
	}
	if r := mustRefusal(t, last); !r.Contention || !strings.Contains(r.Reason, "no longer toggles") {
		t.Fatalf("the wait says the loop stopped: %+v", r)
	}
}

// A facilitator.mode that does not read is a wait for an armed lane: the arming
// stands, and the wait says so rather than that nothing was armed.
func TestTheWatchWaitsWhenTheModeDoesNotRead(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.landedToArmed(t)
	f.queueAnswer(t, stubQueue{head: l.Landing.Pushed, enabledAt: "2026-10-15T07:00:00Z", status: "CLEAN"})
	dir := t.TempDir()
	f.repo.Git("clone", "-q", f.bare, dir)
	if err := os.MkdirAll(filepath.Join(dir, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".abcd", "config.json"), []byte(`{"facilitator":{"mode":"sometimes"}}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f.repo.Git("-C", dir, "add", ".abcd/config.json")
	f.repo.Git("-C", dir, "-c", "user.name=A Person", "-c", "user.email=a@example.com", "-c", "core.hooksPath=/dev/null", "commit", "-q", "-m", "a mode that does not read")
	f.repo.Git("-C", dir, "-c", "core.hooksPath=/dev/null", "push", "-q", "origin", "main")
	before := f.ghLog(t)
	_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
	r := mustRefusal(t, err)
	if !r.Contention || !strings.Contains(r.Reason, "left as it is") || strings.Contains(r.Reason, "nothing was armed") {
		t.Fatalf("a mode that does not read is a wait that says the arming stays: %+v", r)
	}
	if log := strings.TrimPrefix(f.ghLog(t), before); strings.Contains(log, "--disable-auto") || strings.Contains(log, "enqueuePullRequest") {
		t.Fatalf("the arming was touched:\n%s", log)
	}
}
