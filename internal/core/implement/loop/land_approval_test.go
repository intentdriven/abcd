package loop

import (
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
	assertLeftOpen(t, f, f.landedToArmed(t))
}

// TestAMissingRulesetMirrorLeavesThePullRequestOpen: with no mirror at the
// lane's base and no approval of the head, nothing is armed.
func TestAMissingRulesetMirrorLeavesThePullRequestOpen(t *testing.T) {
	f := newLandFixture(t, "")
	l := f.landedToArmed(t)
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
	l := f.landedToArmed(t)
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

// TestAutoArmsOnlyAHeadAReviewReceiptNames: with facilitator.mode auto on the
// default branch, the loop arms a maintainer's pull request with no approval
// only once a review receipt names its pushed head, and says which commit has
// none until then.
func TestAutoArmsOnlyAHeadAReviewReceiptNames(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	f := newLandFixtureWith(t, map[string]string{
		".abcd/work/rulesets/main-protection.json": unreviewedQueueRuleset("MERGE"),
		".abcd/config.json":                        `{"facilitator":{"mode":"auto"}}` + "\n",
	})
	l := f.landedToArmed(t)
	if strings.Contains(f.ghLog(t), "pr merge") || l.Landing == nil || l.Landing.Armed ||
		!strings.Contains(l.Landing.Merge, "no review receipt names "+l.Landing.Pushed) {
		t.Fatalf("an unreviewed head is left open, naming it: %+v\n%s", l.Landing, f.ghLog(t))
	}
	rcp := filepath.Join(f.repo.Root(), ".abcd", ".work.local", "review-receipts", l.Landing.Pushed)
	if err := os.MkdirAll(filepath.Dir(rcp), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rcp, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	f.step(t)
	if l = currentLane(t, f.repo, f.runID); !l.Landing.Armed || !strings.Contains(f.ghLog(t), "pr merge 7 --auto --merge") {
		t.Fatalf("a review receipt naming the head arms it: %+v\n%s", l.Landing, f.ghLog(t))
	}
}

// TestThisRepositorysOwnMirrorAndConfigArm: this repository's committed
// mirror and .abcd/config.json, with the head approved and a review receipt
// naming it, arm by the queue's method, so the merge gate keeps this
// project's landings queueing in either mode.
func TestThisRepositorysOwnMirrorAndConfigArm(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	top := filepath.Join("..", "..", "..", "..")
	files := map[string]string{}
	for _, rel := range []string{".abcd/work/rulesets/main-protection.json", ".abcd/work/rulesets/main-review.json", ".github/CODEOWNERS", ".abcd/config.json"} {
		b, err := os.ReadFile(filepath.Join(top, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("reading this repository's %s: %v", rel, err)
		}
		files[rel] = string(b)
	}
	f := newLandFixtureWith(t, files)
	f.approve(t)
	f.validated(t)
	f.step(t)
	f.step(t)
	l := currentLane(t, f.repo, f.runID)
	preflighted(t, l, l.HeadSHA)
	rcp := filepath.Join(f.repo.Root(), ".abcd", ".work.local", "review-receipts", l.HeadSHA)
	if err := os.MkdirAll(filepath.Dir(rcp), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rcp, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	f.step(t)
	f.step(t)
	f.step(t)
	l = currentLane(t, f.repo, f.runID)
	if !strings.Contains(f.ghLog(t), "pr merge 7 --auto --merge") || l.Landing == nil || !l.Landing.Armed {
		t.Fatalf("this repository's own mirror and config arm the merge: %+v\n%s", l.Landing, f.ghLog(t))
	}
}
