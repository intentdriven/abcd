package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The partial-delivery close is reachable from the CLI: one command closes the
// spec, mints the remainder, and says the intent stayed planned.
func TestSpecCloseRemainderIsWiredAtTheCLI(t *testing.T) {
	repo, _ := specStoreFixture(t)
	plantPlannedIntent(t, repo, "itd-10", "alpha", "spc-1")

	out := runCLI(t, "spec", "close", "spc-1", "--remainder", "the-rest", "--json")
	var res struct {
		IntentMoved bool     `json:"intent_moved"`
		OpenSpecs   []string `json:"open_specs"`
		Remainder   struct {
			ID     string `json:"id"`
			Intent string `json:"intent"`
			Status string `json:"status"`
		} `json:"remainder"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("spec close --json: not JSON: %v\n%s", err, out)
	}
	if res.IntentMoved {
		t.Fatalf("the intent must stay planned while the remainder is open: %s", out)
	}
	if res.Remainder.Intent != "itd-10" || res.Remainder.Status != "open" {
		t.Fatalf("the remainder must be an open spec on itd-10: %s", out)
	}
	if len(res.OpenSpecs) != 1 || res.OpenSpecs[0] != res.Remainder.ID {
		t.Fatalf("the remainder must be reported as what holds the intent: %s", out)
	}
	// Closing it ships the intent.
	out = runCLI(t, "spec", "close", res.Remainder.ID, "--json")
	if !strings.Contains(string(out), `"intent_moved": true`) {
		t.Fatalf("closing the last open spec must ship the intent: %s", out)
	}
}

// A production mode with no remainder to stamp is refused rather than ignored.
func TestSpecCloseRefusesProductionModeWithoutARemainder(t *testing.T) {
	repo, _ := specStoreFixture(t)
	plantPlannedIntent(t, repo, "itd-10", "alpha", "spc-1")

	out, err := runCLIErr(t, "spec", "close", "spc-1", "--production-mode", "hand-written")
	if err == nil {
		t.Fatalf("--production-mode with no --remainder must refuse: %s", out)
	}
	if !strings.Contains(err.Error(), "--remainder") {
		t.Fatalf("the refusal must name the flag it has nothing to stamp for: %s", out)
	}
}

// F4: --remainder on a SHIPPED intent is refused at the CLI, before anything is
// minted. Accepting it produced the "stays shipped — still open" contradiction
// invariant 17 forbids.
func TestSpecCloseRefusesARemainderOnAShippedIntent(t *testing.T) {
	repo, _ := specStoreFixture(t)
	writeShippedIntent(t, repo, "itd-10", "alpha", "spc-1")

	out, err := runCLIErr(t, "spec", "close", "spc-1", "--remainder", "the-rest")
	if err == nil {
		t.Fatalf("--remainder on a shipped intent must exit non-zero: %s", out)
	}
	if !strings.Contains(err.Error(), "shipped") {
		t.Fatalf("the refusal must say why: %v", err)
	}
	if matches := openSpecsWithSlug(t, repo, "the-rest"); len(matches) != 0 {
		t.Fatalf("nothing may be minted on the refusal: %v", matches)
	}
}

// F4: a close that leaves a SHIPPED intent with an open spec is a record that
// disagrees with itself; the render says so rather than printing "stays shipped
// — still open", which reads as an ordinary outcome.
func TestSpecCloseNeverPairsStaysShippedWithStillOpen(t *testing.T) {
	repo, _ := specStoreFixture(t)
	writeShippedIntent(t, repo, "itd-10", "alpha", "spc-1")
	writeSpecRecord(t, repo, "closed", "spc-1-alpha.md",
		"---\nid: spc-1\nslug: alpha\nintent: itd-10\n---\n# alpha\n")
	if err := os.Remove(filepath.Join(repo, ".abcd", "development", "specs", "open", "spc-1-alpha.md")); err != nil {
		t.Fatal(err)
	}
	writeSpecRecord(t, repo, "open", "spc-2-rest.md",
		"---\nid: spc-2\nslug: rest\nintent: itd-10\n---\n# rest\n")

	out := string(runCLI(t, "spec", "close", "spc-1"))
	if strings.Contains(out, "stays shipped") {
		t.Fatalf("a shipped intent has no open spec — the render must not present it as an ordinary outcome:\n%s", out)
	}
	if !strings.Contains(out, "spc-2") {
		t.Fatalf("the render must still name the spec that contradicts the shipped intent:\n%s", out)
	}
}

// writeShippedIntent plants a shipped intent record carrying both link sides.
func writeShippedIntent(t *testing.T, root, id, slug, specID string) {
	t.Helper()
	dir := filepath.Join(root, ".abcd", "development", "intents", "shipped")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nid: " + id + "\nslug: " + slug + "\nspec_id: " + specID +
		"\nkind: standalone\nimpact: fix\n---\n# " + slug +
		"\n\n## Scope Conditions\n\nNONE\n\n## Acceptance Criteria\n\n- ok\n\n## Audit Notes\n"
	if err := os.WriteFile(filepath.Join(dir, id+"-"+slug+".md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// openSpecsWithSlug lists the open spec files whose name ends in the given slug.
func openSpecsWithSlug(t *testing.T, root, slug string) []string {
	t.Helper()
	m, err := filepath.Glob(filepath.Join(root, ".abcd", "development", "specs", "open", "spc-*-"+slug+".md"))
	if err != nil {
		t.Fatal(err)
	}
	return m
}
