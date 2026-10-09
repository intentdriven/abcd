package loop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/drainrule"
	"github.com/intentdriven/abcd/internal/gittest"
)

// The issue key (decision 10 on itd-2609201916151817, for itd-82 scope 4): the
// loop takes `iss-N` as it takes `itd-N`, one lane per eligible issue, through
// the same worktree, brief, validators and landing.

const (
	eligibleIssue = "iss-2609300000000101"
	majorIssue    = "iss-2609300000000102"
	secondIssue   = "iss-2609300000000103"
	issueRemedy   = "Guard the empty list in the renderer and test it."
)

// fileIssue captures one open issue in the repository through the ledger's own
// writer, so the record is one the reader accepts.
func fileIssue(t *testing.T, repo *gittest.Repo, id string, sev capture.Severity, cat capture.Category, remedy string) {
	t.Helper()
	if _, err := capture.Capture(capture.CaptureRequest{RepoRoot: repo.Root(), Text: "The renderer panics on an empty list (" + id + ")",
		Severity: sev, Category: cat, Source: "manual-test", Slug: "renderer", FoundDuring: "the loop's tests", ForceID: id, Remedy: remedy}); err != nil {
		t.Fatalf("capture %s: %v", id, err)
	}
}

// issueRepo is briefRepo with the repository's own drain rule recorded and the
// issues filed, all committed on the default branch.
func issueRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	repo := briefRepo(t, agentsMarked)
	repo.Write(drainrule.ADRsRelDir+"/2609300000000001-drain-rule.md",
		"---\nid: adr-2609300000000001\nslug: drain-rule\nstatus: accepted\ndate: 2026-09-30\n"+drainrule.ProposalFrontmatter()+"---\n\n# The drain rule\n")
	fileIssue(t, repo, eligibleIssue, capture.SeverityMinor, capture.Category("bug"), issueRemedy)
	fileIssue(t, repo, majorIssue, capture.SeverityMajor, capture.Category("bug"), "A remedy for a major one.")
	repo.Commit("the ledger")
	return repo
}

// TestAnIssueKeyOpensOneLaneWhoseBriefIsTheRecordAndItsRemedy is scope 4:
// `Start` takes an eligible issue's id and opens one lane for it, and the brief
// that lane is handed carries the record, its remedy as the work, and the
// definition of done a detector watched to fail before the fix and pass after.
func TestAnIssueKeyOpensOneLaneWhoseBriefIsTheRecordAndItsRemedy(t *testing.T) {
	repo := issueRepo(t)
	start, err := Start(repo.Root(), eligibleIssue, Options{})
	if err != nil {
		t.Fatalf("an eligible issue starts a run: %v", err)
	}
	st, err := ReadState(repo.Root(), start.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if st.Key != eligibleIssue || st.Intent != "" || st.Spec != "" || len(st.Lanes) != 1 || len(st.Pending) != 0 {
		t.Fatalf("one lane for the issue, no intent and no spec: %+v", st)
	}
	if l := st.Lanes[0]; l.Key != eligibleIssue || l.SpecStep != 1 || !strings.Contains(l.StepTitle, "renderer panics") {
		t.Fatalf("the lane names the issue and its title: %+v", l)
	}
	var eligible bool
	for _, c := range start.Checks {
		if c.Name == CheckEligible && c.OK {
			eligible = true
		}
	}
	if !eligible {
		t.Fatalf("the start's checks carry the drain rule's eligibility row: %+v", start.Checks)
	}
	advanceTo(t, repo, start.RunID, StageImplement)
	st, _ = ReadState(repo.Root(), start.RunID)
	brief, err := os.ReadFile(filepath.Join(repo.Root(), filepath.FromSlash(st.Lanes[0].Brief)))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"the issue: " + eligibleIssue,
		"## The work: " + eligibleIssue + "'s remedy",
		issueRemedy,
		"fails before the fix and passes after",
		"\"issue\": \"" + eligibleIssue + "\"",
		"Working conventions",
		lostConnectionRule,
	} {
		if !strings.Contains(string(brief), want) {
			t.Errorf("the issue brief carries %q:\n%s", want, brief)
		}
	}
	for _, not := range []string{"## The spec:", "## The intent:"} {
		if strings.Contains(string(brief), not) {
			t.Errorf("an issue brief has no %q section", not)
		}
	}
}

