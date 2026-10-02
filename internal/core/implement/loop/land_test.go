package loop

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/gittest"
)

// queueRuleset is a ruleset mirror that gates the default branch through a
// merge queue merging with method, behind a pull-request rule requiring one
// approving review, as .abcd/work/rulesets/ holds it.
func queueRuleset(method string) string {
	return `{"bypass_actors":[],"conditions":{"ref_name":{"exclude":[],"include":["~DEFAULT_BRANCH"]}},` +
		`"enforcement":"active","name":"main protection","rules":[{"type":"deletion"},` +
		`{"parameters":{"required_approving_review_count":1},"type":"pull_request"},` +
		`{"parameters":{"merge_method":"` + method + `","grouping_strategy":"ALLGREEN"},"type":"merge_queue"}],"target":"branch"}` + "\n"
}

// unreviewedQueueRuleset gates the default branch through a merge queue with
// no rule requiring a person's approval.
func unreviewedQueueRuleset(method string) string {
	return `{"bypass_actors":[],"conditions":{"ref_name":{"exclude":[],"include":["~DEFAULT_BRANCH"]}},` +
		`"enforcement":"active","name":"main protection","rules":[{"type":"deletion"},` +
		`{"parameters":{"merge_method":"` + method + `","grouping_strategy":"ALLGREEN"},"type":"merge_queue"}],"target":"branch"}` + "\n"
}

// noQueueRuleset gates the default branch with no merge queue.
const noQueueRuleset = `{"bypass_actors":[],"conditions":{"ref_name":{"exclude":[],"include":["~DEFAULT_BRANCH"]}},` +
	`"enforcement":"active","name":"main protection","rules":[{"type":"deletion"},{"type":"non_fast_forward"}],"target":"branch"}` + "\n"

// stubGH is a forge client that records every call in gh.log beside it and
// answers from files there: pr.json (the open pull requests), body.md (the
// body the forge holds), state (the pull request's state), footer (a line the
// "harness" appends to a body at creation), refuse-disarm (present, the forge
// refuses to withdraw an armed merge). It never reaches a network.
const stubGH = `#!/bin/sh
d="$(cd "$(dirname "$0")" && pwd)"
printf '%s\n' "$*" >> "$d/gh.log"
body_from() {
  while [ $# -gt 0 ]; do
    case "$1" in
      --body-file) shift; cat "$1" > "$d/body.md" ;;
      --body-file=*) cat "${1#--body-file=}" > "$d/body.md" ;;
    esac
    shift
  done
}
case "$1 $2" in
  "pr list") if [ -f "$d/pr.json" ]; then cat "$d/pr.json"; else echo '[]'; fi ;;
  "pr create")
    body_from "$@"
    if [ -f "$d/footer" ]; then cat "$d/footer" >> "$d/body.md"; fi
    echo '[{"number":7,"url":"https://example.com/o/r/pull/7"}]' > "$d/pr.json"
    echo "https://example.com/o/r/pull/7" ;;
  "pr view")
    case "$*" in
      *state*) if [ -f "$d/state" ]; then cat "$d/state"; else echo OPEN; fi ;;
      *) cat "$d/body.md" ;;
    esac ;;
  "pr edit") body_from "$@" ;;
  "pr merge")
    case "$*" in
      *--disable-auto*) if [ -f "$d/refuse-disarm" ]; then echo "stub gh: the forge refused" >&2; exit 1; fi ;;
    esac ;;
  "pr close") : ;;
  *) echo "stub gh: unexpected call: $*" >&2; exit 1 ;;
esac
`

// landFixture is a repository whose default branch is on a local bare
// remote, with a stub forge client first on PATH, an open capture the lane
// will fix, and the ruleset mirror given.
type landFixture struct {
	repo   *gittest.Repo
	bare   string
	gh     string
	issue  string
	runID  string
	stages Stages
	// model is the model the lane's receipt reports.
	model string
}

func newLandFixture(t *testing.T, ruleset string) *landFixture {
	t.Helper()
	files := map[string]string{}
	if ruleset != "" {
		files[".abcd/work/rulesets/main-protection.json"] = ruleset
	}
	return newLandFixtureWith(t, files)
}

