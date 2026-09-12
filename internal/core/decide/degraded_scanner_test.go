package decide

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The armed half of decide's fail-closed redaction.
//
// redactDecisionText already refused a degraded scanner, and correctly — but no
// test asserted it, which is the same position as not having the guard: a later
// edit that dropped the two lines would have gone green, and the defect would
// surface as an ADR committed with an unredacted secret in its title and in the
// filename derived from it.

// degradeDecideScanner writes a per-repo .abcd/config/pii.json that cannot be
// parsed.
//
// This is the shape the guard exists for and it is worth stating precisely:
// scanner.New STILL RETURNS A USABLE SCANNER on this path. It falls back to the
// bundled pattern set, so the repository's own detectors are silently dropped and
// ScanText has no way to say so in band — it reports findings from a weaker set
// and a caller reading only the findings sees a clean, confident answer.
// Unavailable() is the only signal that anything is wrong, and the guard under
// test is the only thing that reads it.
func degradeDecideScanner(t *testing.T, repoRoot string) {
	t.Helper()
	dir := filepath.Join(repoRoot, ".abcd", "config")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pii.json"), []byte("{ this is not json"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// adrFilesOnDisk lists what the store actually holds. A refusal that still left a
// file behind is not a refusal, and the returned value is the only proof of that
// which does not depend on the function under test telling the truth.
func adrFilesOnDisk(t *testing.T, repoRoot string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(repoRoot, filepath.FromSlash(ADRsRelDir)))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			out = append(out, e.Name())
		}
	}
	return out
}

// TestCreateRefusesADegradedScanner: an ADR is durable committed prose whose
// TITLE also becomes its filename, so a secret that survives redaction reaches
// both the body and the path. Minting under a silently weakened detector is the
// one outcome worse than not minting at all.
func TestCreateRefusesADegradedScanner(t *testing.T) {
	root := t.TempDir()
	degradeDecideScanner(t, root)

	d, err := Create(root, "The reading is commissioned before it is read")
	if err == nil {
		t.Fatalf("Create minted %+v under a degraded scanner; want a refusal", d)
	}
	if !strings.Contains(err.Error(), "degraded") {
		t.Errorf("the refusal must say the scanner is degraded, so the operator can fix the config rather than the title; got %q", err)
	}
	if files := adrFilesOnDisk(t, root); len(files) > 0 {
		t.Errorf("a refusal wrote %d record(s) anyway: %v", len(files), files)
	}
}

// The negative control. Without it the test above passes on a repository where
// Create refuses for some entirely different reason, and the guard it claims to
// arm could be deleted with no test going red.
func TestCreateMintsWhenTheScannerIsHealthy(t *testing.T) {
	root := t.TempDir()

	if _, err := Create(root, "The reading is commissioned before it is read"); err != nil {
		t.Fatalf("Create on a healthy scanner: %v", err)
	}
	if files := adrFilesOnDisk(t, root); len(files) != 1 {
		t.Fatalf("want exactly one record written, got %v", files)
	}
}