// TestAnIssueKeyIsRefusedUnlessItsShapeAndTheRuleAdmitIt: a key that is not an
// issue id by shape is refused before any path is built from it, and an issue
// the repository's rule hands back is refused naming the rule; neither writes.
func TestAnIssueKeyIsRefusedUnlessItsShapeAndTheRuleAdmitIt(t *testing.T) {
	repo := issueRepo(t)
	for _, key := range []string{"iss-../../x", "iss-", "iss-12a", "iss-1/2", "iss-0", "iss-02609292352131344"} {
		_, err := Start(repo.Root(), key, Options{})
		r := mustRefusal(t, err)
		if r.Check != CheckKey {
			t.Errorf("%q is refused at the key check: %+v", key, r)
		}
	}
	_, err := Start(repo.Root(), majorIssue, Options{})
	r := mustRefusal(t, err)
	if r.Check != CheckEligible || !strings.Contains(r.Reason, "severity major") {
		t.Fatalf("a major issue is refused by the drain rule, naming it: %+v", r)
	}
	_, err = Start(repo.Root(), "iss-2609300000009999", Options{})
	if r := mustRefusal(t, err); r.Check != CheckEligible || !strings.Contains(r.Reason, "not an open issue") {
		t.Fatalf("an unknown issue is refused: %+v", r)
	}
	runTierAbsent(t, repo.Root())
}

// issueAwaiting drives a run for the eligible issue to its implement await.
func issueAwaiting(t *testing.T) (*gittest.Repo, string, Lane, string) {
	t.Helper()
	repo := issueRepo(t)
	start, err := Start(repo.Root(), eligibleIssue, Options{})
	if err != nil {
		t.Fatal(err)
	}
	advanceTo(t, repo, start.RunID, StageImplement)
	if _, err := advance(repo.Root(), start.RunID, DefaultStages(), Options{}); err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(repo.Root(), start.RunID)
	if err != nil {
		t.Fatal(err)
	}
	return repo, start.RunID, st.Lanes[0], filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), start.RunID, "lane-1")
}

// TestAnIssueLanesReceiptMustResolveItsIssue: the landing resolves the issue
// through the receipt's `resolves`, so a receipt for an issue lane that does
// not declare its own issue fixed is refused naming it, and one that does is
// verified and carries the resolution to the landing.
func TestAnIssueLanesReceiptMustResolveItsIssue(t *testing.T) {
	repo, runID, l, dir := issueAwaiting(t)
	sha := laneCommit(t, repo, l, "fix.txt")
	rc := goodReceipt(t, runID, l, dir, sha)
	path := writeReceipt(t, dir, rc)
	_, err := Receipt(repo.Root(), runID, path, DefaultStages(), Options{})
	if r := mustRefusal(t, err); !strings.Contains(r.Reason, eligibleIssue) {
		t.Fatalf("a receipt that does not resolve the lane's issue is refused naming it: %+v", r)
	}
	rc.Resolves = []Resolution{{Issue: eligibleIssue, Commit: sha, Note: "guarded", Impact: "fix", Grounds: "pursued: the guard holds; shown wrong if it panics"}}
	writeReceipt(t, dir, rc)
	if _, err := Receipt(repo.Root(), runID, path, DefaultStages(), Options{}); err != nil {
		t.Fatalf("a receipt resolving the lane's issue verifies: %v", err)
	}
	st, _ := ReadState(repo.Root(), runID)
	if got := st.Lanes[0]; got.Stage != StageValidate || len(got.Resolves) != 1 || got.Resolves[0].Issue != eligibleIssue {
		t.Fatalf("the lane moves to its validators with the resolution recorded: %+v", got)
	}
	if audits, err := auditsHere(Context{RepoRoot: repo.Root(), State: st}, st.Lanes[0]); err != nil || audits {
		t.Fatalf("an issue has no criteria, so its lane takes no fidelity audit: %v %v", audits, err)
	}
}

