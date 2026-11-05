package loop

import (
	"fmt"
	"github.com/intentdriven/abcd/internal/gitutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// land_approval_test.go is the build loop's half of the merge gate
// (spc-2610161609350616 step 4), which replaced ruling AM1: the loop asks
// landing.DecideMerge exactly as `abcd land` does, so a ruleset's approval
// rule no longer decides the arming; the forge's approval of the head, the
// facilitator mode and, in auto, a review receipt do.

// reviewRuleset is a second ruleset on the default branch carrying one
// pull-request rule with the given approving count and code-owner flag, under
// the enforcement given.
func reviewRuleset(enforcement, count string, codeOwner bool) string {
	owner := "false"
	if codeOwner {
		owner = "true"
	}
	return `{"bypass_actors":[],"conditions":{"ref_name":{"exclude":[],"include":["~DEFAULT_BRANCH"]}},` +
		`"enforcement":"` + enforcement + `","name":"main review","rules":[{"parameters":{"require_code_owner_review":` + owner +
		`,"required_approving_review_count":` + count + `},"type":"pull_request"}],"target":"branch"}` + "\n"
}

// leftOpenForAPerson is what the landing says, in its state and its run
// record, when it leaves a maintainer's pull request open in human mode
// because no approval of its head counts.
const leftOpenForAPerson = "left open: facilitator.mode is human"

// assertLeftOpen checks that the lane's pull request was never armed, that its
// landing and the run record both say it waits for the technical
// facilitator's approval, and that later steps, with still no approval, never
// arm it.
func assertLeftOpen(t *testing.T, f *landFixture, l Lane) {
	t.Helper()
	if strings.Contains(f.ghLog(t), "pr merge") {
		t.Fatalf("no merge is armed while no approval of the head counts:\n%s", f.ghLog(t))
	}
	if l.Landing == nil || l.Landing.Armed || !strings.Contains(l.Landing.Merge, leftOpenForAPerson) ||
		!strings.Contains(l.Landing.Merge, "technical facilitator's approval") {
		t.Fatalf("the landing says the pull request waits for the facilitator's approval: %+v", l.Landing)
	}
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	said := false
	for _, e := range st.Record {
		if e.Lane == l.ID && strings.Contains(e.Note, leftOpenForAPerson) {
			said = true
		}
	}
	if !said {
		t.Fatalf("the run record says the pull request is left open: %+v", st.Record)
	}
	for range 3 {
		_, _ = advance(f.repo.Root(), f.runID, f.stages, Options{})
	}
	if strings.Contains(f.ghLog(t), "pr merge") {
		t.Fatalf("a later step never arms a pull request no approval counts for:\n%s", f.ghLog(t))
	}
}

// approve has the stub forge hold a maintainer's approval of every head the
// remote holds, as a person gives it.
func (f *landFixture) approve(t *testing.T) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(f.gh, "approved"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestHumanLeavesAPullRequestWithNoApprovalOpen: with no facilitator.mode
// anywhere (human), a merge queue with no approval of the head leaves the pull
// request open, saying it waits for the technical facilitator's approval.
func TestHumanLeavesAPullRequestWithNoApprovalOpen(t *testing.T) {
	f := newLandFixture(t, unreviewedQueueRuleset("MERGE"))
	assertLeftOpen(t, f, f.landedToArmedOnce(t))
}

// TestAMissingRulesetMirrorLeavesThePullRequestOpen: with no mirror at the
// lane's base and no approval of the head, nothing is armed.
func TestAMissingRulesetMirrorLeavesThePullRequestOpen(t *testing.T) {
	f := newLandFixture(t, "")
	l := f.landedToArmedOnce(t)
	if strings.Contains(f.ghLog(t), "pr merge") || l.Landing == nil || l.Landing.Armed || l.Landing.Merge == "" {
		t.Fatalf("a missing mirror leaves the pull request open: %+v\n%s", l.Landing, f.ghLog(t))
	}
}

// TestTheRulesetsApprovalRuleNoLongerDecidesTheArming keeps AM1's cases and
// turns them over: whatever pull-request rule the mirror holds, a counted
// approval of the head arms by the queue's method, and none leaves the pull
// request open.
func TestTheRulesetsApprovalRuleNoLongerDecidesTheArming(t *testing.T) {
	const owners = "/commands/ @example\n"
	cases := []struct {
		name     string
		files    map[string]string
		approved bool
	}{
		{"an approving count of two, no approval given", map[string]string{
			".abcd/work/rulesets/main-review.json": reviewRuleset("active", "2", false)}, false},
		{"an approving count of two, approved", map[string]string{
			".abcd/work/rulesets/main-review.json": reviewRuleset("active", "2", false)}, true},
		{"a code-owner review with CODEOWNERS present, no approval given", map[string]string{
			".abcd/work/rulesets/main-review.json": reviewRuleset("active", "0", true), ".github/CODEOWNERS": owners}, false},
		{"a code-owner review with no CODEOWNERS file, approved", map[string]string{
			".abcd/work/rulesets/main-review.json": reviewRuleset("active", "0", true)}, true},
		{"a pull-request rule requiring nothing, approved", map[string]string{
			".abcd/work/rulesets/main-review.json": reviewRuleset("active", "0", false), ".github/CODEOWNERS": owners}, true},
		{"an approving count in a ruleset only evaluated, no approval given", map[string]string{
			".abcd/work/rulesets/main-review.json": reviewRuleset("evaluate", "1", true), ".github/CODEOWNERS": owners}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := map[string]string{".abcd/work/rulesets/main-protection.json": unreviewedQueueRuleset("MERGE")}
			for k, v := range tc.files {
				files[k] = v
			}
			f := newLandFixtureWith(t, files)
			// The forge's approval is the case's, whatever the rule asks.
			_ = os.Remove(filepath.Join(f.gh, "approved"))
			if tc.approved {
				f.approve(t)
			}
			l := f.landedToArmed(t)
			if !tc.approved {
				assertLeftOpen(t, f, l)
				return
			}
			if !strings.Contains(f.ghLog(t), "pr merge 7 --auto --merge") || l.Landing == nil || !l.Landing.Armed {
				t.Fatalf("the merge is armed where the head is approved: %+v\n%s", l.Landing, f.ghLog(t))
			}
		})
	}
}

// TestALaterApprovalArmsAParkedLane: a lane left open for the facilitator's
// approval is asked about again on each step, and arms once the approval of
// its head is given.
func TestALaterApprovalArmsAParkedLane(t *testing.T) {
	f := newLandFixture(t, unreviewedQueueRuleset("MERGE"))
	l := f.landedToArmedOnce(t)
	if l.Landing == nil || l.Landing.Armed {
		t.Fatalf("the lane is left open with no approval: %+v", l.Landing)
	}
	f.approve(t)
	f.step(t)
	l = currentLane(t, f.repo, f.runID)
	if !strings.Contains(f.ghLog(t), "pr merge 7 --auto --merge --match-head-commit="+l.Landing.Pushed) || !l.Landing.Armed {
		t.Fatalf("the approval given later arms the parked lane: %+v\n%s", l.Landing, f.ghLog(t))
	}
}

// autoFixture is a lane in a repository whose committed mode is auto.
func autoFixture(t *testing.T) *landFixture {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	return newLandFixtureWith(t, map[string]string{
		".abcd/work/rulesets/main-protection.json": unreviewedQueueRuleset("MERGE"),
		".abcd/config.json":                        `{"facilitator":{"mode":"auto"}}` + "\n",
		relateLaneIssue:                            "",
	})
}

// prBodyOf is the pull request's body as the loop composed it.
func (f *landFixture) prBodyOf(t *testing.T, l Lane) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(f.repo.Root(), filepath.FromSlash(l.Landing.Body)))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// recordsRound takes the step that sends the records commit to its round, and hands
// back each reviewer's return of it, with verdict words by role.
func (f *landFixture) recordsRound(t *testing.T, ruthless, security string) {
	t.Helper()
	f.step(t)
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range []struct{ role, verdict string }{{RoleRuthless, ruthless}, {RoleSecurity, security}} {
		for i := range roundReviewers(st, r.role) {
			handBackOne(t, f.repo, f.runID, f.stages, r.role, fmt.Sprintf("### Reader\n\nReader %d.\n\n%s", i, reviewerReturn(r.verdict)))
		}
	}
}

