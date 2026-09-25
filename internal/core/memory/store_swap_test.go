package memory

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The tests in this file pin the second half of iss-2608291814572914: not only
// that a store handle reads the directory it vetted, but that every read a verb
// makes after opening the handle goes through it. Each one swaps the store
// directory for a symlink to a directory outside the repository at the moment
// openStore hands the handle back (the storeOpened seam), and plants in that
// outside directory the file a read by path would pick up. A read through the
// handle keeps reading the vetted directory; a read by path follows the swap.

// swapStoreOnOpen arranges for the repository's store directory to be replaced
// by a symlink to outside the first time openStore returns a present handle.
// It returns where the vetted directory was moved to.
func swapStoreOnOpen(t *testing.T, repo, outside string) string {
	t.Helper()
	mem := Dir(repo)
	moved := mem + ".vetted"
	swapped := false
	storeOpened = func() {
		if swapped {
			return
		}
		swapped = true
		if err := os.Rename(mem, moved); err != nil {
			t.Fatalf("swap: %v", err)
		}
		if err := os.Symlink(outside, mem); err != nil {
			t.Fatalf("swap: %v", err)
		}
	}
	t.Cleanup(func() { storeOpened = nil })
	return moved
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// quotingStore seeds a store with one page quoting twelve words of an external
// source the registry knows at 1000 tokens: under the default budget it is
// clean, and under a five-word ceiling or an empty registry it is not.
func quotingStore(t *testing.T, repo string) {
	t.Helper()
	sourceHash := strings.Repeat("a", 64)
	quote := "one two three four five six seven eight nine ten eleven twelve"
	page := "---\nsource:\n  class: external_pdf\n  citation: { type: knowledge }\n  licence: MIT\n" +
		"  source_hash: " + sourceHash + "\n  ingested_at: 2026-07-06\n" +
		"topic_hash: " + strings.Repeat("b", 64) + "\n---\n\n# Rotation practice\n\n> " + quote + "\n"
	writeFile(t, filepath.Join(Dir(repo), "topic_auth_rotation.md"), page)
	writeFile(t, SourcesIndexPath(repo), fmt.Sprintf(`{%q: {"source_token_count": 1000}}`, sourceHash))
}

// plantOutside fills the directory the swap points at with the three files a
// read by path would pick up: a five-word quotation ceiling, an empty
// registry, and a coverage index carrying a fingerprint nothing inside wrote.
func plantOutside(t *testing.T) string {
	t.Helper()
	outside := t.TempDir()
	writeFile(t, filepath.Join(outside, "config.json"), `{"quotation_budget": {"max_contiguous_quote_words": 5}}`)
	writeFile(t, filepath.Join(outside, ".sources_index.json"), `{}`)
	writeFile(t, filepath.Join(outside, coverageIndexName), `{"fingerprint": "planted-outside"}`)
	return outside
}

// TestLintReadsOnlyThroughTheStoreHandle: the page lint's quotation check read
// config.json and the registry by path, and the coverage lint read config.json
// and the stored fingerprint by path, so a swap after the handle opened steered
// the budget, the registry lookups and the staleness report.
func TestLintReadsOnlyThroughTheStoreHandle(t *testing.T) {
	repo := t.TempDir()
	quotingStore(t, repo)
	swapStoreOnOpen(t, repo, plantOutside(t))

	lr, err := Lint(LintRequest{RepoRoot: repo, Now: fixedNow})
	if err != nil {
		t.Fatalf("lint: %v", err)
	}
	for _, f := range lr.Findings {
		if f.Code == "MQ001" || f.Code == "MQ003" {
			t.Errorf("lint judged the page against files beyond the swap: %s %s", f.Code, f.Message)
		}
	}
	if fp, ok := lr.CoverageIndex["old_fingerprint"]; ok && fp != nil {
		t.Errorf("coverage lint read the stored fingerprint beyond the swap: %v", fp)
	}
}

// TestCoverageLintWritesOnlyThroughTheStoreHandle pins iss-2609252100150846:
// the coverage index was written by path after the handle vetted the store,
// with no lock or writer check behind it, so a swap in between landed the
// index outside the repository.
func TestCoverageLintWritesOnlyThroughTheStoreHandle(t *testing.T) {
	repo := t.TempDir()
	quotingStore(t, repo)
	outside := plantOutside(t)
	vetted := swapStoreOnOpen(t, repo, outside)

	if _, err := Lint(LintRequest{RepoRoot: repo, Now: fixedNow}); err != nil {
		t.Fatalf("lint: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(outside, coverageIndexName))
	if err != nil || !strings.Contains(string(raw), "planted-outside") {
		t.Errorf("coverage lint wrote its index beyond the swap: %q (%v)", raw, err)
	}
	if _, err := os.Stat(filepath.Join(vetted, coverageIndexName)); err != nil {
		t.Errorf("coverage lint did not write its index into the store it opened: %v", err)
	}
}

// TestBareHeadroomReadsOnlyThroughTheStoreHandle: the headroom lines read the
// quotation budget by path, so a budget beyond the swap changed the fingerprint
// and reported a current coverage index as stale.
func TestBareHeadroomReadsOnlyThroughTheStoreHandle(t *testing.T) {
	repo := t.TempDir()
	quotingStore(t, repo)
	if _, err := Lint(LintRequest{RepoRoot: repo, Now: fixedNow}); err != nil {
		t.Fatalf("fixture lint: %v", err)
	}
	swapStoreOnOpen(t, repo, plantOutside(t))

	status, err := Bare(repo)
	if err != nil {
		t.Fatalf("bare: %v", err)
	}
	for _, line := range status.Headroom {
		if strings.Contains(line, "stale") || strings.Contains(line, "unavailable") {
			t.Fatalf("headroom judged the index against a budget beyond the swap: %q", line)
		}
	}
}

// TestIngestReadsTheRegistryThroughTheStoreHandle: the ingest dedup loaded the
// registry by path with the handle open, so a malformed registry beyond the
// swap was read before the writer refused the symlinked store.
func TestIngestReadsTheRegistryThroughTheStoreHandle(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, SourcesIndexPath(repo), `{}`)
	outside := t.TempDir()
	writeFile(t, filepath.Join(outside, ".sources_index.json"), `{not json`)
	swapStoreOnOpen(t, repo, outside)

	src := writeSource(t, t.TempDir(), "rotation.md", "Rotate tokens every 24 hours.\n")
	_, err := Ingest(IngestRequest{
		RepoRoot:  repo,
		Source:    src,
		Distiller: oneTopicDistiller("topic", "auth", "tokens", "# Token rotation\nRotate tokens every 24 hours."),
		Now:       fixedNow,
	})
	var format *RegistryFormatError
	if errors.As(err, &format) {
		t.Fatalf("ingest read the registry beyond the swap: %v", err)
	}
	var unsafe *UnsafeStorePathError
	if !errors.As(err, &unsafe) {
		t.Fatalf("ingest over a swapped store = %v (%T); want the writer's *UnsafeStorePathError", err, err)
	}
}