// newLandFixtureWith is newLandFixture with the files given (the ruleset
// mirror, a CODEOWNERS file) committed at the lane's base.
func newLandFixtureWith(t *testing.T, files map[string]string) *landFixture {
	t.Helper()
	repo := loopRepo(t, readyIntent("impact: additive\n", settledQuestions), specWithSteps(""))
	for _, k := range []string{"GIT_AUTHOR_NAME", "GIT_COMMITTER_NAME"} {
		t.Setenv(k, "Pat Example")
	}
	for _, k := range []string{"GIT_AUTHOR_EMAIL", "GIT_COMMITTER_EMAIL"} {
		t.Setenv(k, "pat@example.com")
	}
	repo.Write("AGENTS.md", agentsMarked)
	for name, body := range files {
		repo.Write(name, body)
	}
	c, err := capture.Capture(capture.CaptureRequest{RepoRoot: repo.Root(), Text: "The widget refuses a blank name.",
		Severity: "minor", Category: "ux", Source: "agent-observation", FoundDuring: "a landing test",
		Remedy: "accept a blank name as absent"})
	if err != nil {
		t.Fatalf("capturing the issue the lane fixes: %v", err)
	}
	repo.Commit("the record")

	bare := filepath.Join(t.TempDir(), "origin.git")
	repo.Git("init", "-q", "--bare", "--initial-branch=main", bare)
	repo.Git("remote", "add", "origin", bare)
	repo.Git("push", "-q", "origin", "main")
	repo.Git("fetch", "-q", "origin")

	gh := t.TempDir()
	if err := os.WriteFile(filepath.Join(gh, "gh"), []byte(stubGH), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", gh+string(os.PathListSeparator)+os.Getenv("PATH"))

	start, err := Start(repo.Root(), "itd-10", Options{})
	if err != nil {
		t.Fatal(err)
	}
	return &landFixture{repo: repo, bare: bare, gh: gh, issue: c.ID, runID: start.RunID, stages: DefaultStages(),
		model: "claude-test-5"}
}

// validated drives the run's lane through its implement stage, with a receipt
// that declares the capture fixed by the lane's commit, and a passing round.
func (f *landFixture) validated(t *testing.T) Lane {
	t.Helper()
	repo, id := f.repo, f.runID
	stepTo(t, repo, id, f.stages, StageImplement)
	res, err := advance(repo.Root(), id, f.stages, Options{})
	if err != nil || res.Awaiting == nil {
		t.Fatalf("the implement stage awaits an implementer: %+v %v", res, err)
	}
	l := currentLane(t, repo, id)
	dir := filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), id, l.ID)
	sha := laneCommit(t, repo, l, "one.txt")
	rc := goodReceipt(t, id, l, dir, sha)
	path := writeReceipt(t, dir, map[string]any{
		"schema_version": rc.SchemaVersion, "run_id": rc.RunID, "lane": rc.Lane, "branch": rc.Branch,
		"commits": rc.Commits, "definition_of_done": rc.DefinitionOfDone, "report": rc.Report, "model": f.model,
		"resolves": []map[string]string{{"issue": f.issue, "commit": sha, "note": "a blank name reads as absent",
			"impact": "fix", "grounds": "pursued: a blank name is accepted; shown wrong if the widget still refuses one"}},
	})
	if _, err := Receipt(repo.Root(), id, path, f.stages, Options{}); err != nil {
		t.Fatalf("a receipt declaring a fixed capture verifies: %v", err)
	}
	passRound(t, repo, id, f.stages, RoleRuthless, RoleSecurity, RoleAuditor)
	return stepTo(t, repo, id, f.stages, StageLand)
}

// step advances the run once and fails the test on an error.
func (f *landFixture) step(t *testing.T) StepResult {
	t.Helper()
	res, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
	if err != nil {
		t.Fatalf("landing step: %v", err)
	}
	return res
}

// ghLog is every call the stub forge client received, one per line.
func (f *landFixture) ghLog(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(f.gh, "gh.log"))
	if errors.Is(err, os.ErrNotExist) {
		return ""
	}
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// remoteBranch is the bare remote's tip of branch, or "" when it has none.
func (f *landFixture) remoteBranch(t *testing.T, branch string) string {
	t.Helper()
	out := f.repo.Git("ls-remote", "origin", "refs/heads/"+branch)
	if out == "" {
		return ""
	}
	return strings.Fields(out)[0]
}

