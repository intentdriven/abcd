package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFilenameBarHoldsToSecretsOnly pins iss-2609090951282192. The ruling
// behind the page-filename refusal asked for a bar narrow enough that an
// ordinary slug is not refused: secrets only. Selecting on the scanner's
// hard_fail severity also took in two identity kinds — the caller's own
// machine account name and a banned real name — so on a machine whose account
// name is an ordinary word, a page whose slug carries that word was refused
// with advice to repair the slug, when what matched was the machine. The bar
// is the secret patterns; a token-shaped slug is still refused
// (TestWriteRefusesASecretShapedFilename).
func TestFilenameBarHoldsToSecretsOnly(t *testing.T) {
	t.Setenv("HOME", "/Users/garden")
	repo := t.TempDir()
	src := writeSource(t, repo, "notes.md", "Plan the garden beds before spring.\n")
	distiller := func(_ string, sourceBlock map[string]any) ([]map[string]any, error) {
		return []map[string]any{{
			"type": "topic", "domain": "home", "slug": "garden-plan",
			"body": "# Beds\nPlan the beds before spring.\n", "source": sourceBlock,
		}}, nil
	}
	if _, err := Ingest(IngestRequest{RepoRoot: repo, Source: src, Distiller: distiller, Now: fixedNow}); err != nil {
		t.Fatalf("an ordinary slug that happens to carry the machine account name was refused: %v", err)
	}
	if _, err := os.Stat(filepath.Join(Dir(repo), "topic_home_garden-plan.md")); err != nil {
		t.Fatalf("the page was not written: %v", err)
	}
	// The derived index must name the page it indexes, not a redacted stand-in.
	idx, err := os.ReadFile(filepath.Join(Dir(repo), "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(idx), "topic_home_garden-plan.md") {
		t.Errorf("index.md does not name the page it indexes:\n%s", idx)
	}
}
