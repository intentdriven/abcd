package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/recordid"
)

const overTheDefaultCap = "a-slug-of-forty-one-characters-it-is-long" // 41 characters

func slugCapConfig() Config {
	return Config{Rules: map[string]RuleConfig{ruleRecordSlugCap: {Enabled: true, Severity: "blocker"}}}
}

func slugCapRepo(t *testing.T, cfg, name, body string) string {
	t.Helper()
	repo := t.TempDir()
	path := filepath.Join(repo, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if cfg != "" {
		if err := os.MkdirAll(filepath.Join(repo, ".abcd"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repo, ".abcd", "config.json"), []byte(cfg), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return repo
}

// TestRecordSlugCapRuleHoldsARecordToTheRepositorysCap is iss-2611052241331003:
// the linter reads the one value the minting verbs cut at, and judges a name a
// person wrote by hand as it would one a verb minted. A longer name written under
// an earlier, wider cap has no standing of its own: the value in force decides,
// as the tree test the rule replaces decided.
func TestRecordSlugCapRuleHoldsARecordToTheRepositorysCap(t *testing.T) {
	if len(overTheDefaultCap) != 41 {
		t.Fatalf("premise: the slug is %d characters", len(overTheDefaultCap))
	}
	rel := ".abcd/work/decisions/dcn-2611050000000001-" + overTheDefaultCap + ".md"
	body := "---\nid: dcn-2611050000000001\nslug: " + overTheDefaultCap + "\ndate: 2026-11-05\n---\n\n2026-11-05 — text\n"

	findings, err := Lint(slugCapConfig(), slugCapRepo(t, "", rel, body))
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 2 {
		t.Fatalf("the filename and the field are both over the default cap: %+v", findings)
	}
	for _, f := range findings {
		if f.RuleID != ruleRecordSlugCap || f.Severity != "blocker" || f.File != rel ||
			!strings.Contains(f.Message, "41") || !strings.Contains(f.Message, "40") || !strings.Contains(f.Message, "records.slug_cap") {
			t.Errorf("the finding names the record, the length, the cap and the key: %+v", f)
		}
	}

	findings, err = Lint(slugCapConfig(), slugCapRepo(t, `{"records": {"slug_cap": 60}}`, rel, body))
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Fatalf("a repository that sets 60 allows 41: %+v", findings)
	}

	if _, err := Lint(slugCapConfig(), slugCapRepo(t, `{"records": {"slug_cap": 0}}`, rel, body)); err == nil ||
		!strings.Contains(err.Error(), "records.slug_cap") {
		t.Fatalf("a nonsense cap: %v; want the lint refused naming the key", err)
	}
}

// A field longer than its filename is judged on its own, and a store's other
// files are not records.
func TestRecordSlugCapRuleReadsTheFieldAndOnlyRecords(t *testing.T) {
	rel := ".abcd/development/intents/planned/itd-20-short.md"
	repo := slugCapRepo(t, "", rel, "---\nid: itd-20\nslug: \""+overTheDefaultCap+"\"\n---\n\n# Twenty\n")
	readme := filepath.Join(repo, ".abcd/development/intents/README-with-a-name-long-enough-to-pass-any-cap-whatever.md")
	if err := os.WriteFile(readme, []byte("# Intents\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, err := Lint(slugCapConfig(), repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || !strings.Contains(findings[0].Message, "slug field") {
		t.Fatalf("one finding, on the field: %+v", findings)
	}
}

// SlugCapFindings judges the named paths alone, which is what the commit-time
// check hands it, and a path outside every store is no record.
func TestSlugCapFindingsJudgesTheNamedPathsAlone(t *testing.T) {
	over := ".abcd/work/issues/open/iss-1-" + overTheDefaultCap + ".md"
	repo := slugCapRepo(t, "", over, "---\nid: iss-1\nslug: "+overTheDefaultCap+"\n---\n")
	if got, err := SlugCapFindings(repo, []string{"docs/iss-1-" + overTheDefaultCap + ".md", "main.go"}); err != nil || len(got) != 0 {
		t.Fatalf("paths outside the stores: %+v, %v", got, err)
	}
	got, err := SlugCapFindings(repo, []string{over})
	if err != nil || len(got) != 2 {
		t.Fatalf("the named record: %+v, %v", got, err)
	}
	// A staged path git holds and the working tree does not still has its name to judge.
	gone := ".abcd/work/issues/open/iss-2-" + overTheDefaultCap + ".md"
	got, err = SlugCapFindings(repo, []string{gone})
	if err != nil || len(got) != 1 {
		t.Fatalf("a path with no file: %+v, %v", got, err)
	}
}

// TestCommittedRecordSlugsWithinTheCap holds this repository's own records to its
// own cap, every family and bucket, and proves the walk reads each store.
func TestCommittedRecordSlugsWithinTheCap(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	abs, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	findings, err := checkRecordSlugCap(abs, RuleConfig{Enabled: true, Severity: "blocker"})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		t.Errorf("%s: %s", f.File, f.Message)
	}
	rels, err := recordid.RecordFiles(abs)
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, rel := range rels {
		for _, dir := range []string{".abcd/development/intents/", ".abcd/development/specs/", ".abcd/work/issues/", ".abcd/development/decisions/adrs/", ".abcd/work/decisions/"} {
			if strings.HasPrefix(rel, dir) {
				counts[dir]++
			}
		}
	}
	if len(counts) != 5 {
		t.Errorf("the walk did not read every store: %v", counts)
	}
	t.Logf("records per store: %v", counts)
}
