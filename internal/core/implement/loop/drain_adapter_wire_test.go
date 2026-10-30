package loop

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/openaiapi"
	"github.com/intentdriven/abcd/internal/core/lab"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/core/runner"
)

// The drain's judgement through the decision adapter, as a drain move runs it
// (itd-2609221009495079 criteria 1, 3 and 5): the type eligibility-residual on
// the host leaves the drain as it was; in shadow the host's answer is taken
// and the adapter's is recorded beside it, or skipped with the state's size
// when the state is over the window; turned on, the adapter's answer is taken
// in the move that asks it, the run record naming the route, and an adapter
// that cannot answer leaves the host's request standing with a fallback
// receipt in the drain's state, asked once per request.

// drainStateOf reads the drain's state as a move left it.
func drainStateOf(t *testing.T, root string) DrainState {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(DrainStateRel)))
	if err != nil {
		t.Fatal(err)
	}
	var st DrainState
	if err := json.Unmarshal(raw, &st); err != nil {
		t.Fatal(err)
	}
	return st
}

// TestADrainOnTheHostAsksNoAdapter: with eligibility-residual on the host,
// the move writes the host's request as it always did, the host's answer
// opens the lane, and the adapter is never asked and no pair is kept.
func TestADrainOnTheHostAsksNoAdapter(t *testing.T) {
	repo := issueRepo(t)
	adapter, got := &drainFakeJudge{value: HandBackTrustRule, p: 0.9}, []lab.Pair{}
	withDrainRouter(t, oracle.JudgementHost, adapter, &got)
	res, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Judging == nil || res.Judged != nil || !strings.Contains(res.Next, "abcd drain --judgement") || strings.Contains(res.Next, drainAdapterRoute) {
		t.Fatalf("the host's request is written and awaited: %+v", res)
	}
	res = judge(t, repo, Options{}, res, JudgementNo, "", "a renderer guard")
	if res.Judged == nil || res.Judged.Route != nil || res.Start == nil {
		t.Fatalf("the host's no is taken and opens the lane: %+v", res)
	}
	if adapter.calls != 0 || len(got) != 0 || len(drainStateOf(t, repo.Root()).JudgeFallbacks) != 0 {
		t.Fatalf("on the host the adapter is asked %d times, %d pairs are kept", adapter.calls, len(got))
	}
}

// TestADrainInShadowRecordsThePairAndKeepsTheHostsAnswer is criterion 1 in
// the drain: the host's answer decides, and once it is taken the adapter is
// asked the same question and the pair is recorded; the adapter's different
// answer changes nothing.
func TestADrainInShadowRecordsThePairAndKeepsTheHostsAnswer(t *testing.T) {
	repo := issueRepo(t)
	adapter, got := &drainFakeJudge{value: HandBackTrustRule, p: 0.8}, []lab.Pair{}
	withDrainRouter(t, oracle.JudgementShadow, adapter, &got)
	res, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Judging == nil || adapter.calls != 0 {
		t.Fatalf("shadow asks the adapter nothing before the host answers: %+v, calls %d", res, adapter.calls)
	}
	res = judge(t, repo, Options{}, res, JudgementNo, "", "a renderer guard")
	if res.Judged == nil || res.Judged.Answer != JudgementNo || res.Judged.Route != nil || res.Start == nil {
		t.Fatalf("the host's no is taken and opens the lane, whatever the adapter said: %+v", res)
	}
	if adapter.calls != 1 || len(got) != 1 {
		t.Fatalf("the adapter is asked once beside the host and one pair kept: calls %d, pairs %+v", adapter.calls, got)
	}
	p := got[0]
	if p.Type != "eligibility-residual" || p.Host != JudgementNo || p.Adapter != HandBackTrustRule || p.Probability == nil ||
		*p.Probability != 0.8 || len(p.StateSHA256) != 64 || p.Skipped != "" {
		t.Fatalf("pair = %+v", p)
	}
	if st := drainStateOf(t, repo.Root()); len(st.JudgeFallbacks) != 0 {
		t.Fatalf("shadow keeps no fallback in the drain's state: %+v", st.JudgeFallbacks)
	}
}

// TestADrainInShadowSkipsAStateOverTheWindow is criterion 5 in the drain: a
// state larger than the adapter's window is not sent; the pair is recorded as
// skipped with the state's size, and the host's answer decides alone.
func TestADrainInShadowSkipsAStateOverTheWindow(t *testing.T) {
	repo := issueRepo(t)
	adapter, got := &drainFakeJudge{value: JudgementNo, p: 0.9}, []lab.Pair{}
	withDrainRouterWindow(t, oracle.JudgementShadow, 8, adapter, &got)
	res, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	res = judge(t, repo, Options{}, res, JudgementNo, "", "a renderer guard")
	if res.Judged == nil || res.Start == nil {
		t.Fatalf("the host's no opens the lane: %+v", res)
	}
	if adapter.calls != 0 || len(got) != 1 {
		t.Fatalf("a state over the window is not sent: calls %d, pairs %+v", adapter.calls, got)
	}
	if p := got[0]; p.Skipped != lab.SkippedWindow || p.StateTokens <= 8 || p.Host != JudgementNo ||
		!strings.Contains(p.Detail, "tokens") || p.Adapter != "" {
		t.Fatalf("pair = %+v, want a window skip with the size", p)
	}
}

