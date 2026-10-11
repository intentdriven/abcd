package loop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The host judgement before a lane opens (itd-82 scope 3, criterion 2;
// spc-2609212015054359 step 1): an eligible issue whose remedy changes what a
// user sees or a trust boundary is handed back before any lane opens, one
// whose remedy changes neither opens its lane, and the judgement never makes
// an ineligible issue eligible.

// judgementAnswer is the host's answer, as a map so a test can add a field
// the answer's shape does not name.
func judgementAnswer(j *DrainJudging, answer, kind, reason string) map[string]any {
	a := map[string]any{"schema_version": DrainJudgementSchemaVersion, "issue": j.Issue,
		"remedy_sha256": j.RemedySHA256, "answer": answer, "reason": reason}
	if kind != "" {
		a["kind"] = kind
	}
	return a
}

// writeAnswer writes the host's answer where the drain asked for it and
// returns the path.
func writeAnswer(t *testing.T, repo *gittest.Repo, j *DrainJudging, a map[string]any) string {
	t.Helper()
	data, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(repo.Root(), filepath.FromSlash(j.Answer))
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// judge answers the judgement res awaits and makes the drain's next move.
func judge(t *testing.T, repo *gittest.Repo, o Options, res DrainResult, answer, kind, reason string) DrainResult {
	t.Helper()
	if res.Judging == nil {
		t.Fatalf("the drain awaits no judgement: %+v", res)
	}
	path := writeAnswer(t, repo, res.Judging, judgementAnswer(res.Judging, answer, kind, reason))
	out, err := Drain(repo.Root(), o, DrainOptions{Judgement: path})
	if err != nil {
		t.Fatalf("the host's %s answer on %s: %v", answer, res.Judging.Issue, err)
	}
	return out
}

// drainToLane makes one drain move and, when it awaits the host judgement,
// answers that the remedy changes neither, so the move opens the lane.
func drainToLane(t *testing.T, repo *gittest.Repo, o Options, d DrainOptions) DrainResult {
	t.Helper()
	res, err := Drain(repo.Root(), o, d)
	if err != nil {
		t.Fatal(err)
	}
	if res.Judging == nil {
		return res
	}
	return judge(t, repo, o, res, JudgementNo, "", "the remedy changes neither what a user sees nor a trust boundary")
}

func runCount(t *testing.T, repo *gittest.Repo) int {
	t.Helper()
	runs, err := Runs(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	return len(runs)
}

// TestAnEligibleIssueIsJudgedBeforeItsLaneOpens: the drain's first move over
// an eligible issue opens no lane; it writes a request that asks the one
// question over the issue's remedy and names where the answer goes.
func TestAnEligibleIssueIsJudgedBeforeItsLaneOpens(t *testing.T) {
	repo := issueRepo(t)
	res, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Start != nil || res.Lane != nil || runCount(t, repo) != 0 {
		t.Fatalf("no lane opens before the judgement: %+v", res)
	}
	j := res.Judging
	if j == nil || j.Issue != eligibleIssue || j.Request != DrainJudgementRequestRel || j.Answer != DrainJudgementAnswerRel || len(j.RemedySHA256) != 64 {
		t.Fatalf("the drain awaits the judgement on the eligible issue: %+v", j)
	}
	req, err := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(j.Request)))
	if err != nil {
		t.Fatalf("the request is written: %v", err)
	}
	for _, want := range []string{eligibleIssue, issueRemedy, "what a user sees", "trust boundary", j.RemedySHA256, j.Answer, "abcd drain --judgement"} {
		if !strings.Contains(string(req), want) {
			t.Errorf("the request does not carry %q:\n%s", want, req)
		}
	}
	if !strings.Contains(res.Next, j.Request) || !strings.Contains(res.Next, "--judgement") {
		t.Fatalf("next names the request and the answer's verb: %s", res.Next)
	}
	again, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil || again.Judging == nil || again.Judging.Issue != eligibleIssue || again.Start != nil {
		t.Fatalf("until it is answered the drain still awaits the judgement: %+v %v", again, err)
	}
}

// TestARemedyThatChangesNeitherOpensItsLane: a no changes nothing, so the
// issue's lane opens as its fields allow, and the judgement is recorded.
func TestARemedyThatChangesNeitherOpensItsLane(t *testing.T) {
	repo := issueRepo(t)
	first, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	res := judge(t, repo, Options{}, first, JudgementNo, "", "an internal guard and its test")
	if res.Start == nil || res.Lane == nil || res.Lane.Issue != eligibleIssue || res.Judging != nil {
		t.Fatalf("a no opens the issue's lane: %+v", res)
	}
	if res.Judged == nil || res.Judged.Answer != JudgementNo || !res.Judged.Applied || len(res.Judgements) != 1 {
		t.Fatalf("the judgement is recorded with the disposition: %+v", res)
	}
	st, _, err := readDrain(repo.Root())
	if err != nil || len(st.Judgements) != 1 || st.Judgements[0].Reason != "an internal guard and its test" || st.Judging != nil {
		t.Fatalf("the drain's state keeps the judgement: %+v %v", st, err)
	}
}

// TestAUserVisibleRemedyIsHandedBackBeforeAnyLaneOpens: a yes of kind
// user-visible routes the issue as a lane's user-visible hand-back is routed,
// promoted to an intent draft, and no lane opens for it.
func TestAUserVisibleRemedyIsHandedBackBeforeAnyLaneOpens(t *testing.T) {
	repo := drainRepo(t)
	first, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if first.Judging == nil || first.Judging.Issue != secondIssue {
		t.Fatalf("the first issue in the drain order is judged first: %+v", first.Judging)
	}
	res := judge(t, repo, Options{}, first, JudgementYes, HandBackUserVisible, "the page gains a flag users read")
	r := routeOf(t, res.Routed, secondIssue)
	if r.From != DrainFromJudgement || r.Route != RoutePromoted || !strings.HasPrefix(r.Draft, "itd-") || !strings.Contains(r.Reason, "users read") {
		t.Fatalf("a user-visible remedy is handed back by the judgement and promoted: %+v", r)
	}
	if runCount(t, repo) != 0 || res.Start != nil {
		t.Fatalf("no lane opens for a handed-back issue: %+v", res)
	}
	if res.Judging == nil || res.Judging.Issue != eligibleIssue {
		t.Fatalf("the drain moves on to judge the next eligible issue: %+v", res.Judging)
	}
	next := judge(t, repo, Options{}, res, JudgementNo, "", "internal")
	if next.Start == nil || next.Lane.Issue != eligibleIssue {
		t.Fatalf("the next issue's lane opens; the handed-back one is never taken: %+v", next)
	}
	if len(next.HandBacks) != 1 || next.HandBacks[0].Issue != secondIssue {
		t.Fatalf("the summary carries the judgement's hand-back: %+v", next.HandBacks)
	}
}

// TestATrustBoundaryRemedyIsFlaggedAndNothingIsMinted: a yes of kind
// trust-rule is flagged as needing a decision record with the reason as its
// question; no draft is minted and no lane opens.
func TestATrustBoundaryRemedyIsFlaggedAndNothingIsMinted(t *testing.T) {
	repo := issueRepo(t)
	drafts := func() int {
		n, _ := os.ReadDir(filepath.Join(repo.Root(), ".abcd", "development", "intents", "drafts"))
		return len(n)
	}
	before := drafts()
	first, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	res := judge(t, repo, Options{}, first, JudgementYes, HandBackTrustRule, "may the renderer skip the owner check on an empty list?")
	r := routeOf(t, res.Routed, eligibleIssue)
	if r.From != DrainFromJudgement || r.Route != RouteDecisionRecord || !strings.Contains(r.Question, "owner check") || drafts() != before {
		t.Fatalf("a trust-boundary remedy is flagged with its question and nothing minted: %+v", r)
	}
	if runCount(t, repo) != 0 || res.Start != nil || res.Stopped != DrainStoppedEmpty {
		t.Fatalf("no lane opens, and with nothing eligible left the drain ends: %+v", res)
	}
}

// TestTheJudgementNeverMakesAnIneligibleIssueEligible: an issue that leaves
// the eligible set while its judgement is out gets no lane, whatever the
// answer says.
func TestTheJudgementNeverMakesAnIneligibleIssueEligible(t *testing.T) {
	repo := issueRepo(t)
	first, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	matches, _ := filepath.Glob(filepath.Join(repo.Root(), ".abcd", "work", "issues", "open", eligibleIssue+"-*.md"))
	if len(matches) != 1 {
		t.Fatalf("one open record for %s", eligibleIssue)
	}
	raw, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `severity: "minor"`) {
		t.Fatalf("the record's severity line is not the shape this test rewrites:\n%s", raw)
	}
	if err := os.WriteFile(matches[0], []byte(strings.Replace(string(raw), `severity: "minor"`, `severity: "major"`, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	res := judge(t, repo, Options{}, first, JudgementNo, "", "internal")
	if res.Start != nil || runCount(t, repo) != 0 {
		t.Fatalf("a no over an issue no longer eligible opens nothing: %+v", res)
	}
	if res.Judged == nil || res.Judged.Applied || !strings.Contains(res.Judged.Note, "no longer eligible") {
		t.Fatalf("the judgement is recorded as not applied, saying why: %+v", res.Judged)
	}
	if routeOf(t, res.Flags, eligibleIssue).Rule != string(capture.RuleSeverity) {
		t.Fatalf("the issue is handed back by the rule instead: %+v", res.Flags)
	}
}

// TestAJudgementAnswerIsValidatedAndARefusalWritesNothing: an answer that is
// not the strict shape, not for the issue or the remedy the request named, or
// not at the path the drain named, is refused, and the drain still awaits it.
func TestAJudgementAnswerIsValidatedAndARefusalWritesNothing(t *testing.T) {
	repo := issueRepo(t)
	first, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	j := first.Judging
	statePath := filepath.Join(repo.Root(), filepath.FromSlash(DrainStateRel))
	stateBefore, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	good := func() map[string]any { return judgementAnswer(j, JudgementNo, "", "internal") }
	cases := map[string]func(map[string]any){
		"another issue":        func(a map[string]any) { a["issue"] = majorIssue },
		"another remedy":       func(a map[string]any) { a["remedy_sha256"] = strings.Repeat("0", 64) },
		"a schema version":     func(a map[string]any) { a["schema_version"] = 2 },
		"an answer":            func(a map[string]any) { a["answer"] = "maybe" },
		"a yes without a kind": func(a map[string]any) { a["answer"] = JudgementYes },
		"a yes of a lane kind": func(a map[string]any) { a["answer"], a["kind"] = JudgementYes, HandBackDesignFinding },
		"a no with a kind":     func(a map[string]any) { a["kind"] = HandBackUserVisible },
		"no reason":            func(a map[string]any) { a["reason"] = "  " },
		"an unknown field":     func(a map[string]any) { a["confidence"] = 0.9 },
	}
	for name, mutate := range cases {
		a := good()
		mutate(a)
		path := writeAnswer(t, repo, j, a)
		if _, err := Drain(repo.Root(), Options{}, DrainOptions{Judgement: path}); err == nil {
			t.Errorf("%s: the answer is accepted", name)
		}
	}
	other := filepath.Join(repo.Root(), ".abcd", ".work.local", "elsewhere.json")
	data, _ := json.Marshal(good())
	if err := os.WriteFile(other, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Drain(repo.Root(), Options{}, DrainOptions{Judgement: other}); err == nil || !strings.Contains(err.Error(), j.Answer) {
		t.Errorf("an answer at another path is refused naming the path the drain named: %v", err)
	}
	if after, _ := os.ReadFile(statePath); string(after) != string(stateBefore) || runCount(t, repo) != 0 {
		t.Fatalf("a refused answer writes nothing")
	}
	if res := judge(t, repo, Options{}, first, JudgementNo, "", "internal"); res.Start == nil {
		t.Fatalf("the drain still awaits the judgement, and a good answer opens the lane: %+v", res)
	}
}

// TestAJudgementIsRefusedWithoutADrainAwaitingOne: --judgement on a checkout
// whose drain awaits none is refused, and writes no drain state.
func TestAJudgementIsRefusedWithoutADrainAwaitingOne(t *testing.T) {
	repo := issueRepo(t)
	if err := os.MkdirAll(filepath.Join(repo.Root(), ".abcd", ".work.local", "run"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(repo.Root(), filepath.FromSlash(DrainJudgementAnswerRel))
	if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Drain(repo.Root(), Options{}, DrainOptions{Judgement: path}); err == nil || !strings.Contains(err.Error(), "awaits no judgement") {
		t.Fatalf("a judgement no drain awaits is refused: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo.Root(), filepath.FromSlash(DrainStateRel))); !os.IsNotExist(err) {
		t.Fatalf("the refusal wrote the drain's state: %v", err)
	}
}

// TestARemedyRewrittenWhileJudgedIsJudgedAgain: the answer binds to the
// remedy the request showed; once the remedy changes the answer is refused,
// and the next move asks again over the new remedy.
func TestARemedyRewrittenWhileJudgedIsJudgedAgain(t *testing.T) {
	repo := issueRepo(t)
	first, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := capture.SetRemedy(capture.RemedyRequest{RepoRoot: repo.Root(), ID: eligibleIssue, Remedy: "Print the empty list as a dash users see."}); err != nil {
		t.Fatal(err)
	}
	path := writeAnswer(t, repo, first.Judging, judgementAnswer(first.Judging, JudgementNo, "", "internal"))
	if _, err := Drain(repo.Root(), Options{}, DrainOptions{Judgement: path}); err == nil || !strings.Contains(err.Error(), "remedy") {
		t.Fatalf("an answer over a remedy since rewritten is refused: %v", err)
	}
	again, err := Drain(repo.Root(), Options{}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if again.Judging == nil || again.Judging.RemedySHA256 == first.Judging.RemedySHA256 || runCount(t, repo) != 0 {
		t.Fatalf("the next move asks again over the new remedy: %+v", again.Judging)
	}
	req, _ := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(again.Judging.Request)))
	if !strings.Contains(string(req), "a dash users see") {
		t.Fatalf("the new request carries the new remedy:\n%s", req)
	}
}

// TestAJudgementAnsweredDuringAPauseIsRecorded: a judgement asked inside a
// window and answered while the drain is paused is applied and recorded in
// the drain's state, so the move after the pause does not ask it again.
func TestAJudgementAnsweredDuringAPauseIsRecorded(t *testing.T) {
	repo := drainRepo(t)
	c := &clock{t: time.Date(2026, 9, 30, 9, 0, 0, 0, time.UTC)}
	pace := "1/5"
	first, err := Drain(repo.Root(), Options{Now: c.now, Pace: &pace}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if first.Judging == nil || first.Judging.Issue != secondIssue {
		t.Fatalf("the drain asks the judgement inside its window: %+v", first.Judging)
	}

	c.t = c.t.Add(2 * time.Minute)
	paused, err := Drain(repo.Root(), Options{Now: c.now}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if paused.NextEligibleAt == nil || paused.Judging == nil || paused.Judging.Issue != secondIssue {
		t.Fatalf("the window elapses with the judgement still awaited: %+v", paused)
	}

	c.t = c.t.Add(time.Minute)
	res := judge(t, repo, Options{Now: c.now}, paused, JudgementYes, HandBackUserVisible, "the page gains a flag users read")
	if res.NextEligibleAt == nil || res.Start != nil || len(res.Routed) != 1 {
		t.Fatalf("the answer is applied during the pause and opens nothing: %+v", res)
	}
	st, _, err := readDrain(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if st.Judging != nil || len(st.Judgements) != 1 || st.Judgements[0].Issue != secondIssue || len(st.HandBacks) != 1 || st.HandBacks[0].Issue != secondIssue {
		t.Fatalf("the drain's state records the judgement answered during the pause: judging %+v, judgements %+v, hand-backs %+v",
			st.Judging, st.Judgements, st.HandBacks)
	}

	c.t = paused.NextEligibleAt.Add(time.Second)
	after, err := Drain(repo.Root(), Options{Now: c.now}, DrainOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if after.Judging == nil || after.Judging.Issue != eligibleIssue {
		t.Fatalf("after the pause the drain asks the next issue, not the one already judged: %+v", after.Judging)
	}
}
