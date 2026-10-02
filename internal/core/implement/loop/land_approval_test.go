package loop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
// record, when it leaves a pull request open because nothing requires a
// person's approval.
const leftOpenForAPerson = "left open for a person to merge: the ruleset requires no approval"

// assertLeftOpen checks that the lane's pull request was never armed, that its
// landing and the run record both say it was left open for a person, and that
// later steps never arm it.
func assertLeftOpen(t *testing.T, f *landFixture, l Lane) {
	t.Helper()
	if strings.Contains(f.ghLog(t), "pr merge") {
		t.Fatalf("no merge is armed where nothing requires a person's approval:\n%s", f.ghLog(t))
	}
	if l.Landing == nil || l.Landing.Armed || !strings.Contains(l.Landing.Merge, leftOpenForAPerson) {
		t.Fatalf("the landing says the pull request is left open for a person: %+v", l.Landing)
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
		t.Fatalf("the run record says the pull request is left open for a person: %+v", st.Record)
	}
	for range 3 {
		_, _ = advance(f.repo.Root(), f.runID, f.stages, Options{})
	}
	if strings.Contains(f.ghLog(t), "pr merge") {
		t.Fatalf("a later step never arms a pull request left open:\n%s", f.ghLog(t))
	}
}

// TestAMergeQueueWithoutARequiredApprovalLeavesThePullRequestOpen is ruling
// AM1: a merge queue alone is not a person's approval, so the landing leaves
// the pull request open for a person to merge.
func TestAMergeQueueWithoutARequiredApprovalLeavesThePullRequestOpen(t *testing.T) {
	f := newLandFixture(t, unreviewedQueueRuleset("MERGE"))
	assertLeftOpen(t, f, f.landedToArmed(t))
}

// TestAMissingRulesetMirrorLeavesThePullRequestOpen: with no mirror at the
// lane's base nothing is known to require approval, so nothing is armed.
func TestAMissingRulesetMirrorLeavesThePullRequestOpen(t *testing.T) {
	f := newLandFixture(t, "")
	l := f.landedToArmed(t)
	if strings.Contains(f.ghLog(t), "pr merge") || l.Landing == nil || l.Landing.Armed || l.Landing.Merge == "" {
		t.Fatalf("a missing mirror leaves the pull request open: %+v\n%s", l.Landing, f.ghLog(t))
	}
}

// TestTheLandingArmsOnlyWhereTheRulesetRequiresApproval reads the approval
// requirement from every active ruleset on the default branch at the lane's
// base: an approving count of one or more, or a code-owner review with a
// CODEOWNERS file present to name the owners.
func TestTheLandingArmsOnlyWhereTheRulesetRequiresApproval(t *testing.T) {
	const owners = "/commands/ @example\n"
	cases := []struct {
		name  string
		files map[string]string
		armed bool
	}{
		{"an approving count of two in a separate ruleset", map[string]string{
			".abcd/work/rulesets/main-review.json": reviewRuleset("active", "2", false)}, true},
		{"a code-owner review with CODEOWNERS present", map[string]string{
			".abcd/work/rulesets/main-review.json": reviewRuleset("active", "0", true), ".github/CODEOWNERS": owners}, true},
		{"a code-owner review with a root CODEOWNERS", map[string]string{
			".abcd/work/rulesets/main-review.json": reviewRuleset("active", "0", true), "CODEOWNERS": owners}, true},
		{"a code-owner review with no CODEOWNERS file", map[string]string{
			".abcd/work/rulesets/main-review.json": reviewRuleset("active", "0", true)}, false},
		{"a code-owner review whose CODEOWNERS names nobody", map[string]string{
			".abcd/work/rulesets/main-review.json": reviewRuleset("active", "0", true), ".github/CODEOWNERS": "# nobody yet\n\n"}, false},
		{"a code-owner review whose CODEOWNERS gives a bare @", map[string]string{
			".abcd/work/rulesets/main-review.json": reviewRuleset("active", "0", true), ".github/CODEOWNERS": "* @\n"}, false},
		{"a code-owner review whose CODEOWNERS gives patterns only", map[string]string{
			".abcd/work/rulesets/main-review.json": reviewRuleset("active", "0", true), ".github/CODEOWNERS": "/commands/\n*.go # @example\n"}, false},
		{"a comment-only .github/CODEOWNERS shadows a root one naming an owner", map[string]string{
			".abcd/work/rulesets/main-review.json": reviewRuleset("active", "0", true), ".github/CODEOWNERS": "# nobody yet\n", "CODEOWNERS": owners}, false},
		{"a code-owner review whose docs/CODEOWNERS names an e-mail owner", map[string]string{
			".abcd/work/rulesets/main-review.json": reviewRuleset("active", "0", true), "docs/CODEOWNERS": "*.md docs@example.com\n"}, true},
		{"a code-owner review whose CODEOWNERS names a team", map[string]string{
			".abcd/work/rulesets/main-review.json": reviewRuleset("active", "0", true), ".github/CODEOWNERS": "* @example/reviewers\n"}, true},
		{"a pull-request rule requiring nothing", map[string]string{
			".abcd/work/rulesets/main-review.json": reviewRuleset("active", "0", false), ".github/CODEOWNERS": owners}, false},
		{"an approving count in a ruleset only evaluated", map[string]string{
			".abcd/work/rulesets/main-review.json": reviewRuleset("evaluate", "1", true), ".github/CODEOWNERS": owners}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := map[string]string{".abcd/work/rulesets/main-protection.json": unreviewedQueueRuleset("MERGE")}
			for k, v := range tc.files {
				files[k] = v
			}
			f := newLandFixtureWith(t, files)
			l := f.landedToArmed(t)
			if !tc.armed {
				assertLeftOpen(t, f, l)
				return
			}
			if !strings.Contains(f.ghLog(t), "pr merge 7 --auto --merge") || l.Landing == nil || !l.Landing.Armed {
				t.Fatalf("the merge is armed where the ruleset requires approval: %+v\n%s", l.Landing, f.ghLog(t))
			}
		})
	}
}

// TestThisRepositorysOwnMirrorArms: this repository's committed mirror (a
// merge queue, and a code-owner review with approving count 0 beside
// .github/CODEOWNERS) still arms, so ruling AM1 leaves this project unchanged.
func TestThisRepositorysOwnMirrorArms(t *testing.T) {
	top := filepath.Join("..", "..", "..", "..")
	files := map[string]string{}
	for _, rel := range []string{".abcd/work/rulesets/main-protection.json", ".abcd/work/rulesets/main-review.json", ".github/CODEOWNERS"} {
		b, err := os.ReadFile(filepath.Join(top, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("reading this repository's %s: %v", rel, err)
		}
		files[rel] = string(b)
	}
	f := newLandFixtureWith(t, files)
	l := f.landedToArmed(t)
	if !strings.Contains(f.ghLog(t), "pr merge 7 --auto --merge") || l.Landing == nil || !l.Landing.Armed {
		t.Fatalf("this repository's own mirror arms the merge: %+v\n%s", l.Landing, f.ghLog(t))
	}
}
