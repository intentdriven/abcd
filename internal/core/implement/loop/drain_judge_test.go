package loop

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/runner"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// The routed drain judge (spc-2610161044461448 step 14; iss-2610161015422306;
// dcn-2610161040390001): a drain-judge role routed to a runner is asked the
// judgement and its answer taken in the same move, abcd computing the remedy
// digest itself, so no person writes the answer; with no route the host
// request is written as it always was.

// judgeRoutedConfig routes the drain's judge to the claude runner on the machine.
const judgeRoutedConfig = `{"roles":{"drain-judge":{"runner":"claude"}},"runner":{"claude":{}}}`

// judgeEnv is newDriveEnv with git beside the fake harness, since a drain
// move runs git where the fake's PATH would hide it.
func judgeEnv(t *testing.T, mode string) driveEnv {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git on PATH")
	}
	return newDriveEnv(t, mode, "claude")
}

// TestARoutedDrainJudgesAnswerIsTakenWithAbcdComputingTheDigest: the judge
// routed to a runner answers no, with no digest in its answer, and the same
// move takes it, records the digest abcd computed over the remedy, names the
// runner that ran it, and opens the lane, with no --judgement given.
func TestARoutedDrainJudgesAnswerIsTakenWithAbcdComputingTheDigest(t *testing.T) {
	repo := issueRepo(t)
	e := judgeEnv(t, "judge-no")
	tr := &memTranscripts{}
	res, err := Drain(repo.Root(), Options{Runners: runnerConfig(t, judgeRoutedConfig, "")}, DrainOptions{Transcripts: tr})
	if err != nil {
		t.Fatal(err)
	}
	if res.Judging != nil {
		t.Fatalf("no judgement is left awaiting a person: %+v", res.Judging)
	}
	j := res.Judged
	if j == nil || j.Issue != eligibleIssue || j.Answer != JudgementNo || !j.Applied {
		t.Fatalf("the judge's no is taken in this move: %+v", j)
	}
	if j.RemedySHA256 != remedyDigest(issueRemedy) {
		t.Fatalf("abcd computes the remedy's digest itself: %q", j.RemedySHA256)
	}
	if j.Route == nil || j.Route.Asked != runner.Claude || j.Route.Ran != runner.Claude || j.Route.Model != "fake-model" {
		t.Fatalf("the judgement names the runner that ran it: %+v", j.Route)
	}
	if res.Start == nil || res.Lane == nil || res.Lane.Issue != eligibleIssue {
		t.Fatalf("a no opens the issue's lane in the same move: %+v", res)
	}
	if len(tr.stored) != 1 || tr.stored[0] != "claude "+RoleDrainJudge+" true" {
		t.Fatalf("the judge's transcript is stored: %v", tr.stored)
	}
	brief, err := os.ReadFile(filepath.Join(e.log, "claude.judge-brief.md"))
	if err != nil || len(brief) == 0 {
		t.Fatalf("the judge's brief is written: %v", err)
	}
	for _, not := range []string{"remedy_sha256", "abcd drain --judgement"} {
		if strings.Contains(string(brief), not) {
			t.Errorf("the judge's brief asks for no %q:\n%s", not, brief)
		}
	}
	if !strings.Contains(string(brief), drainJudgeAnswerName) {
		t.Errorf("the judge's brief names where its answer goes:\n%s", brief)
	}
	argv := e.argv(t, "claude")
	if !strings.Contains(strings.Join(argv, " "), "drain-judge") {
		t.Fatalf("the runner is launched as the drain judge: %v", argv)
	}
}

