package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/scanner/augmenttest"
)

func packSourceWithValue(t *testing.T) string {
	t.Helper()
	source := t.TempDir()
	if err := os.MkdirAll(filepath.Join(source, ".abcd", "development"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "AGENTS.md"), []byte("# Router\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// An issue record travels verbatim, so the value reaches the planned
	// content the pack scans.
	issues := filepath.Join(source, ".abcd", "work", "issues", "open")
	if err := os.MkdirAll(issues, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nid: \"iss-1\"\n---\n\nthe config holds " + augmenttest.Value + " in prose\n"
	if err := os.WriteFile(filepath.Join(issues, "iss-1-augmented.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return source
}

// TestDisembarkPackReportsTheAugmentersFinding: the source repository's
// opt-in augmenter reaches the CLI's pack scan, so what it flags refuses the
// lifeboat, and the refusal does not echo the value (iss-2608291814575788).
func TestDisembarkPackReportsTheAugmentersFinding(t *testing.T) {
	augmenttest.Install(t, augmenttest.Fake())
	source := packSourceWithValue(t)
	dest := filepath.Join(t.TempDir(), "lifeboat")
	var stdout, stderr bytes.Buffer
	code := runPack(t, &stdout, &stderr, "disembark", "pack", source, dest)
	if code == 0 {
		t.Fatalf("pack succeeded over an augmented finding\nstdout: %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), augmenttest.Kind) {
		t.Errorf("the refusal does not name the augmented finding:\n%s", stderr.String())
	}
	if strings.Contains(stdout.String()+stderr.String(), augmenttest.Value) {
		t.Errorf("the refusal echoes the value")
	}
}

// TestDisembarkPackRefusesOnTheAugmenterGap: a lifeboat is written OUT of the
// repository, as a release is, so a configured augmenter that is not installed
// refuses the pack as it refuses a launch.
func TestDisembarkPackRefusesOnTheAugmenterGap(t *testing.T) {
	augmenttest.Install(t, augmenttest.NotFound())
	source := packSourceWithValue(t)
	dest := filepath.Join(t.TempDir(), "lifeboat")
	var stdout, stderr bytes.Buffer
	code := runPack(t, &stdout, &stderr, "disembark", "pack", source, dest)
	if code != 2 {
		t.Fatalf("pack exit %d on the augmenter gap, want 2\nstdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "fake augmenter not on PATH") {
		t.Errorf("the refusal does not name the gap:\n%s", stderr.String())
	}
	if entries, err := os.ReadDir(dest); err == nil && len(entries) > 0 {
		t.Errorf("a refusal wrote %d entr(ies) into the destination", len(entries))
	}
}