// pushToArm pushes the lane at its preflight receipt, opens its pull request and asks
// the gate.
func (f *landFixture) pushToArm(t *testing.T) Lane {
	t.Helper()
	l := currentLane(t, f.repo, f.runID)
	preflighted(t, l, l.HeadSHA)
	f.step(t)
	f.step(t)
	f.step(t)
	return currentLane(t, f.repo, f.runID)
}

// TestAPairReadsTheRecordsCommitBeforeAnythingIsPushed: with facilitator.mode auto a
// lane's records commit goes to a round of its own, scoped to that commit, nothing
// pushed until it has read it; the pair that read the pushed head is its receipt, and
// the lane arms itself.
func TestAPairReadsTheRecordsCommitBeforeAnythingIsPushed(t *testing.T) {
	f := autoFixture(t)
	f.validated(t)
	f.step(t)
	f.step(t)
	records := currentLane(t, f.repo, f.runID)
	judged := records.Validation[len(records.Validation)-1].HeadSHA
	if records.HeadSHA == judged || records.Landing == nil || records.Landing.Records != records.HeadSHA {
		t.Fatalf("the landing made a records commit: %+v", records.Landing)
	}
	f.step(t)
	res, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
	if err != nil || res.Awaiting == nil || res.Awaiting.Role != RoleRuthless {
		t.Fatalf("a reviewer is handed the records commit: %+v %v", res, err)
	}
	if strings.Contains(f.ghLog(t), "pr create") {
		t.Fatalf("nothing is opened before the round:\n%s", f.ghLog(t))
	}
	raw, err := os.ReadFile(filepath.Join(f.repo.Root(), filepath.FromSlash(res.Awaiting.Brief)))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{judged + ".." + records.HeadSHA, "couple of minutes", "resolution's note, impact and grounds", "records and link rewrites", "`Reviewed head: " + records.HeadSHA + "`"} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("the brief says %q:\n%s", want, raw)
		}
	}
	if err := os.WriteFile(filepath.Join(f.repo.Root(), filepath.FromSlash(res.Awaiting.Receipt)),
		[]byte("Reviewed head: "+records.HeadSHA+"\n\nReader 0.\n\n"+reviewerReturn("SHIP")), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Receipt(f.repo.Root(), f.runID, res.Awaiting.Receipt, f.stages, Options{}); err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i < len(roundReviewers(st, RoleRuthless)); i++ {
		handBackOne(t, f.repo, f.runID, f.stages, RoleRuthless, fmt.Sprintf("### Reader\n\nReader %d.\n\n%s", i, reviewerReturn("SHIP")))
	}
	for i := range roundReviewers(st, RoleSecurity) {
		handBackOne(t, f.repo, f.runID, f.stages, RoleSecurity, fmt.Sprintf("### Reader\n\nOther %d.\n\n%s", i, reviewerReturn("APPROVE")))
	}
	f.step(t)
	l := f.pushToArm(t)
	if !strings.Contains(f.ghLog(t), "pr merge 7 --auto --merge") || l.Landing == nil || !l.Landing.Armed {
		t.Fatalf("a lane whose pair read its head arms itself: %+v\n%s", l.Landing, f.ghLog(t))
	}
	if l.Landing.Pushed != records.HeadSHA {
		t.Fatalf("the head the pair read is the one pushed: %+v", l.Landing)
	}
	for _, v := range l.Validation[len(l.Validation)-1].Validators {
		if v.Role == RoleAuditor {
			t.Fatalf("a records round takes no audit: %+v", v)
		}
	}
	if _, err := os.Stat(filepath.Join(f.repo.Root(), ".abcd", ".work.local", "review-receipts", l.Landing.Pushed)); err != nil {
		t.Fatalf("no receipt names the pushed head: %v", err)
	}
	if _, err := os.Stat(filepath.Join(f.repo.Root(), ".abcd", ".work.local", "review-receipts", judged)); err == nil {
		t.Fatal("nothing signs the head no one pushed")
	}
}