// TestALaneReportHandBackStopsTheLaneAndDiscardsItsWork: a receipt carrying
// `handback` ends the lane with that outcome before the validators: the lane
// is handed back with the kind and reason, its worktree and branch are
// discarded, and the discarded head is recorded so nothing is dropped silently.
func TestALaneReportHandBackStopsTheLaneAndDiscardsItsWork(t *testing.T) {
	repo, runID, l, dir := issueAwaiting(t)
	sha := laneCommit(t, repo, l, "partial.txt")
	rc := goodReceipt(t, runID, l, dir, sha)
	rc.HandBack = &LaneHandBack{Kind: HandBackUserVisible, Reason: "the fix changes what the status board shows"}
	path := writeReceipt(t, dir, rc)
	res, err := Receipt(repo.Root(), runID, path, DefaultStages(), Options{})
	if err != nil {
		t.Fatalf("a hand-back receipt is taken: %v", err)
	}
	if res.HandBack == nil || res.HandBack.Kind != HandBackUserVisible || res.HandBack.Discarded != sha {
		t.Fatalf("the result names the hand-back, its kind and the discarded head: %+v", res.HandBack)
	}
	st, _ := ReadState(repo.Root(), runID)
	got := st.Lanes[0]
	if got.Stage != StageHandedBack || got.HandBack == nil || !strings.Contains(got.HandBack.Reason, "status board") {
		t.Fatalf("the lane stands handed back with the reason: %+v", got)
	}
	if _, err := os.Lstat(l.Worktree); !os.IsNotExist(err) {
		t.Fatalf("the lane's worktree is discarded: %v", err)
	}
	if out := repo.Git("branch", "--list", l.Branch); strings.TrimSpace(out) != "" {
		t.Fatalf("the lane's branch is discarded: %q", out)
	}
	if _, err := advance(repo.Root(), runID, DefaultStages(), Options{}); err == nil {
		t.Fatal("a handed-back lane refuses every later step")
	}

	// A hand-back of a kind the loop does not route is refused, as is one
	// without its reason.
	repo2, runID2, l2, dir2 := issueAwaiting(t)
	sha2 := laneCommit(t, repo2, l2, "p.txt")
	for _, hb := range []LaneHandBack{{Kind: "whim", Reason: "x"}, {Kind: HandBackTrustRule}, {Kind: HandBackDesignFinding, Reason: "x"}} {
		rc2 := goodReceipt(t, runID2, l2, dir2, sha2)
		rc2.HandBack = &hb
		p2 := writeReceipt(t, dir2, rc2)
		if _, err := Receipt(repo2.Root(), runID2, p2, DefaultStages(), Options{}); err == nil {
			t.Errorf("hand-back %+v is refused", hb)
		}
	}
}

// TestAnIssueLandingNamesTheIssueNotASpec: the landing's pull request and
// records commit speak of the issue the lane fixes.
func TestAnIssueLandingNamesTheIssueNotASpec(t *testing.T) {
	st := State{RunID: "run-1", Key: eligibleIssue}
	lane := Lane{ID: "lane-1", Key: eligibleIssue, SpecStep: 1, StepTitle: "The renderer panics",
		Resolves: []Resolution{{Issue: eligibleIssue, Commit: strings.Repeat("a", 40)}}}
	for name, got := range map[string]string{"title": prTitle(st, lane), "body": prBody(st, lane, nil)} {
		if !strings.Contains(got, eligibleIssue) || strings.Contains(got, "step 1 of") {
			t.Errorf("the pull request %s names the issue and no spec step: %q", name, got)
		}
	}
}

