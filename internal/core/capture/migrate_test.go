package capture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/intent"
)

// issueRecordText is a minimal well-formed issue record with extra frontmatter
// lines spliced in before the closing delimiter.
func issueRecordText(id, slug, extra string) string {
	return "---\nschema_version: 1\nid: \"" + id + "\"\nslug: \"" + slug + "\"\nseverity: \"minor\"\n" +
		"category: \"observation\"\nsource: \"user-observation\"\nfound_during: \"t\"\n" + extra + "---\n\nThe observation.\n"
}

func writeTree(t *testing.T, root, rel, body string) string {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return abs
}

func readTree(t *testing.T, root, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// migrateFixture lays a tree written by an older abcd: the retired promote
// back-links in every shape the ledger and the intent store carry them.
//
//   - iss-1 → itd-1, both halves present, iss-1 also loosely related to itd-2;
//   - itd-3 promoted from iss-3, only the intent half present;
//   - iss-4 → itd-4, only the ledger half present (a link-mode promote of a
//     hand-filed draft, which wrote the ledger half alone);
//   - rdi-5 → itd-5, both halves present, in the readings store;
//   - iss-6 carries neither and must be left byte-identical.
func migrateFixture(t *testing.T) (repo, ir string, rels []string) {
	t.Helper()
	repo, ir = ledger(t)
	const led = ".abcd/work/issues/"
	const itd = ".abcd/development/intents/"
	files := map[string]string{
		led + "open/iss-1-one.md":      issueRecordText("iss-1", "one", "related_intents: [itd-2]\npromoted_to: itd-1\n"),
		led + "open/iss-3-three.md":    issueRecordText("iss-3", "three", ""),
		led + "resolved/iss-4-four.md": issueRecordText("iss-4", "four", "promoted_to: itd-4\nresolution: \"folded into itd-4\"\n"),
		led + "open/iss-6-six.md":      issueRecordText("iss-6", "six", "related_intents: [itd-2]\n"),
		led + "readings/rdg-9/rdi-5.md": "---\nschema_version: 1\nid: rdi-5\nrun: rdg-9\npattern: a pattern\n" +
			"promoted_to: itd-5\n---\n\nbody\n",
		itd + "drafts/itd-1-one.md":    "---\nid: itd-1\nslug: one\nspec_id: null\nkind: null\nseverity: minor\npromoted_from: iss-1\norigin: extracted-from-record\n---\n\n# One\n",
		itd + "drafts/itd-2-two.md":    "---\nid: itd-2\nslug: two\nspec_id: null\nkind: null\n---\n\n# Two\n",
		itd + "planned/itd-3-three.md": "---\nid: itd-3\nslug: three\nspec_id: null\nkind: standalone\npromoted_from: iss-3\n---\n\n# Three\n",
		itd + "planned/itd-4-four.md":  "---\nid: itd-4\nslug: four\nspec_id: null\nkind: standalone\n---\n\n# Four\n",
		itd + "drafts/itd-5-five.md":   "---\nid: itd-5\nslug: five\nspec_id: null\nkind: null\npromoted_from: rdi-5\n---\n\n# Five\n",
	}
	for rel, body := range files {
		writeTree(t, repo, rel, body)
		rels = append(rels, rel)
	}
	return repo, ir, rels
}

// TestMigrateReportsWithoutWriting — the migration reports by default: the
// records are the only copy, so writing is the explicit ask.
func TestMigrateReportsWithoutWriting(t *testing.T) {
	repo, ir, rels := migrateFixture(t)
	before := map[string]string{}
	for _, rel := range rels {
		before[rel] = readTree(t, repo, rel)
	}
	res, err := Migrate(MigrateRequest{RepoRoot: repo, IssuesRoot: ir})
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if res.Applied {
		t.Fatal("a report must not claim it applied anything")
	}
	// iss-1, iss-3, iss-4, rdi-5 and itd-1, itd-3, itd-4, itd-5 change; iss-6 and itd-2 do not.
	if len(res.Changes) != 8 {
		t.Fatalf("changes = %d, want 8: %+v", len(res.Changes), res.Changes)
	}
	for _, rel := range rels {
		if got := readTree(t, repo, rel); got != before[rel] {
			t.Fatalf("a report rewrote %s:\n%s", rel, got)
		}
	}
}

// TestMigrateRewritesEveryRetiredBackLinkIntoTheTwoSidedJoin — itd-4 AC3: the
// retired `promoted_to` / `promoted_from` become `related_intents` /
// `related_issues`, and because "promoted" is now the PAIR, a join an older
// abcd wrote from one end only is completed from the other. A loose relation
// is kept, a record carrying neither retired key is untouched, the migrated
// ledger reads with nothing skipped, and a second run changes nothing.
func TestMigrateRewritesEveryRetiredBackLinkIntoTheTwoSidedJoin(t *testing.T) {
	repo, ir, _ := migrateFixture(t)
	untouched := readTree(t, repo, ".abcd/work/issues/open/iss-6-six.md")

	res, err := Migrate(MigrateRequest{RepoRoot: repo, IssuesRoot: ir, Apply: true})
	if err != nil {
		t.Fatalf("Migrate --apply: %v", err)
	}
	if !res.Applied {
		t.Fatal("an applied migration must say so")
	}

	for rel, want := range map[string]string{
		".abcd/work/issues/open/iss-1-one.md":              "\nrelated_intents: [itd-2, itd-1]\n",
		".abcd/work/issues/open/iss-3-three.md":            "\nrelated_intents: [itd-3]\n",
		".abcd/work/issues/resolved/iss-4-four.md":         "\nrelated_intents: [itd-4]\n",
		".abcd/work/issues/readings/rdg-9/rdi-5.md":        "\nrelated_intents: [itd-5]\n",
		".abcd/development/intents/drafts/itd-1-one.md":    "\nseverity: minor\nrelated_issues: [iss-1]\norigin:",
		".abcd/development/intents/planned/itd-3-three.md": "\nrelated_issues: [iss-3]\n",
		".abcd/development/intents/planned/itd-4-four.md":  "\nrelated_issues: [iss-4]\n",
		".abcd/development/intents/drafts/itd-5-five.md":   "\nrelated_issues: [rdi-5]\n",
	} {
		got := readTree(t, repo, rel)
		if !strings.Contains(got, want) {
			t.Errorf("%s does not carry %q:\n%s", rel, strings.TrimSpace(want), got)
		}
		if strings.Contains(got, "promoted_to") || strings.Contains(got, "promoted_from") {
			t.Errorf("%s still carries a retired key:\n%s", rel, got)
		}
	}
	if got := readTree(t, repo, ".abcd/work/issues/open/iss-6-six.md"); got != untouched {
		t.Errorf("a record carrying no retired key was rewritten:\n%s", got)
	}

	list, err := List(ListRequest{RepoRoot: repo, IssuesRoot: ir, State: StateAll})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Skipped) != 0 {
		t.Fatalf("the migrated ledger still has records the reader skips: %+v", list.Skipped)
	}
	corpus, err := intent.Load(repo)
	if err != nil {
		t.Fatal(err)
	}
	if it, _ := corpus.Lookup("itd-4"); strings.Join(it.RelatedIssues, ",") != "iss-4" {
		t.Fatalf("itd-4 related_issues = %q, want [iss-4]", it.RelatedIssues)
	}
	iss1 := readIssue(t, ir, "iss-1")
	if into, err := PromotedInto(repo, iss1); err != nil || into != "itd-1" {
		t.Fatalf("iss-1 must read as promoted into itd-1 after the migration; got %q, %v", into, err)
	}

	again, err := Migrate(MigrateRequest{RepoRoot: repo, IssuesRoot: ir, Apply: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Changes) != 0 {
		t.Fatalf("a second run must change nothing, got %+v", again.Changes)
	}
}
