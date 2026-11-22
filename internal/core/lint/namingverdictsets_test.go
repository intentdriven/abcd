package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNamingRegistersEveryClosedVerdictSet (iss-2610262030580466): the
// register's reserved vocabulary once held one unscoped "Review verdicts" row,
// {SHIP, NEEDS_WORK, MAJOR_RETHINK}, while the saved docs review returns
// PROMOTE, HOLD or INCONCLUSIVE and the implement loop's reviewers state words
// of their own, none of them registered. "Review verdict" so named two closed
// sets. The family-1 row must be scoped to the lifeboat review, and each other
// closed verdict set in use must have a row of its own.
func TestNamingRegistersEveryClosedVerdictSet(t *testing.T) {
	path := filepath.Join(repoRootFromPackage, ".abcd", "development", "brief", "02-constraints", "04-naming.md")
	naming, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	rows := map[string]string{}
	for _, line := range strings.Split(string(naming), "\n") {
		if !strings.HasPrefix(line, "| ") {
			continue
		}
		cells := strings.Split(line, " | ")
		if len(cells) < 3 {
			continue
		}
		rows[strings.TrimPrefix(cells[0], "| ")] = line
	}
	for term := range rows {
		if strings.HasPrefix(term, "Review verdicts") {
			t.Errorf("04-naming.md still registers an unscoped %q row; scope it to the lifeboat review (family 1)", term)
		}
	}
	want := []struct{ term, says string }{
		{"Lifeboat review verdicts ∈ `{SHIP, NEEDS_WORK, MAJOR_RETHINK}`", "family 1"},
		{"Docs review verdicts ∈ `{PROMOTE, HOLD, INCONCLUSIVE}`", "verificationResult"},
		{"`ruthless-reviewer` verdicts ∈ `{SHIP, FIX FIRST}`", "implement loop"},
		{"`security-reviewer` verdicts ∈ `{APPROVE, BLOCK, NEEDS-INPUT}`", "implement loop"},
	}
	for _, w := range want {
		row, ok := rows[w.term]
		if !ok {
			t.Errorf("04-naming.md registers no %q row", w.term)
			continue
		}
		if !strings.Contains(row, w.says) {
			t.Errorf("04-naming.md's %q row does not say %q: %q", w.term, w.says, row)
		}
	}
}
