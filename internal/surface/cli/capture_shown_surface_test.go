package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCaptureListAndStatusMaskAStoredRuneAtTheFrontDoor: capture.List and
// capture.Status return a record's text as stored, for the in-process callers
// that compare it, so the mask is the front door's. A hand-edited remedy
// carrying U+202E reaches neither `capture list --json` nor `capture --json`
// raw, and both carry it masked.
func TestCaptureListAndStatusMaskAStoredRuneAtTheFrontDoor(t *testing.T) {
	repo := captureLedgerRepo(t)
	runCLI(t, "capture", "the empty list panics", "--severity", "minor", "--remedy", "Does it panic? Print a dash.", "--json")
	matches, err := filepath.Glob(filepath.Join(repo, ".abcd", "work", "issues", "open", "*.md"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("the captured record: %v %v", matches, err)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "panic? Print") {
		t.Fatalf("the record carries the remedy: %s", data)
	}
	if err := os.WriteFile(matches[0], []byte(strings.Replace(string(data), "panic? Print", "panic‮ Print", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"capture", "list", "--open", "--json"}, {"capture", "--json"}} {
		out := string(runCLI(t, args...))
		if strings.Contains(out, "‮") || strings.Contains(out, `‮`) {
			t.Errorf("%v: the stored rune reaches the payload raw: %s", args, out)
		}
		if !strings.Contains(out, "panic? Print") {
			t.Errorf("%v: the remedy is kept, masked: %s", args, out)
		}
	}
}
