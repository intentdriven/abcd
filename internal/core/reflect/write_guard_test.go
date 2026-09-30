package reflect

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// degradeScanner makes the per-repo scanner config fail to load, the state in
// which scanner.New still returns a scanner that silently lacks the repo's own
// detectors and only Unavailable() says so.
func degradeScanner(t *testing.T, root string, bad func(path string)) {
	t.Helper()
	dir := filepath.Join(root, ".abcd", "config")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	bad(filepath.Join(dir, "pii.json"))
}

// The retrospective is committed prose, so a scanner that cannot vouch for its
// own pattern set refuses the write, and nothing lands.
func TestWriteRefusesADegradedScannerAndWritesNothing(t *testing.T) {
	for name, bad := range map[string]func(string){
		"invalid json": func(p string) {
			if err := os.WriteFile(p, []byte("{ this is not json"), 0o644); err != nil {
				t.Fatal(err)
			}
		},
		"not a regular file": func(p string) {
			if err := os.Mkdir(p, 0o755); err != nil {
				t.Fatal(err)
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			r := releaseRepo(t)
			degradeScanner(t, r.Root(), bad)
			_, err := Write(r.Root(), WriteRequest{Tag: "v0.2.0", Answers: fullAnswers(), ProceedDespiteUnshipped: true, Now: fixedNow})
			if err == nil || !strings.Contains(err.Error(), "degraded scanner") {
				t.Fatalf("Write under a degraded scanner: want a refusal naming it, got %v", err)
			}
			if exists(t, abs(r, outputRel("v0.2.0"))) {
				t.Error("a refused write left a retrospective behind")
			}
		})
	}
}
