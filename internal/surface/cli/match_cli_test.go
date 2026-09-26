package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The filing-time match on the front door (itd-2609212137116617): `capture` and
// the quoted-text `intent` create print each link they wrote, --json carries
// the whole outcome with the near misses and their scores, and a configuration
// the reader refuses files the record unlinked rather than refusing it.

const (
	cliFinding = "The capture ledger reader silently skips a record whose frontmatter carries " +
		"a duplicated key, so the finding disappears from every listing without a warning."
	cliDouble = "Capture ledger reader silently skips any record whose frontmatter carries a " +
		"duplicated key: the finding disappears from every listing, and no warning is printed."
	cliFiller1 = "The site builder renders a stale anchor for a heading renamed since the last build."
	cliFiller2 = "The history store drops a transcript that exceeds its byte budget without saying so."
)

// matchRepo is a checkout with its own HOME, so the machine configuration
// layer is the fixture's, and two unrelated issues to weigh terms against.
func matchRepo(t *testing.T) string {
	t.Helper()
	repo := captureLedgerRepo(t)
	t.Setenv("HOME", filepath.Join(t.TempDir(), "home"))
	if err := os.MkdirAll(os.Getenv("HOME"), 0o700); err != nil {
		t.Fatal(err)
	}
	runCLI(t, "capture", cliFiller1)
	runCLI(t, "capture", cliFiller2)
	return repo
}

type captureJSON struct {
	ID    string `json:"id"`
	Path  string `json:"path"`
	Match *struct {
		Threshold float64 `json:"threshold"`
		Skipped   string  `json:"skipped"`
		Matches   []struct {
			ID       string  `json:"id"`
			Relation string  `json:"relation"`
			Score    float64 `json:"score"`
			Linked   bool    `json:"linked"`
		} `json:"matches"`
		NearMisses []struct {
			ID    string  `json:"id"`
			Score float64 `json:"score"`
		} `json:"near_misses"`
	} `json:"match"`
}

func captureJSONOf(t *testing.T, out []byte) captureJSON {
	t.Helper()
	var c captureJSON
	if err := json.Unmarshal(out, &c); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	return c
}

func TestCaptureVerbLinksAndPrintsTheMatch(t *testing.T) {
	repo := matchRepo(t)
	first := captureJSONOf(t, runCLI(t, "capture", cliFinding, "--json"))

	out := string(runCLI(t, "capture", cliDouble))
	if !strings.Contains(out, "matched "+first.ID+" — duplicates") || !strings.Contains(out, "link written") {
		t.Fatalf("the verb did not print the match against %s:\n%s", first.ID, out)
	}
	// The newest record is the double; it carries the link.
	matches, _ := filepath.Glob(filepath.Join(repo, ".abcd/work/issues/open/iss-*.md"))
	linked := 0
	for _, m := range matches {
		b, _ := os.ReadFile(m)
		if strings.Contains(string(b), "\nduplicates: ["+first.ID+"]\n") {
			linked++
		}
	}
	if linked != 1 {
		t.Fatalf("%d record(s) carry the duplicates link, want 1", linked)
	}
}

func TestCaptureVerbJSONListsNearMisses(t *testing.T) {
	repo := matchRepo(t)
	runCLI(t, "capture", cliFinding)
	if err := os.WriteFile(filepath.Join(repo, ".abcd/config.json"), []byte(`{"match":{"threshold":0.99}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	c := captureJSONOf(t, runCLI(t, "capture", cliDouble, "--json"))
	if c.Match == nil || c.Match.Threshold != 0.99 || len(c.Match.Matches) != 0 {
		t.Fatalf("match = %+v, want nothing above the configured 0.99", c.Match)
	}
	if len(c.Match.NearMisses) == 0 || c.Match.NearMisses[0].Score <= 0 {
		t.Fatalf("near misses = %+v, want the finding with its score", c.Match.NearMisses)
	}
	b, _ := os.ReadFile(filepath.Join(repo, c.Path))
	if strings.Contains(string(b), "duplicates:") || strings.Contains(string(b), "refines:") {
		t.Fatalf("a link was written below the threshold:\n%s", b)
	}
}

func TestCaptureVerbFilesThroughARefusedConfiguration(t *testing.T) {
	repo := matchRepo(t)
	runCLI(t, "capture", cliFinding)
	if err := os.WriteFile(filepath.Join(repo, ".abcd/config.json"), []byte(`{"match":{"treshold":0.5}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	out := string(runCLI(t, "capture", cliDouble))
	if !strings.Contains(out, "captured iss-") || !strings.Contains(out, "not matched") || !strings.Contains(out, "match.treshold") {
		t.Fatalf("want the capture filed and the refused key named:\n%s", out)
	}
}

func TestIntentCreateVerbLinksAndPrintsTheMatch(t *testing.T) {
	repo := matchRepo(t)
	dir := filepath.Join(repo, ".abcd/development/intents/planned")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "itd-9-reader.md"), []byte("---\nid: itd-9\nslug: reader\nspec_id: null\nkind: null\n---\n\n# Ledger reader skips\n\n## Press Release\n\n> "+cliFinding+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := string(runCLI(t, "intent", cliDouble))
	if !strings.Contains(out, "created itd-") || !strings.Contains(out, "matched itd-9 — duplicates") {
		t.Fatalf("the create did not print the match against itd-9:\n%s", out)
	}
	// Filed again, the text now doubles itd-9 and the draft just created.
	c := captureJSONOf(t, runCLI(t, "intent", cliDouble+" Filed once more.", "--json"))
	if c.Match == nil || len(c.Match.Matches) != 2 {
		t.Fatalf("--json match = %+v, want itd-9 and the first draft", c.Match)
	}
	named := false
	for _, m := range c.Match.Matches {
		named = named || (m.ID == "itd-9" && m.Linked)
	}
	if !named {
		t.Fatalf("--json match = %+v, want itd-9 linked", c.Match)
	}
}