// TestAnIssueLaneLandsOnePullRequestThatResolvesItsIssue is criterion 3's
// landing: the issue lane's validators take no audit, its landing resolves the
// issue with the commit the receipt named in the lane's own change (a
// Resolves: trailer, no Delivers:), and it opens one pull request, armed by the
// repository's merge rule, through the forge client.
func TestAnIssueLaneLandsOnePullRequestThatResolvesItsIssue(t *testing.T) {
	repo := issueRepo(t)
	for _, k := range []string{"GIT_AUTHOR_NAME", "GIT_COMMITTER_NAME"} {
		t.Setenv(k, "Pat Example")
	}
	for _, k := range []string{"GIT_AUTHOR_EMAIL", "GIT_COMMITTER_EMAIL"} {
		t.Setenv(k, "pat@example.com")
	}
	repo.Write(".abcd/work/rulesets/main-protection.json", queueRuleset("SQUASH"))
	repo.Commit("the ruleset mirror")
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
	start, err := Start(repo.Root(), eligibleIssue, Options{})
	if err != nil {
		t.Fatal(err)
	}
	f := &landFixture{repo: repo, bare: bare, gh: gh, issue: eligibleIssue, runID: start.RunID, stages: DefaultStages()}

	stepTo(t, repo, f.runID, f.stages, StageImplement)
	if res, err := advance(repo.Root(), f.runID, f.stages, Options{}); err != nil || res.Awaiting == nil {
		t.Fatalf("the implement stage awaits an implementer: %+v %v", res, err)
	}
	l := currentLane(t, repo, f.runID)
	dir := filepath.Join(repo.Root(), filepath.FromSlash(RunRelDir), f.runID, l.ID)
	sha := laneCommit(t, repo, l, "fix.txt")
	rc := goodReceipt(t, f.runID, l, dir, sha)
	// The records commit names the implementer's model (loopLanding), so the
	// receipt reports one in a form the Assisted-by: trailer takes.
	rc.Model = "claude-test-5"
	rc.Resolves = []Resolution{{Issue: eligibleIssue, Commit: sha, Note: "the empty list is guarded", Impact: "fix",
		Grounds: "pursued: the renderer takes an empty list; shown wrong if it still panics"}}
	if _, err := Receipt(repo.Root(), f.runID, writeReceipt(t, dir, rc), f.stages, Options{}); err != nil {
		t.Fatal(err)
	}
	passRound(t, repo, f.runID, f.stages, RoleRuthless, RoleSecurity)
	stepTo(t, repo, f.runID, f.stages, StageLand)
	f.step(t)
	f.step(t)
	l = currentLane(t, repo, f.runID)
	msg := repo.Git("-C", l.Worktree, "log", "-1", "--format=%B", l.HeadSHA)
	if !strings.Contains(msg, "Resolves: "+eligibleIssue) || strings.Contains(msg, "Delivers:") || !strings.Contains(msg, "landing for "+eligibleIssue) {
		t.Fatalf("the records commit resolves the issue and delivers no intent:\n%s", msg)
	}
	if !strings.Contains(msg, "Assisted-by: Claude:claude-test-5") || strings.Contains(msg, "Assisted-by: None") {
		t.Fatalf("the records commit names the implementer's model, never None:\n%s", msg)
	}
	files := repo.Git("-C", l.Worktree, "ls-tree", "-r", "--name-only", l.HeadSHA)
	if !strings.Contains(files, ".abcd/work/issues/resolved/"+eligibleIssue) {
		t.Fatalf("the issue is resolved in the lane's change:\n%s", files)
	}
	preflighted(t, l, l.HeadSHA)
	f.step(t)
	f.step(t)
	f.step(t)
	log := f.ghLog(t)
	if strings.Count(log, "pr create") != 1 || !strings.Contains(log, "fix("+eligibleIssue+")") || !strings.Contains(log, "pr merge 7 --auto --squash") {
		t.Fatalf("one pull request, titled for the issue, armed by the ruleset's method:\n%s", log)
	}
	body, err := os.ReadFile(filepath.Join(gh, "body.md"))
	if err != nil || !strings.Contains(string(body), "fixes "+eligibleIssue) || !strings.Contains(string(body), "Resolves: "+eligibleIssue) {
		t.Fatalf("the body is the issue's: %v\n%s", err, body)
	}
	st, _ := ReadState(repo.Root(), f.runID)
	if outcome, pr, _ := laneOutcome(st); outcome != DrainLanePullRequest || pr != 7 {
		t.Fatalf("the drain reads an armed issue lane as its pull request: %s %d", outcome, pr)
	}
}

// TestAPaddedIssueIdIsNoIssueKey: a leading zero is not an issue id's shape
// anywhere the loop reads one (the key, a drain lane's issue, a state file's
// key, a receipt's resolves), so a padded spelling can never become a run's
// identity or pass a dedupe the canonical spelling would have caught.
func TestAPaddedIssueIdIsNoIssueKey(t *testing.T) {
	for _, key := range []string{"iss-0", "iss-01", "iss-02609292352131344"} {
		if validIssueKey(key) {
			t.Errorf("%q is refused as an issue key", key)
		}
		gaps := resolutionGaps([]Resolution{{Issue: key, Commit: "c"}}, []string{"c"})
		if len(gaps) == 0 || !strings.Contains(gaps[0], "an issue id") {
			t.Errorf("a receipt declaring %q fixed is refused at its id: %v", key, gaps)
		}
	}
	for _, key := range []string{"iss-1", "iss-10", "iss-2609292352131344"} {
		if !validIssueKey(key) {
			t.Errorf("%q is an issue key", key)
		}
	}
}
