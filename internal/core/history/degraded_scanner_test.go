package history

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// degradeScanner writes a per-repo .abcd/config/pii.json that cannot be parsed.
// scanner.New still returns a USABLE scanner on that path — it falls back to the
// bundled pattern set — so the repository's own detectors are silently dropped
// and ScanText has no way to say so in-band. Unavailable() is the only signal,
// and the guards under test are the only things that read it.
func degradeScanner(t *testing.T, repoRoot string) {
	t.Helper()
	dir := filepath.Join(repoRoot, ".abcd", "config")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pii.json"), []byte("{ this is not json"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// storedRecords lists the record filenames actually on disk, which is the only
// proof that matters for a fail-closed write path: a refusal that still left a
// file behind is not a refusal.
func storedRecords(t *testing.T, home string) []string {
	t.Helper()
	dir := filepath.Join(home, ".abcd", "transcripts", testRootSHA, "records")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		// Records only. The lock file the write path takes before the guard runs
		// lives in the same directory and is not a stored transcript.
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			out = append(out, e.Name())
		}
	}
	return out
}

// TestCaptureRefusesADegradedScanner arms the refusal that protects every
// capture from a weakened secret scanner (iss-2609091915350221). The guard is
// older than the sub-agent work and was never given a detector of its own, so
// nothing would have noticed it being removed, reordered behind the write, or
// made conditional — and an unasserted guard on a fail-closed path is
// indistinguishable from an absent one until the day it matters.
//
// The transcript carries a PAT the BUNDLED patterns would catch, so the
// assertion is not "the secret leaked": it is that the store stays EMPTY.
// Capturing under a silently weakened pattern set and reporting success is the
// failure, whatever the bundled half happens to cover.
func TestCaptureRefusesADegradedScanner(t *testing.T) {
	repoRoot, home := setupStore(t)
	degradeScanner(t, repoRoot)

	transcript := "assistant: the token is ghp_" + strings.Repeat("a", 40) + "\n"
	res, err := Capture(repoRoot, testRootSHA, []byte(transcript),
		CaptureMeta{SessionID: "sess-degraded", Kind: "native"})
	if err == nil {
		t.Fatalf("Capture stored a transcript under a degraded scanner (wrote=%v, path=%q)", res.Wrote, res.Record.Path)
	}
	if !strings.Contains(err.Error(), "degraded scanner") {
		t.Errorf("the refusal must name the degraded scanner so the caller can repair it; got %v", err)
	}
	if res.Wrote {
		t.Error("a refused capture must not report Wrote=true")
	}
	if got := storedRecords(t, home); len(got) != 0 {
		t.Errorf("a refused capture must leave the store empty; found %v", got)
	}
}

// TestMigrateRefusesADegradedScanner is the same guard on the other history
// write path. Migration recovers externally supplied lineage out of a record's
// own body and writes it into frontmatter, so it redacts under the destination
// repository's configuration exactly as Capture does — and it must refuse on the
// same terms, leaving the record it was about to rewrite byte-identical.
func TestMigrateRefusesADegradedScanner(t *testing.T) {
	repoRoot, home := setupStore(t)
	const full = "5a9221e2-fa77-4be5-84d3-779199c449d7"
	path := planted(t, home, "20260101T000000.000000000Z-5a9221e2--agent-acf07c33.md",
		compositeRecord("5a9221e2--agent-acf07c33", full))
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	degradeScanner(t, repoRoot)

	res, err := Migrate(testRootSHA, MigrateOptions{RepoRoot: repoRoot, Apply: true})
	if err == nil {
		t.Fatalf("Migrate rewrote records under a degraded scanner: %+v", res)
	}
	if !strings.Contains(err.Error(), "degraded scanner") {
		t.Errorf("the refusal must name the degraded scanner; got %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Errorf("a refused migration must leave the record untouched;\nbefore:\n%s\nafter:\n%s", before, after)
	}
}