// preflighted mints the preflight receipt for sha in the lane's worktree, as
// `make preflight` does on a clean tree.
func preflighted(t *testing.T, l Lane, sha string) {
	t.Helper()
	dir := filepath.Join(l.Worktree, ".abcd", ".work.local", "preflight-receipts")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, sha), []byte("commit "+sha+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// merged merges the lane's pushed head into the remote's default branch, as
// the merge queue does, and tells the stub forge the pull request merged.
func (f *landFixture) merged(t *testing.T, head string) {
	t.Helper()
	f.repo.Git("push", "-q", "origin", head+":refs/heads/main")
	if err := os.WriteFile(filepath.Join(f.gh, "state"), []byte("MERGED\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestTheLandingClosesTheSpecResolvesTheCapturesAndArmsTheMerge is criterion
// 6: after the validators pass, the loop closes the spec (shipping the intent,
// with the audit the closing lane took ingested) and resolves the capture the
// lane fixed with its commit, both committed on the lane's branch with their
// trailers; it requires the preflight receipt before it pushes, opens the pull
// request through the forge client with a body built from the records, arms
// the merge by the ruleset's method, waits for the merge, and cleans the lane
// up only once its commit is on the default branch.
func TestTheLandingClosesTheSpecResolvesTheCapturesAndArmsTheMerge(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.validated(t)
	implHead := l.HeadSHA

	// The records: spec close and capture resolve, committed on the lane.
	for range 2 {
		if res := f.step(t); res.PerformedStage != "" || res.Stage != StageLand {
			t.Fatalf("a landing step leaves the lane at land until it is done: %+v", res)
		}
	}
	l = currentLane(t, f.repo, f.runID)
	if l.HeadSHA == implHead {
		t.Fatalf("the landing's records are committed on the lane: head still %s", implHead)
	}
	msg := f.repo.Git("-C", l.Worktree, "log", "-1", "--format=%B", l.HeadSHA)
	for _, want := range []string{"Delivers: itd-10", "Resolves: " + f.issue, "Assisted-by: Claude:claude-test-5"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("the records commit carries %q:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "Assisted-by: None") {
		t.Fatalf("the records commit carries model prose and never claims no assistance:\n%s", msg)
	}
	files := f.repo.Git("-C", l.Worktree, "ls-tree", "-r", "--name-only", l.HeadSHA)
	for _, want := range []string{".abcd/development/specs/closed/spc-1-alpha.md", ".abcd/development/intents/shipped/itd-10-alpha.md"} {
		if !strings.Contains(files, want) {
			t.Fatalf("the spec is closed and the intent shipped in the landing change (%s missing):\n%s", want, files)
		}
	}
	if !strings.Contains(files, ".abcd/work/issues/resolved/"+f.issue) {
		t.Fatalf("the capture the lane fixed is resolved in the landing change:\n%s", files)
	}
	resolved := f.repo.Git("-C", l.Worktree, "show", l.HeadSHA+":"+issuePath(t, files, f.issue))
	if !strings.Contains(resolved, implHead[:12]) && !strings.Contains(resolved, implHead) {
		t.Fatalf("the capture is resolved with the lane's commit %s:\n%s", implHead, resolved)
	}
	shipped := f.repo.Git("-C", l.Worktree, "show", l.HeadSHA+":.abcd/development/intents/shipped/itd-10-alpha.md")
	if strings.Contains(shipped, "abcd-review: OWED") || !strings.Contains(shipped, "INGESTED") {
		t.Fatalf("the audit the closing lane took is ingested at the close, not owed again:\n%s", shipped)
	}

	// No preflight receipt: the lane waits for its full check (ruling DR6d-2),
	// nothing pushed.
	_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
	r := mustRefusal(t, err)
	if r.Stage != string(StageLand) || !r.Contention || !strings.Contains(r.Reason, "preflight receipt") || !strings.Contains(r.Remedy, "preflight") {
		t.Fatalf("a landing without the preflight receipt waits for it, naming it: %+v", r)
	}
	if got := f.remoteBranch(t, l.Branch); got != "" {
		t.Fatalf("nothing is pushed without the receipt, but the remote has %s", got)
	}

	preflighted(t, l, l.HeadSHA)
	f.step(t)
	if got := f.remoteBranch(t, l.Branch); got != l.HeadSHA {
		t.Fatalf("the lane's head is pushed once the receipt exists: remote %q, head %s", got, l.HeadSHA)
	}

	// The pull request, its body from the records through the outbound policy.
	f.step(t)
	log := f.ghLog(t)
	if !strings.Contains(log, "pr create") || !strings.Contains(log, "--base main") || !strings.Contains(log, "--head "+l.Branch) {
		t.Fatalf("the pull request is opened through the forge client against the default branch:\n%s", log)
	}
	body, err := os.ReadFile(filepath.Join(f.gh, "body.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"itd-10", "spc-1", "Resolves: " + f.issue, "Delivers: itd-10", "ruthless-reviewer SHIP"} {
		if !strings.Contains(string(body), want) {
			t.Fatalf("the body is built from the records (%q missing):\n%s", want, body)
		}
	}
	if l = currentLane(t, f.repo, f.runID); l.PR != 7 {
		t.Fatalf("the state records the pull request: %+v", l)
	}

	// Armed by the ruleset's method.
	f.step(t)
	if log := f.ghLog(t); !strings.Contains(log, "pr merge 7 --auto --merge") {
		t.Fatalf("the merge is armed by the ruleset's merge-queue method:\n%s", log)
	}

	// Not merged yet: the loop waits, and cleans nothing up.
	_, err = advance(f.repo.Root(), f.runID, f.stages, Options{})
	if r := mustRefusal(t, err); !r.Contention || !strings.Contains(r.Reason, "not on") {
		t.Fatalf("an unmerged lane waits for its merge: %+v", r)
	}
	if _, err := os.Stat(l.Worktree); err != nil {
		t.Fatalf("the lane is not cleaned up before its commit is on the default branch: %v", err)
	}

	f.merged(t, l.HeadSHA)
	res := f.step(t)
	if res.PerformedStage != StageLand || !res.Complete {
		t.Fatalf("once merged, the landing completes and the run with it: %+v", res)
	}
	if _, err := os.Stat(l.Worktree); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the lane's worktree is removed after the ancestor check: %v", err)
	}
	if out := f.repo.Git("branch", "--list", l.Branch); out != "" {
		t.Fatalf("the lane's branch is removed after the ancestor check: %q", out)
	}
}

// issuePath finds the resolved record of issue in a tree listing.
func issuePath(t *testing.T, files, issue string) string {
	t.Helper()
	for _, f := range strings.Split(files, "\n") {
		if strings.HasPrefix(f, ".abcd/work/issues/resolved/"+issue) {
			return f
		}
	}
	t.Fatalf("no resolved record for %s", issue)
	return ""
}

// landedToArmed takes a validated lane through its records, push, pull request
// and arming, and returns it.
func (f *landFixture) landedToArmed(t *testing.T) Lane {
	t.Helper()
	f.validated(t)
	f.step(t)
	f.step(t)
	l := currentLane(t, f.repo, f.runID)
	preflighted(t, l, l.HeadSHA)
	f.step(t)
	f.step(t)
	f.step(t)
	return currentLane(t, f.repo, f.runID)
}

// TestNothingIsPushedAfterArming is criterion 6's last clause: once the merge
// is armed, a commit added to the lane's branch is never pushed, and the
// cleanup's ancestor check is made against what was pushed.
func TestNothingIsPushedAfterArming(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.landedToArmed(t)
	pushed := f.remoteBranch(t, l.Branch)
	if pushed != l.HeadSHA {
		t.Fatalf("the lane was pushed before arming: remote %q, head %s", pushed, l.HeadSHA)
	}
	late := laneCommit(t, f.repo, l, "late.txt")
	preflighted(t, l, late)
	for range 3 {
		_, _ = advance(f.repo.Root(), f.runID, f.stages, Options{})
	}
	if got := f.remoteBranch(t, l.Branch); got != pushed {
		t.Fatalf("nothing is pushed after arming: the remote moved from %s to %s", pushed, got)
	}
	if n := strings.Count(f.ghLog(t), "pr merge"); n != 1 {
		t.Fatalf("the merge is armed once: %d arm calls\n%s", n, f.ghLog(t))
	}
}

// TestAPullRequestBodyThatArrivesDirtyIsReReadAndStripped is the outbound
// policy on the landing: the body the loop composes passes through the scrub,
// and after creating the pull request the loop re-reads what the forge holds
// and strips a footer the harness appended outside the loop's own text.
func TestAPullRequestBodyThatArrivesDirtyIsReReadAndStripped(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	// Built at runtime, and linking a host outside the reserved documentation
	// domains (a footer linking one reads as documentation): the committed file
	// carries no footer shape.
	footer := "Generated " + "with [a tool](https://" + "tool" + ".dev)\n"
	if err := os.WriteFile(filepath.Join(f.gh, "footer"), []byte(footer), 0o644); err != nil {
		t.Fatal(err)
	}
	f.landedToArmed(t)
	log := f.ghLog(t)
	if !strings.Contains(log, "pr edit 7") {
		t.Fatalf("a body that arrived carrying a footer is edited:\n%s", log)
	}
	body, err := os.ReadFile(filepath.Join(f.gh, "body.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "Generated "+"with") {
		t.Fatalf("the footer is stripped from the body the forge holds:\n%s", body)
	}
	if !strings.Contains(string(body), "Delivers: itd-10") {
		t.Fatalf("the strip keeps the body's own text:\n%s", body)
	}
}

// TestWithoutAMergeQueueThePullRequestIsLeftOpen is decision 3: where the
// ruleset gates no merge through a queue, the pull request is left open and no
// merge is armed.
func TestWithoutAMergeQueueThePullRequestIsLeftOpen(t *testing.T) {
	f := newLandFixture(t, noQueueRuleset)
	l := f.landedToArmed(t)
	if strings.Contains(f.ghLog(t), "pr merge") {
		t.Fatalf("no merge is armed without a merge queue:\n%s", f.ghLog(t))
	}
	if l.Landing == nil || !strings.Contains(l.Landing.Merge, "left open") {
		t.Fatalf("the state says the pull request is left open: %+v", l.Landing)
	}
}

// TestAClosedPullRequestIsRefusedAndNothingIsCleanedUp: a pull request closed
// without merging is refused loudly, and the lane's worktree and branch stay.
func TestAClosedPullRequestIsRefusedAndNothingIsCleanedUp(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	l := f.landedToArmed(t)
	if err := os.WriteFile(filepath.Join(f.gh, "state"), []byte("CLOSED\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
	if r := mustRefusal(t, err); r.Contention || !strings.Contains(r.Reason, "closed") {
		t.Fatalf("a pull request closed without merging is refused, not waited on: %+v", r)
	}
	if _, err := os.Stat(l.Worktree); err != nil {
		t.Fatalf("nothing is cleaned up: %v", err)
	}
}

// TestAKilledLandingResumesAtTheStepThatDidNotComplete is criterion 7 on the
// landing: a process killed after a landing step's effect and before its state
// write repeats the step on the next call, finding what it made rather than
// making it twice — one records commit, one pull request.
func TestAKilledLandingResumesAtTheStepThatDidNotComplete(t *testing.T) {
	f := newLandFixture(t, queueRuleset("MERGE"))
	killed := map[int]bool{}
	calls := 0
	stages := DefaultStages()
	for i := range stages {
		if stages[i].Name == StageLand {
			body := stages[i].Run
			stages[i].Run = func(c Context, lane *Lane) (Outcome, error) {
				calls++
				out, err := body(c, lane)
				// Kill the second call (the records commit) and the fifth (the
				// pull request) after their effect, before the state write.
				if err == nil && (calls == 2 || calls == 5) && !killed[calls] {
					killed[calls] = true
					return Outcome{}, errors.New("killed")
				}
				return out, err
			}
		}
	}
	f.validated(t)
	f.stages = stages
	f.step(t)
	if _, err := advance(f.repo.Root(), f.runID, stages, Options{}); err == nil {
		t.Fatal("the kill surfaces")
	}
	f.step(t)
	l := currentLane(t, f.repo, f.runID)
	commits := f.repo.Git("rev-list", "--count", l.BaseSHA+".."+l.HeadSHA)
	if commits != "2" {
		t.Fatalf("a killed records step is found, not made twice: %s commits past the base", commits)
	}
	preflighted(t, l, l.HeadSHA)
	f.step(t)
	if _, err := advance(f.repo.Root(), f.runID, stages, Options{}); err == nil {
		t.Fatal("the kill surfaces")
	}
	f.step(t)
	if n := strings.Count(f.ghLog(t), "pr create"); n != 1 {
		t.Fatalf("a killed pull-request step finds the pull request it opened: %d creates\n%s", n, f.ghLog(t))
	}
	if l = currentLane(t, f.repo, f.runID); l.PR != 7 {
		t.Fatalf("the resumed step records the pull request it found: %+v", l)
	}
}