// TestARoutedDrainJudgesYesHandsTheIssueBackInTheSameMove: a yes from the
// routed judge routes the issue as a host's yes is routed, and no lane opens.
func TestARoutedDrainJudgesYesHandsTheIssueBackInTheSameMove(t *testing.T) {
	repo := issueRepo(t)
	judgeEnv(t, "judge-yes")
	res, err := Drain(repo.Root(), Options{Runners: runnerConfig(t, judgeRoutedConfig, "")}, DrainOptions{Transcripts: &memTranscripts{}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Judged == nil || res.Judged.Answer != JudgementYes || !res.Judged.Applied {
		t.Fatalf("the judge's yes is taken in this move: %+v", res.Judged)
	}
	r := routeOf(t, res.Routed, eligibleIssue)
	if r.From != DrainFromJudgement || r.Route != RouteDecisionRecord {
		t.Fatalf("a trust-rule yes is flagged as needing a decision record: %+v", r)
	}
	if res.Start != nil || runCount(t, repo) != 0 || res.Judging != nil {
		t.Fatalf("no lane opens and nothing awaits: %+v", res)
	}
}

// TestWithNoRouteTheDrainJudgementIsTheHostsRequest: with the judge on the
// host, the request is written for the host as it always was, and no runner
// is launched, though the build roles are routed.
func TestWithNoRouteTheDrainJudgementIsTheHostsRequest(t *testing.T) {
	repo := issueRepo(t)
	e := judgeEnv(t, "judge-no")
	res, err := Drain(repo.Root(), Options{Runners: runnerConfig(t, claudeImplements, "")}, DrainOptions{Transcripts: &memTranscripts{}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Judging == nil || res.Judging.Answer != DrainJudgementAnswerRel || res.Judged != nil || res.Start != nil {
		t.Fatalf("the host request is written and awaited: %+v", res)
	}
	if !strings.Contains(res.Next, "abcd drain --judgement") {
		t.Fatalf("next names the host's answer: %s", res.Next)
	}
	if _, err := os.Stat(filepath.Join(e.log, "claude.argv.json")); err == nil {
		t.Fatal("no runner is launched for a judge left on the host")
	}
}

// TestARoutedJudgeThatDoesNotAnswerLeavesTheHostsRequest: a runner that
// writes no answer is a fallback: the receipt is kept in the drain's state,
// and the host request stands for a person or the host to answer.
func TestARoutedJudgeThatDoesNotAnswerLeavesTheHostsRequest(t *testing.T) {
	repo := issueRepo(t)
	judgeEnv(t, "silent")
	res, err := Drain(repo.Root(), Options{Runners: runnerConfig(t, judgeRoutedConfig, "")}, DrainOptions{Transcripts: &memTranscripts{}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Judging == nil || res.Judged != nil || res.Start != nil {
		t.Fatalf("the host request stands: %+v", res)
	}
	if !strings.Contains(res.Next, "abcd drain --judgement") || !strings.Contains(res.Next, "claude") {
		t.Fatalf("next names the fallback and the host's answer: %s", res.Next)
	}
	raw, err := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(DrainStateRel)))
	if err != nil {
		t.Fatal(err)
	}
	var st DrainState
	if err := json.Unmarshal(raw, &st); err != nil {
		t.Fatal(err)
	}
	if len(st.JudgeFallbacks) != 1 || st.JudgeFallbacks[0].Role != RoleDrainJudge || st.JudgeFallbacks[0].Asked != runner.Claude {
		t.Fatalf("the fallback receipt is kept in the drain's state: %+v", st.JudgeFallbacks)
	}
}

// TestARoutedNoOpensTheLaneOnTheHoldTakenToAskIt: on origin, the lane a
// routed no opens starts on the reservation the drain took before it asked,
// which this move's view could not show, and pushes no second reservation.
func TestARoutedNoOpensTheLaneOnTheHoldTakenToAskIt(t *testing.T) {
	repo, bare := reservingDrainRepo(t, "commands/drain.md")
	judgeEnv(t, "judge-no")
	res, err := Drain(repo.Root(), Options{Runners: runnerConfig(t, judgeRoutedConfig, "")}, DrainOptions{Transcripts: &memTranscripts{}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Start == nil || res.Lane == nil || res.Lane.Issue != secondIssue {
		t.Fatalf("the routed no opens the lane: %+v", res)
	}
	if n := gitIn(t, bare, "rev-list", "--count", claim.RefPrefix+secondIssue); n != "1" {
		t.Fatalf("the lane starts on the judgement's reservation, one commit, not %s", n)
	}
	if note := recordNote(t, repo, res.Start.RunID, StageReservation); !strings.Contains(note, "before its judgement") {
		t.Fatalf("the run record names the hold it started on: %q", note)
	}
}

// judgeDirOf is where the fake judge was started and what it found there.
type judgeDirOf struct {
	Cwd  string   `json:"cwd"`
	Held []string `json:"held"`
	Mode string   `json:"mode"`
}

func readJudgeDir(t *testing.T, e driveEnv) judgeDirOf {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(e.log, "claude.judge-dir.json"))
	if err != nil {
		t.Fatalf("the judge was not started: %v", err)
	}
	var d judgeDirOf
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	return d
}

// TestTheRoutedJudgeRunsInADirectoryOfItsOwn: the routed judge is started in
// a new private directory outside the checkout, holding its brief alone, and
// is handed no directory of the checkout's to write; the directory is gone
// once the move has read the answer, and nothing of the judge's is left in
// the checkout. Its answer steers an unattended lane, and its brief quotes a
// record a stranger may have written, so it may write nothing the drain or a
// peer relies on.
func TestTheRoutedJudgeRunsInADirectoryOfItsOwn(t *testing.T) {
	repo := issueRepo(t)
	e := judgeEnv(t, "judge-no")
	res, err := Drain(repo.Root(), Options{Runners: runnerConfig(t, judgeRoutedConfig, "")}, DrainOptions{Transcripts: &memTranscripts{}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Judged == nil || res.Judged.Answer != JudgementNo || res.Start == nil {
		t.Fatalf("the judge's no is taken and the lane opens: %+v", res)
	}
	d := readJudgeDir(t, e)
	checkout := fsutil.RealExistingPath(repo.Root())
	cwd := fsutil.RealExistingPath(d.Cwd)
	if fsutil.PathWithin(cwd, checkout, fsutil.CaseFoldingFS()) {
		t.Fatalf("the judge runs inside the checkout: %s in %s", cwd, checkout)
	}
	if !slices.Equal(d.Held, []string{drainJudgeBriefName}) {
		t.Fatalf("the judge's directory holds its brief alone: %v", d.Held)
	}
	if d.Mode != "-rwx------" {
		t.Fatalf("the judge's directory is private to this account: %s", d.Mode)
	}
	if _, err := os.Lstat(d.Cwd); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the judge's directory is removed once its answer is read: %v", err)
	}
	for _, a := range e.argv(t, "claude") {
		v, ok := strings.CutPrefix(a, "--add-dir=")
		if ok && fsutil.PathWithin(fsutil.RealExistingPath(v), checkout, fsutil.CaseFoldingFS()) {
			t.Fatalf("the judge is handed a directory of the checkout's: %s", a)
		}
	}
	run := filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir))
	es, err := os.ReadDir(run)
	if err != nil {
		t.Fatal(err)
	}
	for _, en := range es {
		if strings.HasPrefix(en.Name(), "drain-judgement.runner") {
			t.Fatalf("the judge's brief or answer is left in the checkout: %s", en.Name())
		}
	}
}

// launches counts the times the fake harness was started.
func launches(t *testing.T, e driveEnv, harness string) int {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(e.log, harness+".launches"))
	if errors.Is(err, os.ErrNotExist) {
		return 0
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Count(string(raw), "\n")
}

// TestASilentRoutedJudgeIsAskedOncePerRequest: a routed judge that does not
// answer is asked once for the request it was started on; the polls that
// follow while the host's request stands start no runner and record no
// further fallback, and say why the request is the host's.
func TestASilentRoutedJudgeIsAskedOncePerRequest(t *testing.T) {
	repo := issueRepo(t)
	e := judgeEnv(t, "silent")
	o := Options{Runners: runnerConfig(t, judgeRoutedConfig, "")}
	var last DrainResult
	for i := range 3 {
		res, err := Drain(repo.Root(), o, DrainOptions{Transcripts: &memTranscripts{}})
		if err != nil {
			t.Fatalf("poll %d: %v", i+1, err)
		}
		if res.Judging == nil || res.Judging.Issue != eligibleIssue || res.Judged != nil {
			t.Fatalf("poll %d: the host's request stands: %+v", i+1, res)
		}
		last = res
	}
	if n := launches(t, e, "claude"); n != 1 {
		t.Fatalf("the judge's runner is started once per request, not on every poll: %d launches", n)
	}
	st, _, err := readDrain(repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	if len(st.JudgeFallbacks) != 1 {
		t.Fatalf("one fallback receipt for the one request: %+v", st.JudgeFallbacks)
	}
	if !strings.Contains(last.Next, "abcd drain --judgement") || !strings.Contains(last.Next, "already") {
		t.Fatalf("a later poll names the host's answer and why the judge is not asked again: %s", last.Next)
	}
}

// TestTheDrainKeepsTheLatestJudgeFallbacksAlone: the fallback receipts a
// drain keeps are bounded, the oldest dropped and the latest kept last.
func TestTheDrainKeepsTheLatestJudgeFallbacksAlone(t *testing.T) {
	var st DrainState
	for i := range maxJudgeFallbacks + 8 {
		keepJudgeFallback(&st, runner.FallbackReceipt{Role: RoleDrainJudge, Detail: strconv.Itoa(i)})
	}
	if len(st.JudgeFallbacks) != maxJudgeFallbacks {
		t.Fatalf("the receipts are bounded at %d: %d", maxJudgeFallbacks, len(st.JudgeFallbacks))
	}
	if first, last := st.JudgeFallbacks[0].Detail, st.JudgeFallbacks[maxJudgeFallbacks-1].Detail; first != "8" || last != strconv.Itoa(maxJudgeFallbacks+7) {
		t.Fatalf("the oldest are dropped and the latest kept last: %s .. %s", first, last)
	}
}

// TestTheRoutedJudgeMayReadAndWriteOnlyItsOwnDirectory: the routed judge is
// granted the read of its own directory and the write of its answer file
// alone, each by its real absolute path in the harness's permission-rule form,
// and refused both over the checkout; no tool is granted on every path.
func TestTheRoutedJudgeMayReadAndWriteOnlyItsOwnDirectory(t *testing.T) {
	repo := issueRepo(t)
	e := judgeEnv(t, "judge-no")
	if _, err := Drain(repo.Root(), Options{Runners: runnerConfig(t, judgeRoutedConfig, "")}, DrainOptions{Transcripts: &memTranscripts{}}); err != nil {
		t.Fatal(err)
	}
	dir := fsutil.RealExistingPath(readJudgeDir(t, e).Cwd)
	checkout := fsutil.RealExistingPath(repo.Root())
	var allowed, denied []string
	for _, a := range e.argv(t, "claude") {
		if v, ok := strings.CutPrefix(a, "--allowedTools="); ok {
			allowed = append(allowed, v)
		}
		if v, ok := strings.CutPrefix(a, "--disallowedTools="); ok {
			denied = append(denied, v)
		}
	}
	if want := "Read(/" + dir + "/**),Edit(/" + filepath.Join(dir, drainJudgeAnswerName) + ")"; !slices.Equal(allowed, []string{want}) {
		t.Fatalf("the judge is granted its own directory alone:\n got %q\nwant %q", allowed, want)
	}
	if want := "Read(/" + checkout + "/**),Edit(/" + checkout + "/**)"; !slices.Equal(denied, []string{want}) {
		t.Fatalf("the judge is refused the checkout:\n got %q\nwant %q", denied, want)
	}
}

// TestAJudgePathIsWrittenAsALiteralRule: a character the rule's pattern
// syntax reads as a wildcard is escaped, so a rule names its path alone.
func TestAJudgePathIsWrittenAsALiteralRule(t *testing.T) {
	if got := absPathRule(`/tmp/a*b/c?[d]\e`); got != `//tmp/a\*b/c\?\[d\]\\e` {
		t.Fatalf("absPathRule = %q", got)
	}
}

// TestAJudgeThatNeverStartedIsAskedOnTheNextMove: when the judge's directory
// cannot be made, no runner starts, so the request is not marked as asked of
// it, and the next move, the directory made, starts the judge and takes its
// answer rather than saying it was already asked.
func TestAJudgeThatNeverStartedIsAskedOnTheNextMove(t *testing.T) {
	repo := issueRepo(t)
	e := judgeEnv(t, "judge-no")
	cfg := runnerConfig(t, judgeRoutedConfig, "")
	tmp := os.Getenv("TMPDIR")
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "gone"))
	first, err := Drain(repo.Root(), Options{Runners: cfg}, DrainOptions{Transcripts: &memTranscripts{}})
	if err != nil {
		t.Fatal(err)
	}
	if first.Judging == nil || first.Judging.RoutedAsked || !strings.Contains(first.Next, "was not started") {
		t.Fatalf("a judge that never started is not marked as asked: %+v next %q", first.Judging, first.Next)
	}
	if n := launches(t, e, "claude"); n != 0 {
		t.Fatalf("no runner started: %d", n)
	}
	t.Setenv("TMPDIR", tmp)
	again, err := Drain(repo.Root(), Options{Runners: cfg}, DrainOptions{Transcripts: &memTranscripts{}})
	if err != nil {
		t.Fatal(err)
	}
	if again.Judged == nil || again.Judged.Answer != JudgementNo || strings.Contains(again.Next, "already asked") {
		t.Fatalf("the next move starts the judge and takes its answer: judged %+v next %q", again.Judged, again.Next)
	}
}
