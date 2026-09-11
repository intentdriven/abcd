package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMemoryIngestKeepOriginalPartialFailure proves the iss-30 fix is wired to
// the CLI surface: when --keep-original fails after the pages are durably
// written, `abcd memory ingest` reports the successful ingest with a warning,
// exits non-zero, and leaks no absolute path. The failure is forced by making
// the kept-originals directory a regular file.
func TestMemoryIngestKeepOriginalPartialFailure(t *testing.T) {
	repo := t.TempDir()
	// The memory front door resolves the checkout root before it writes, so the
	// fixture is a git working tree (iss-2609091729516940).
	gitInitAt(t, repo)
	t.Chdir(repo)

	src := filepath.Join(repo, "article.txt")
	if err := os.WriteFile(src, []byte("Rotate tokens every 24 hours."), 0o644); err != nil {
		t.Fatal(err)
	}
	// Make .abcd/memory/sources a regular file so storeOriginal fails.
	if err := os.MkdirAll(filepath.Join(repo, ".abcd", "memory"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".abcd", "memory", "sources"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	pages := filepath.Join(repo, "pages.json")
	if err := os.WriteFile(pages, []byte(`[{"type":"topic","domain":"auth","slug":"tokens","body":"# Rotation\nRotate tokens every 24 hours."}]`), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"memory", "ingest", src, "--keep-original", "--pages-json", pages}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected a non-zero exit when --keep-original fails\nstdout: %s\nstderr: %s", stdout.String(), stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "warning: --keep-original failed") {
		t.Fatalf("expected a --keep-original warning in the output:\n%s", out)
	}
	if !strings.Contains(out, "topic_auth_tokens.md") {
		t.Fatalf("expected the durably-written page to be reported:\n%s", out)
	}
	if strings.Contains(out, repo) || strings.Contains(stderr.String(), repo) {
		t.Fatalf("ingest output leaked the absolute repo path:\nstdout: %s\nstderr: %s", out, stderr.String())
	}
	// The page really did reach disk despite the keep-original failure.
	if _, err := os.Stat(filepath.Join(repo, ".abcd", "memory", "topic_auth_tokens.md")); err != nil {
		t.Fatalf("page not durably written: %v", err)
	}
}