// TestARecordsRoundThatFailsParksNamingTheFinding: no fix round follows: the lane goes
// on to its pull request, left open naming the finding, and a pair a person records
// later still arms it.
func TestARecordsRoundThatFailsParksNamingTheFinding(t *testing.T) {
	f := autoFixture(t)
	f.validated(t)
	f.step(t)
	f.step(t)
	f.recordsRound(t, "FIX FIRST", "APPROVE")
	f.step(t)
	before := len(currentLane(t, f.repo, f.runID).Validation)
	l := f.pushToArm(t)
	if strings.Contains(f.ghLog(t), "pr merge") || l.Landing == nil || l.Landing.Armed ||
		!strings.Contains(l.Landing.Merge, "FIX FIRST") {
		t.Fatalf("the lane parks naming the finding: %+v\n%s", l.Landing, f.ghLog(t))
	}
	if len(l.Validation) != before || len(l.Awaits) != 0 {
		t.Fatalf("no fix round follows: %+v %+v", l.Validation, l.Awaits)
	}
	if body := f.prBodyOf(t, l); !strings.Contains(body, "did not pass") || !strings.Contains(body, "parked at this pull request for a person") {
		t.Fatalf("the public body states the round's outcome:\n%s", body)
	}
	rcp := filepath.Join(f.repo.Root(), ".abcd", ".work.local", "review-receipts", l.Landing.Pushed)
	if err := os.MkdirAll(filepath.Dir(rcp), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rcp, reviewPairReceipt(l.Landing.Pushed), 0o600); err != nil {
		t.Fatal(err)
	}
	f.step(t)
	if l = currentLane(t, f.repo, f.runID); !l.Landing.Armed {
		t.Fatalf("a person's pair arms it: %+v\n%s", l.Landing, f.ghLog(t))
	}
}

