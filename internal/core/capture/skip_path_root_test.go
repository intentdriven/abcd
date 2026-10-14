package capture

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestASkippedRecordIsRelativeUnderARootTheMaskChanges: a skipped record's
// path and the reader's refusal are made relative to the checkout before they
// are masked, so a checkout whose path holds a rune the mask replaces (a
// zero-width joiner, a tab) still yields a repo-relative path and an error
// that carries no absolute path, on `capture list --json` and `capture status
// --json` alike (iss-81's rule). Masking first left the strip unmatched: the
// path came out as ../../dev?work/... and the error carried the root whole.
func TestASkippedRecordIsRelativeUnderARootTheMaskChanges(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "dev‍work\tbox", "repo")
	ir := filepath.Join(repo, LedgerRelPath)
	if err := os.MkdirAll(filepath.Join(ir, "open"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ir, "open", "iss-7-a.md"), []byte("---\nschema_version: 2\nid: \"iss-7\"\n---\n\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A directory where a record should be is refused by the guarded read,
	// whose error quotes the path it read.
	if err := os.Mkdir(filepath.Join(ir, "open", "iss-8-b.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{
		filepath.ToSlash(filepath.Join(LedgerRelPath, "open", "iss-7-a.md")): true,
		filepath.ToSlash(filepath.Join(LedgerRelPath, "open", "iss-8-b.md")): true,
	}

	list, err := List(ListRequest{RepoRoot: repo, IssuesRoot: ir})
	if err != nil {
		t.Fatal(err)
	}
	st, err := Status(StatusRequest{RepoRoot: repo, IssuesRoot: ir})
	if err != nil {
		t.Fatal(err)
	}
	for what, skipped := range map[string][]SkipRecord{"capture list": list.Skipped, "capture status": st.Skipped} {
		if len(skipped) != 2 {
			t.Fatalf("%s: both broken records are skipped: %+v", what, skipped)
		}
		for _, sk := range skipped {
			if !want[filepath.ToSlash(sk.Path)] {
				t.Errorf("%s: the skip's path is repo-relative: %q", what, sk.Path)
			}
			b, err := json.Marshal(sk)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(b), "work box") || strings.Contains(string(b), "work?box") {
				t.Errorf("%s: the skip carries no absolute path: %s", what, b)
			}
			if strings.ContainsAny(string(b), "‍\t") {
				t.Errorf("%s: a raw rune reaches the skip: %s", what, b)
			}
		}
		if !strings.Contains(skipped[0].Error+skipped[1].Error, "iss-8-b.md") {
			t.Errorf("%s: the guarded read's refusal names the record it read: %+v", what, skipped)
		}
	}
}
