package claim

import (
	"testing"
)

// A release that has ended with nothing of the reader's left to change is
// marked done for them, so no later act reads it again
// (iss-2610290809526033). A `kept` entry, a withdrawal the forge refused,
// left the pull request armed: once the release has ended there is nothing
// to re-arm, so an ended release whose only entry of the reader's is `kept`
// is done at the first act that reads it.

// pushAsksOf is how many times the forge's record of pushes has been asked.
func pushAsksOf(f *fakeForge) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.pushAsks
}

// TestAnEndedReleaseHoldingOnlyTheReadersKeptEntryIsDoneAtTheFirstAct: the
// forge refuses octo's withdrawal of #5, so his cut records it kept; he
// releases the cut, and its tag lands. His next act reads the release and
// changes nothing; the act after it does not ask the forge's record of
// pushes again.
func TestAnEndedReleaseHoldingOnlyTheReadersKeptEntryIsDoneAtTheFirstAct(t *testing.T) {
	w := newWorld(t)
	f := newFakeForge(t, map[int]*fakePR{5: {id: "PR_five", head: headA, method: "SQUASH"}})
	releaseForge.comments = f
	f.refuse["disablePullRequestAutoMerge #5"] = "GraphQL: octo does not have permission to disable auto-merge"
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := octo.do(Request{Record: "rel-v0.16.0"}); err != nil {
		t.Fatal(err)
	}
	if _, ws := w.tipWithdrawals("rel-v0.16.0"); stateOf(ws, 5) != WithdrawnKept {
		t.Fatalf("the cut records %+v; want #5 kept", ws)
	}
	if _, err := octo.do(Request{Record: "rel-v0.16.0", Release: true}); err != nil {
		t.Fatal(err)
	}
	if tip, ws := w.tipWithdrawals("rel-v0.16.0"); !tip.released() || stateOf(ws, 5) != WithdrawnKept {
		t.Fatalf("the release's tombstone records %+v; want #5 kept", ws)
	}
	w.tagOrigin("v0.16.0")
	f.reset()
	tip := w.remoteTip("rel-v0.16.0")
	if _, err := octo.reserve("iss-8", "o.go"); err != nil {
		t.Fatal(err)
	}
	asked := pushAsksOf(f)
	if _, err := octo.reserve("iss-8", "o.go"); err != nil {
		t.Fatal(err)
	}
	if got := pushAsksOf(f) - asked; got != 0 {
		t.Fatalf("the second act asked the record of pushes %d times; want the ended release done after the first", got)
	}
	if got := f.mutations(); len(got) != 0 || w.remoteTip("rel-v0.16.0") != tip {
		t.Fatalf("mutations = %v, tip moved %v; want nothing acted on or pushed", got, w.remoteTip("rel-v0.16.0") != tip)
	}
}