// TestAResyncDoesNotEraseAFailedRecordsRound: a lane parked over its records is refused
// for a person, naming the finding, rather than merged to a head a fresh round passes.
func TestAResyncDoesNotEraseAFailedRecordsRound(t *testing.T) {
	root := t.TempDir()
	const rel = "return.md"
	if err := os.WriteFile(filepath.Join(root, rel), []byte("### Analysis\n\n- not this one\n\n### Findings\n\n- the impact is wrong\n\n### Verdict\n\nBLOCK\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	head := strings.Repeat("a", 40)
	lane := Lane{ID: "lane-1", PR: 7, Landing: &Landing{Records: head, Pushed: head},
		Validation: []ValidationRound{{Round: 2, HeadSHA: head, Validators: []ValidatorRun{{Role: RoleSecurity, Verdict: "BLOCK", Return: rel}}}}}
	_, err := resyncLane(Context{RepoRoot: root}, &lane, "main", "origin/main", mergeability{})
	r, ok := AsRefusal(err)
	if !ok || r.Contention || !strings.Contains(r.Reason, "the impact is wrong") || strings.Contains(r.Reason, "not this one") {
		t.Fatalf("the sync is refused naming the finding: %v", err)
	}
}

// TestFirstFindingIsTheHeadingsFirstItem: the word elsewhere is no heading.
func TestFirstFindingIsTheHeadingsFirstItem(t *testing.T) {
	for text, want := range map[string]string{
		"### Analysis\n\nFindings below.\n- refuted\n\n### Findings\n\n- real\n": ": real",
		"### Analysis\n\nFindings below.\n- refuted\n":                           "",
		"## Findings\n\nNone.\n\n### Verdict\n\n- SHIP\n":                        "",
	} {
		if got := firstFinding(text); got != want {
			t.Errorf("%q: got %q, want %q", text, got, want)
		}
	}
}

// TestAStaleRoundDoesNotServe: a round that read another head is no pair for the
// pushed one, whatever it passed, so the lane stays open saying why.
func TestAStaleRoundDoesNotServe(t *testing.T) {
	f := autoFixture(t)
	f.validated(t)
	f.step(t)
	f.step(t)
	f.recordsRound(t, "SHIP", "APPROVE")
	f.step(t)
	l := currentLane(t, f.repo, f.runID)
	preflighted(t, l, l.HeadSHA)
	f.step(t)
	f.step(t)
	if err := mutate(f.repo.Root(), f.runID, func(_ *os.Root, st *State) (bool, error) {
		lane := &st.Lanes[st.current()]
		lane.Validation[len(lane.Validation)-1].HeadSHA = lane.BaseSHA
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	f.step(t)
	l = currentLane(t, f.repo, f.runID)
	if strings.Contains(f.ghLog(t), "pr merge") || l.Landing == nil || l.Landing.Armed ||
		!strings.Contains(l.Landing.Merge, "the loop did not record its own review") {
		t.Fatalf("a stale round arms nothing: %+v\n%s", l.Landing, f.ghLog(t))
	}
}

// TestAnExtraResolutionParksSayingWhich: a receipt that resolves an issue beyond the
// lane's own and its intent's related ones goes to a person before any round.
func TestAnExtraResolutionParksSayingWhich(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	f := newLandFixtureWith(t, map[string]string{
		".abcd/work/rulesets/main-protection.json": unreviewedQueueRuleset("MERGE"),
		".abcd/config.json":                        `{"facilitator":{"mode":"auto"}}` + "\n",
	})
	f.validated(t)
	before := len(currentLane(t, f.repo, f.runID).Validation)
	f.step(t)
	f.step(t)
	l := f.pushToArm(t)
	if strings.Contains(f.ghLog(t), "pr merge") || l.Landing == nil || l.Landing.Armed || !strings.Contains(l.Landing.Merge, f.issue) {
		t.Fatalf("the lane parks naming the issue: %+v\n%s", l.Landing, f.ghLog(t))
	}
	if len(l.Validation) != before {
		t.Fatalf("no round read it: %+v", l.Validation)
	}
	if body := f.prBodyOf(t, l); !strings.Contains(body, "parked at this pull request for a person") {
		t.Fatalf("the public body says it is parked:\n%s", body)
	}
}

// TestACommitHoldingMoreThanRecordsParks: what is no markdown record goes to a person.
func TestACommitHoldingMoreThanRecordsParks(t *testing.T) {
	f := autoFixture(t)
	f.validated(t)
	f.step(t)
	l := currentLane(t, f.repo, f.runID)
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(l.Worktree, "code.txt"), []byte("x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := gitutil.LocalGit(l.Worktree, "add", "code.txt"); err != nil {
		t.Fatal(err)
	}
	if out, err := gitutil.LocalGit(l.Worktree, "commit", "-q", "-m", landSubject(st, l)); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	f.step(t)
	l = f.pushToArm(t)
	if strings.Contains(f.ghLog(t), "pr merge") || l.Landing == nil || l.Landing.Armed || !strings.Contains(l.Landing.Merge, "code.txt") {
		t.Fatalf("the lane parks naming the path: %+v\n%s", l.Landing, f.ghLog(t))
	}
}

// TestACommitFoundOnTheJudgedHeadIsReadLikeAnyOther: a commit under the landing's
// subject that the landing did not make still goes to the round.
func TestACommitFoundOnTheJudgedHeadIsReadLikeAnyOther(t *testing.T) {
	f := autoFixture(t)
	f.validated(t)
	f.step(t)
	l := currentLane(t, f.repo, f.runID)
	st, err := ReadState(f.repo.Root(), f.runID)
	if err != nil {
		t.Fatal(err)
	}
	if out, err := gitutil.LocalGit(l.Worktree, "commit", "-q", "--allow-empty", "-m", landSubject(st, l)); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	f.step(t)
	f.step(t)
	res, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
	if err != nil || res.Awaiting == nil || res.Awaiting.Role != RoleRuthless {
		t.Fatalf("a reviewer reads it: %+v %v", res, err)
	}
	if strings.Contains(f.ghLog(t), "pr create") {
		t.Fatalf("it is not pushed first:\n%s", f.ghLog(t))
	}
}

// TestALandingOverARefusedHookKeepsItsOwnEdits: a markdown change, as a relink leaves,
// is not called stray.
func TestALandingOverARefusedHookKeepsItsOwnEdits(t *testing.T) {
	f := autoFixture(t)
	f.validated(t)
	f.step(t)
	l := currentLane(t, f.repo, f.runID)
	if err := os.WriteFile(filepath.Join(l.Worktree, "relinked.md"), []byte("x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	f.step(t)
	if l := currentLane(t, f.repo, f.runID); !l.Landing.RecordsDone {
		t.Fatalf("the records commit is made: %+v", l.Landing)
	}
}

// TestAStrayChangeStopsTheRecordsCommit: what the landing did not write never joins
// its commit.
func TestAStrayChangeStopsTheRecordsCommit(t *testing.T) {
	f := autoFixture(t)
	f.validated(t)
	f.step(t)
	l := currentLane(t, f.repo, f.runID)
	if err := os.WriteFile(filepath.Join(l.Worktree, "stray.txt"), []byte("x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := advance(f.repo.Root(), f.runID, f.stages, Options{}); err == nil || !strings.Contains(err.Error(), "stray.txt") {
		t.Fatalf("the step is refused naming it: %v", err)
	}
	if l := currentLane(t, f.repo, f.runID); l.Landing.RecordsDone {
		t.Fatalf("no records commit: %+v", l.Landing)
	}
}

// TestAnUnreadGateAsksAgain: a read that failed says nothing of the lane, so it is not
// left open; a later turn asks again, and the answer then decides.
func TestAnUnreadGateAsksAgain(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	f := newLandFixture(t, unreviewedQueueRuleset("MERGE"))
	f.validated(t)
	f.step(t)
	f.step(t)
	l := currentLane(t, f.repo, f.runID)
	preflighted(t, l, l.HeadSHA)
	f.step(t)
	f.step(t)
	role := filepath.Join(f.gh, "role-author-a")
	if err := os.WriteFile(role, []byte("not json\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := advance(f.repo.Root(), f.runID, f.stages, Options{})
	if r, ok := AsRefusal(err); !ok || !r.Contention {
		t.Fatalf("the step waits on the read: %v", err)
	}
	if l := currentLane(t, f.repo, f.runID); l.Landing.Merge != "" {
		t.Fatalf("a read that failed left the lane open: %+v", l.Landing)
	}
	if st, err := ReadState(f.repo.Root(), f.runID); err != nil || st.Lanes[0].Refused != nil {
		t.Fatalf("the wait does not count toward stuck: %+v %v", st.Lanes[0].Refused, err)
	}
	if err := os.Remove(role); err != nil {
		t.Fatal(err)
	}
	f.approve(t)
	f.step(t)
	if l := currentLane(t, f.repo, f.runID); !l.Landing.Armed {
		t.Fatalf("the later turn asks again and arms: %+v", l.Landing)
	}
}

// TestHumanModeRecordsNoReviewOfTheLoops: only auto takes the records round and the
// loop's own pair; in human the lane waits for the facilitator's approval, no round
// reads the records commit, and no receipt appears.
func TestHumanModeRecordsNoReviewOfTheLoops(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	f := newLandFixtureWith(t, map[string]string{
		".abcd/work/rulesets/main-protection.json": unreviewedQueueRuleset("MERGE"),
		".abcd/config.json":                        `{"facilitator":{"mode":"human"}}` + "\n",
	})
	l := f.landedToArmed(t)
	if strings.Contains(f.ghLog(t), "pr merge") || l.Landing == nil || l.Landing.Armed {
		t.Fatalf("human mode waits for an approval: %+v\n%s", l.Landing, f.ghLog(t))
	}
	if entries, err := os.ReadDir(filepath.Join(f.repo.Root(), ".abcd", ".work.local", "review-receipts")); err == nil && len(entries) > 0 {
		t.Fatalf("human mode wrote a receipt: %v", entries)
	}
}

// TestThisRepositorysOwnMirrorAndConfigArm: this repository's committed
// mirror and .abcd/config.json, with the head approved and a pair that read it, arm by the queue's method, so the merge gate keeps this
// project's landings queueing in either mode. It reads every CODEOWNERS path
// the forge reads, so one restored shows up here.
func TestThisRepositorysOwnMirrorAndConfigArm(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	top := filepath.Join("..", "..", "..", "..")
	files := map[string]string{}
	for _, rel := range []string{".abcd/work/rulesets/main-protection.json", ".abcd/work/rulesets/main-review.json", ".abcd/config.json"} {
		b, err := os.ReadFile(filepath.Join(top, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("reading this repository's %s: %v", rel, err)
		}
		files[rel] = string(b)
	}
	for _, rel := range []string{".github/CODEOWNERS", "CODEOWNERS", "docs/CODEOWNERS"} {
		b, err := os.ReadFile(filepath.Join(top, filepath.FromSlash(rel)))
		if err == nil {
			files[rel] = string(b)
		} else if !os.IsNotExist(err) {
			t.Fatalf("reading this repository's %s: %v", rel, err)
		}
	}
	files[relateLaneIssue] = ""
	f := newLandFixtureWith(t, files)
	f.approve(t)
	f.validated(t)
	f.step(t)
	f.step(t)
	f.recordsRound(t, "SHIP", "APPROVE")
	f.step(t)
	l := f.pushToArm(t)
	if !strings.Contains(f.ghLog(t), "pr merge 7 --auto --merge") || l.Landing == nil || !l.Landing.Armed {
		t.Fatalf("this repository's own mirror and config arm the merge: %+v\n%s", l.Landing, f.ghLog(t))
	}
}

// reviewPairReceipt is a review receipt for sha holding a passing pair, as
// abcd land --record-review writes one.
func reviewPairReceipt(sha string) []byte {
	return []byte(`{"commit":"` + sha + `","reviews":[{"name":"a","verdict":"SHIP","sha256":"x"},{"name":"b","verdict":"APPROVE","sha256":"y"}]}` + "\n")
}
