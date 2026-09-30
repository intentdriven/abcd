package memory

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/adapter/scanner/augmenttest"
)

func ingestValue(t *testing.T, repo string) (IngestResult, error) {
	t.Helper()
	return Ingest(IngestRequest{
		RepoRoot: repo,
		Source:   docURL,
		Fetcher:  staticFetcher(nil, textFetched(docURL, "text/plain", "An ordinary paragraph of prose.")),
		Distiller: oneTopicDistiller("topic", "auth", "tokens",
			"# Token rotation\nThe config holds "+augmenttest.Value+" in prose."),
		Now: fixedNow,
	})
}

// storeHolds reports whether any file under the memory store holds s.
func storeHolds(t *testing.T, repo, s string) bool {
	t.Helper()
	found := false
	_ = filepath.WalkDir(Dir(repo), func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if b, rerr := os.ReadFile(p); rerr == nil && strings.Contains(string(b), s) {
			found = true
		}
		return nil
	})
	return found
}

// TestIngestReportsTheAugmentersFinding: the repository's opt-in augmenter
// reaches the memory store's redactor, so what it flags never lands
// (iss-2608291814575788).
func TestIngestReportsTheAugmentersFinding(t *testing.T) {
	augmenttest.Install(t, augmenttest.Fake())
	repo := t.TempDir()
	if _, err := ingestValue(t, repo); err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if storeHolds(t, repo, augmenttest.Value) {
		t.Fatal("the augmented value landed in the memory store")
	}
}

// TestIngestRecordsTheAugmenterGap: a configured augmenter that is not
// installed does not stop the ingest, which writes on the native scanner and
// records the gap in its receipt.
func TestIngestRecordsTheAugmenterGap(t *testing.T) {
	augmenttest.Install(t, augmenttest.NotFound())
	res, err := ingestValue(t, t.TempDir())
	if err != nil {
		t.Fatalf("Ingest refused on the gap: %v", err)
	}
	if !strings.Contains(res.ScanGap, "fake augmenter not on PATH") {
		t.Fatalf("the receipt does not record the gap: %q", res.ScanGap)
	}
}

// TestIngestRefusesAFailedAugmenterRun: a run that fails during the ingest
// degrades the scanner, and the store refuses as it does on a broken pii.json.
func TestIngestRefusesAFailedAugmenterRun(t *testing.T) {
	augmenttest.Install(t, &augmenttest.Func{F: func(string, string) ([]scanner.Finding, error) {
		return nil, errors.New("run failed")
	}})
	repo := t.TempDir()
	if _, err := ingestValue(t, repo); err == nil || !strings.Contains(err.Error(), "run failed") {
		t.Fatalf("Ingest = %v, want a refusal naming the failed run", err)
	}
}