// TestADrainTurnedOnTakesTheAdaptersAnswerAndNamesTheRoute is criterion 3's
// first half in the drain: the adapter's no is taken in the move that asks
// it, the lane opens, and the drain's record names the adapter's route.
func TestADrainTurnedOnTakesTheAdaptersAnswerAndNamesTheRoute(t *testing.T) {
	repo := issueRepo(t)
	adapter, got := &drainFakeJudge{value: JudgementNo, p: 0.97}, []lab.Pair{}
	withDrainRouter(t, oracle.JudgementOn, adapter, &got)
	res, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Judging != nil || res.Judged == nil || res.Judged.Answer != JudgementNo || !res.Judged.Applied || res.Start == nil {
		t.Fatalf("the adapter's no is taken and opens the lane in the same move: %+v", res)
	}
	want := drainAdapterRoute + " openrouter/typesafe/jev-1.13"
	st := drainStateOf(t, repo.Root())
	if len(st.Judgements) != 1 {
		t.Fatalf("judgements = %+v", st.Judgements)
	}
	if r := st.Judgements[0].Route; r == nil || r.Asked != want || r.Ran != want || r.Model != "typesafe/jev-1.13-0901" {
		t.Fatalf("the drain's record names the route: %+v", r)
	}
	if adapter.calls != 1 || len(got) != 0 || len(st.JudgeFallbacks) != 0 {
		t.Fatalf("calls %d, pairs %d, fallbacks %+v", adapter.calls, len(got), st.JudgeFallbacks)
	}
}

// TestADrainTurnedOnFallsBackToTheHostWithAReceiptOnce is criterion 3's
// second half in the drain: an adapter that cannot answer leaves the host's
// request standing, the fallback receipt kept in the drain's state; the polls
// that follow while the request stands ask it nothing again, and the host's
// answer is taken as it always was.
func TestADrainTurnedOnFallsBackToTheHostWithAReceiptOnce(t *testing.T) {
	repo := issueRepo(t)
	adapter, got := &drainFakeJudge{err: openaiapi.ErrUnreachable}, []lab.Pair{}
	withDrainRouter(t, oracle.JudgementOn, adapter, &got)
	res, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Judging == nil || res.Judged != nil || res.Start != nil {
		t.Fatalf("the host's request stands: %+v", res)
	}
	if !strings.Contains(res.Next, "abcd drain --judgement") || !strings.Contains(res.Next, "the decision adapter did not judge "+eligibleIssue) {
		t.Fatalf("next names the fallback and the host's answer: %s", res.Next)
	}
	fbs := drainStateOf(t, repo.Root()).JudgeFallbacks
	if len(fbs) != 1 || fbs[0].Role != RoleDrainJudge || fbs[0].Ran != runner.Host ||
		fbs[0].Asked != drainAdapterRoute+" openrouter/typesafe/jev-1.13" || !strings.Contains(fbs[0].Detail, "the host's request stands") {
		t.Fatalf("the fallback receipt is kept in the drain's state: %+v", fbs)
	}
	for range 2 {
		poll, err := Drain(repo.Root(), Options{}, DrainOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if poll.Judging == nil || poll.Judged != nil || !strings.Contains(poll.Next, "the decision adapter was already asked over "+eligibleIssue) {
			t.Fatalf("the host's request still stands, and the adapter is not asked again: %+v", poll)
		}
		res = poll
	}
	if adapter.calls != 1 || len(drainStateOf(t, repo.Root()).JudgeFallbacks) != 1 {
		t.Fatalf("the adapter is asked once per request: %d calls, fallbacks %+v", adapter.calls, drainStateOf(t, repo.Root()).JudgeFallbacks)
	}
	res = judge(t, repo, Options{}, res, JudgementNo, "", "a renderer guard")
	if res.Judged == nil || res.Judged.Route != nil || res.Start == nil {
		t.Fatalf("the host's answer is taken and opens the lane: %+v", res)
	}
}

// TestADrainStoppedWhileTheAdapterJudgesTakesNoAnswer: a drain stopped while
// the decision adapter is asked refuses the move, records no fallback for the
// stop, and leaves the request unasked, so the next move asks again.
func TestADrainStoppedWhileTheAdapterJudgesTakesNoAnswer(t *testing.T) {
	repo := issueRepo(t)
	adapter, got := &drainFakeJudge{err: context.Canceled}, []lab.Pair{}
	withDrainRouter(t, oracle.JudgementOn, adapter, &got)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Drain(repo.Root(), Options{}, DrainOptions{Context: ctx}); err == nil || !strings.Contains(err.Error(), "stopped while the decision adapter judged") {
		t.Fatalf("a stopped drain refuses the move: %v", err)
	}
	// A first move refused writes no state at all; any it wrote keeps no
	// fallback and leaves the request unasked.
	if _, err := os.Stat(filepath.Join(repo.Root(), filepath.FromSlash(DrainStateRel))); err == nil {
		st := drainStateOf(t, repo.Root())
		if len(st.JudgeFallbacks) != 0 || (st.Judging != nil && st.Judging.RoutedAsked) {
			t.Fatalf("the stop is no fallback and the request is asked again: %+v", st)
		}
	}
	adapter.err, adapter.value, adapter.p = nil, JudgementNo, 0.9
	res, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil || res.Judged == nil || res.Judged.Route == nil || adapter.calls != 2 {
		t.Fatalf("the next move asks the adapter again and takes its answer: %+v, %v, calls %d", res, err, adapter.calls)
	}
}
