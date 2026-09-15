package issueschema_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/issueschema"
)

// The ledger's status directories are one list, and the day a sibling directory
// joins the ledger tree is the day a fourth spelling of that list becomes a
// silent divergence: a writer that provisions three folders, a reader that scans
// three, and a gate that scopes to three all have to mean the SAME three, or a
// record lands somewhere one of them never looks.
//
// The list was spelled three times in Go — lint's issueStatusDirs and two
// literals in capture's allocator — plus once in the shell gate, which cannot
// import Go. issueschema is the canonical home for the same reason it holds the
// property allow-list: it is the one leaf both core/capture and core/lint already
// read the ledger's schema from.
func TestStatusDirsAreTheOneCanonicalList(t *testing.T) {
	want := []string{"open", "resolved", "wontfix"}
	if !slices.Equal(issueschema.StatusDirs, want) {
		t.Fatalf("StatusDirs = %v, want %v", issueschema.StatusDirs, want)
	}

	// The sibling record families that join the ledger tree are NOT status
	// directories: their status signal is the presence of a keyed disposition, not
	// folder membership, so a gate scoped to this list must keep ignoring them.
	for _, sibling := range []string{"readings", "dispositions"} {
		if slices.Contains(issueschema.StatusDirs, sibling) {
			t.Errorf("StatusDirs contains %q; the sibling record families are not status directories", sibling)
		}
	}

	// No second spelling anywhere in the Go tree. A copy is not a style
	// complaint — it is the divergence above, waiting for one side to be edited.
	root := filepath.Join("..", "..", "..")
	canonical := filepath.FromSlash("internal/core/issueschema/issueschema.go")
	var offenders []string
	err := filepath.WalkDir(filepath.Join(root, "internal"), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Test files are skipped: a fixture legitimately writes the folder names
		// it is building on disk, and that is data, not a second declaration.
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		if rel == canonical {
			return nil
		}
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.Contains(string(b), `"open", "resolved", "wontfix"`) {
			offenders = append(offenders, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk internal/: %v", err)
	}
	if len(offenders) > 0 {
		t.Errorf("the status-directory list is spelled again in %s;\n"+
			"there is one canonical list (issueschema.StatusDirs) and every Go consumer reads it",
			strings.Join(offenders, ", "))
	}
}
