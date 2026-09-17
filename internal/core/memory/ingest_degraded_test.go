package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestIngestRefusesADegradedScanner arms the memory store's half of the
// degraded-scanner refusal (the sweep on iss-2609091915350221). The lint side of
// openStoreRedactor has a detector (GHSA-xj89-cc2c-wgwr, in
// lint_residue_test.go, where a degraded scanner is a BLOCKER FINDING rather
// than a refusal); the WRITE side had none, so the two halves of one seam were
// asserted in opposite directions with only one of them tested.
//
// An unparseable per-repo pii.json leaves scanner.New returning a usable scanner
// with the repository's own detectors silently dropped, and ScanText cannot say
// so in-band. The ingest must refuse before anything lands.
func TestIngestRefusesADegradedScanner(t *testing.T) {
	repo := t.TempDir()
	cfg := filepath.Join(repo, ".abcd", "config")
	if err := os.MkdirAll(cfg, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg, "pii.json"), []byte("{ this is not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Ingest(IngestRequest{
		RepoRoot: repo,
		Source:   docURL,
		Fetcher:  staticFetcher(nil, textFetched(docURL, "text/plain", "An ordinary paragraph of prose.")),
		// A page the schema ACCEPTS, so the only thing that can refuse the write is
		// the guard under test. A malformed fixture would make the test pass for
		// the wrong reason once the guard was gone.
		Distiller: func(string, map[string]any) ([]map[string]any, error) {
			return []map[string]any{{
				"type": "topic", "domain": "auth", "slug": "tokens",
				"body": "# Token rotation\nRotate tokens every 24 hours.",
			}}, nil
		},
		Now: fixedNow,
	})
	if err == nil {
		t.Fatal("Ingest wrote into the store with a degraded scanner")
	}
	if !strings.Contains(err.Error(), "degraded scanner") {
		t.Errorf("the refusal must name the degraded scanner so the caller can repair pii.json; got %v", err)
	}
	if _, err := os.Stat(SourcesIndexPath(repo)); !os.IsNotExist(err) {
		t.Errorf("a refused ingest must not create the sources index (stat err = %v)", err)
	}
	// As in TestIngestRejectedSourceWritesNothing: an absent store is itself a
	// pass, so the read is only asserted on when it succeeds.
	if entries, err := os.ReadDir(Dir(repo)); err == nil && len(entries) > 0 {
		t.Errorf("a refused ingest must not write into the store, found %d entries", len(entries))
	} else if err != nil && !os.IsNotExist(err) {
		t.Fatalf("reading the store: %v", err)
	}
}
