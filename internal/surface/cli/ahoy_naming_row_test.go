package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ahoy_naming_row_test.go: the /abcd:ahoy row of 02-constraints/04-naming.md
// says which sub-verbs write, and it names every one that does
// (iss-2611040554013625).

// ahoyWritePaths are the ahoy sub-verbs the naming row calls its write paths,
// each as the row spells it and as the path below `abcd ahoy` that carries it.
var ahoyWritePaths = []struct {
	spelling string
	path     []string
}{
	{"`install`", []string{"install"}},
	{"`uninstall`", []string{"uninstall"}},
	{"`connect`", []string{"connect"}},
	{"`remote apply`", []string{"remote", "apply"}},
	{"`credential`", []string{"credential"}},
	{"`worktree add`", []string{"worktree", "add"}},
	{"`worktree prune`", []string{"worktree", "prune"}},
}

// TestNamingRowNamesAhoyWritePaths reads the row and fails for a write path it
// leaves out of its write-paths clause, or one the command tree no longer
// carries: `worktree add` writes the worktree, its start record and its local
// tier, and `worktree prune` writes the notes archive and removes worktrees.
func TestNamingRowNamesAhoyWritePaths(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRootFromTest(t), ".abcd", "development", "brief", "02-constraints", "04-naming.md"))
	if err != nil {
		t.Fatal(err)
	}
	var row string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "| `/abcd:ahoy` |") {
			row = line
			break
		}
	}
	if row == "" {
		t.Fatal("04-naming.md has no `/abcd:ahoy` row")
	}
	// The clause runs from the sentence's semicolon to "are the write paths":
	// a sub-verb named elsewhere in the row is not named as one that writes.
	end := strings.Index(row, "are the write paths")
	if end < 0 {
		t.Fatalf("the `/abcd:ahoy` row no longer says which sub-verbs are the write paths:\n%s", row)
	}
	start := strings.LastIndex(row[:end], ";")
	if start < 0 {
		t.Fatalf("the `/abcd:ahoy` row's write-paths clause has no opening semicolon:\n%s", row)
	}
	clause := row[start:end]

	ahoy, _, err := NewRootCommand().Find([]string{"ahoy"})
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range ahoyWritePaths {
		if !strings.Contains(clause, w.spelling) {
			t.Errorf("the `/abcd:ahoy` row's write paths leave out %s:\n%s", w.spelling, clause)
		}
		cmd, rest, err := ahoy.Find(w.path)
		if err != nil || len(rest) != 0 || cmd.Name() != w.path[len(w.path)-1] {
			t.Errorf("`abcd ahoy %s` is not in the command tree, so the row names a write path that does not exist", strings.Join(w.path, " "))
		}
	}
}
