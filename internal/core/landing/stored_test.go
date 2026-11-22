package landing

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// writeStoredGraft writes lines as r's .git/info/grafts file.
func writeStoredGraft(t *testing.T, r *gittest.Repo, lines ...string) {
	t.Helper()
	info := filepath.Join(r.Root(), ".git", "info")
	if err := os.MkdirAll(info, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(info, "grafts"), []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestBranchItemsReadsTheBranchAsStoredUnderAGraft: a graft that parents the
// branch's tip straight on the base hides the commit before it, and its
// trailer, from a log that honours grafts. land reads the branch as stored,
// which is what its push sends, so the item that commit names is still
// announced.
func TestBranchItemsReadsTheBranchAsStoredUnderAGraft(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("README.md", "a project\n")
	r.Commit("start")
	base := r.Git("rev-parse", "HEAD")
	r.Write("fix.txt", "the fix\n")
	r.Commit("fix: the widget\n\nRefs: iss-5")
	r.Write("docs.txt", "a page\n")
	r.Commit("docs: a page")
	tip := r.Git("rev-parse", "HEAD")
	writeStoredGraft(t, r, tip+" "+base)
	if got := r.Git("-c", "advice.graftFileDeprecated=false", "log", "--format=%s", base+".."+tip); got != "docs: a page" {
		t.Fatalf("the graft does not stand in: the branch reads as %q", got)
	}

	items, err := branchItems(r.Root(), base, tip)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(items, "iss-5") {
		t.Fatalf("items %q, want iss-5 from the stored commit", items)
	}
}

// TestReadMergeRuleReadsTheMirrorAsStoredUnderAReplaceRef: a refs/replace
// entry that stands a commit without the ruleset mirror in for the base
// changes nothing land reads. The merge rule comes from the mirror the base
// holds as stored.
func TestReadMergeRuleReadsTheMirrorAsStoredUnderAReplaceRef(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("README.md", "a project\n")
	r.Commit("start")
	start := r.Git("rev-parse", "HEAD")
	r.Write(RulesetsRelDir+"/main-protection.json", strings.Replace(queueRuleset, `"MERGE"`, `"SQUASH"`, 1))
	r.Commit("mirror the rulesets")
	base := r.Git("rev-parse", "HEAD")
	standIn := r.Git("commit-tree", start+"^{tree}", "-p", start, "-m", "mirror the rulesets")
	r.Git("replace", base, standIn)
	if got := r.Git("ls-tree", "--name-only", base, "--", RulesetsRelDir+"/"); got != "" {
		t.Fatalf("the replace ref does not stand in: the base lists %q", got)
	}

	rule, err := ReadMergeRule(r.Root(), base, "main", "the base")
	if err != nil {
		t.Fatal(err)
	}
	if rule.Method != "SQUASH" {
		t.Fatalf("merge rule %+v, want the stored mirror's SQUASH", rule)
	}
}

// TestComposeBodyReadsTheBranchAsStoredUnderAReplaceRef: a refs/replace entry
// that stands a commit with another subject and no trailers in for the
// branch's tip changes nothing in the composed body. The body is composed
// from the commits as stored, which is what the push sends, so the subject
// and the Refs: line the tip carries are in it.
func TestComposeBodyReadsTheBranchAsStoredUnderAReplaceRef(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("README.md", "a project\n")
	r.Commit("start")
	base := r.Git("rev-parse", "HEAD")
	r.Write("fix.txt", "the fix\n")
	r.Commit("fix: the widget\n\nRefs: iss-5")
	tip := r.Git("rev-parse", "HEAD")
	standIn := r.Git("commit-tree", tip+"^{tree}", "-p", base, "-m", "tidy the notes")
	r.Git("replace", tip, standIn)
	if got := r.Git("log", "-1", "--format=%s", tip); got != "tidy the notes" {
		t.Fatalf("the replace ref does not stand in: %s reads as %q", tip, got)
	}

	c, err := composeBody(r.Root(), base, tip)
	if err != nil {
		t.Fatal(err)
	}
	if c.title != "fix: the widget" || !strings.Contains(c.body, "- fix: the widget\n") || !strings.Contains(c.body, "\nRefs: iss-5\n") {
		t.Fatalf("composed %q / %q, want the stored subject and its Refs: line", c.title, c.body)
	}
	if strings.Contains(c.body, "tidy the notes") {
		t.Fatalf("composed body %q carries the stand-in's subject", c.body)
	}
}
