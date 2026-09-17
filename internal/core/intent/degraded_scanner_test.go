package intent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The armed half of the intent store's fail-closed redaction.
//
// newIntentRedactor already refused a degraded scanner, and correctly — but no
// test asserted it, which in practice is the same as not having the guard: the
// two lines could be deleted and every gate would stay green. That matters more
// here than at most sites, because the guard is the ONLY thing standing between a
// weakened detector and the defect gh-486 closed — `abcd intent "<text>"` writing
// a secret token or an absolute home path into a committed record verbatim, with
// every lint gate green.

// degradeIntentScanner writes a per-repo .abcd/config/pii.json that cannot be
// parsed.
//
// The property that makes this the right fixture, and that makes the guard
// necessary at all: scanner.New STILL SUCCEEDS here. It falls back to the bundled
// pattern set, so the repository's own detectors are silently dropped, and
// ScanText then reports findings from a weaker set with no way to say in band
// that it is weaker. Unavailable() is the only signal, and the guard under test
// is the only thing that reads it.
func degradeIntentScanner(t *testing.T, repoRoot string) {
	t.Helper()
	dir := filepath.Join(repoRoot, ".abcd", "config")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pii.json"), []byte("{ this is not json"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// draftsOnDisk lists what the drafts bucket actually holds. A refusal that still
// left a record behind is not a refusal, and this is the only proof of that which
// does not rely on the function under test reporting itself honestly.
func draftsOnDisk(t *testing.T, repoRoot string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(repoRoot, filepath.FromSlash(IntentsRelDir), BucketDrafts))
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

// TestCreateFromTextRefusesADegradedScanner is the quoted-text create path, which
// is the one with nothing upstream of it: the caller's prose becomes the title,
// the body AND the filename, and no schema constrains what it holds. Storing that
// under a silently weakened detector is the fail-open this guard exists to close.
func TestCreateFromTextRefusesADegradedScanner(t *testing.T) {
	root := t.TempDir()
	degradeIntentScanner(t, root)

	it, err := CreateFromText(root, "The collector reaches the lab box directly", "", "")
	if err == nil {
		t.Fatalf("CreateFromText wrote %+v under a degraded scanner; want a refusal", it)
	}
	if !strings.Contains(err.Error(), "degraded") {
		t.Errorf("the refusal must say the scanner is degraded, so the operator fixes the config rather than the prose; got %q", err)
	}
	if files := draftsOnDisk(t, root); len(files) > 0 {
		t.Errorf("a refusal wrote %d record(s) anyway: %v", len(files), files)
	}
}

// The negative control. Without it the test above passes on a repository where
// CreateFromText refuses for some entirely unrelated reason, and the guard could
// be deleted with nothing going red.
func TestCreateFromTextWritesWhenTheScannerIsHealthy(t *testing.T) {
	root := t.TempDir()

	if _, err := CreateFromText(root, "The collector reaches the lab box directly", "", ""); err != nil {
		t.Fatalf("CreateFromText on a healthy scanner: %v", err)
	}
	if files := draftsOnDisk(t, root); len(files) != 1 {
		t.Fatalf("want exactly one draft written, got %v", files)
	}
}
