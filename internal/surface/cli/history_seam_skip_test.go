package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestHistoryFormsThatSkipTheSeamCreateNothing pins iss-2611060810065301.
// The history chapter said every history verb reaches the store through one
// resolve seam that creates the store chain. Two forms resolve no repository
// and skip that seam: `history staged --all-repos` and `history discard
// --pending`. On a fresh home each leaves the home empty, while `history list`
// creates the chain; the chapter must name the two and say they create
// nothing.
func TestHistoryFormsThatSkipTheSeamCreateNothing(t *testing.T) {
	repo, _ := sessionEndRepo(t)

	// Run from outside any repository: both forms answer from anywhere.
	t.Chdir(t.TempDir())
	home := os.Getenv("HOME")

	runCLI(t, "history", "staged", "--all-repos")
	if entries, _ := os.ReadDir(home); len(entries) != 0 {
		t.Errorf("history staged --all-repos left %d entr(ies) in a fresh home, want none", len(entries))
	}
	if out, err := runCLIErr(t, "history", "discard", "--pending", "sess-none.raw", "--yes"); err == nil {
		t.Fatalf("discard --pending of a transcript that does not exist succeeded:\n%s", out)
	}
	if entries, _ := os.ReadDir(home); len(entries) != 0 {
		t.Errorf("history discard --pending left %d entr(ies) in a fresh home, want none", len(entries))
	}

	// The contrast the chapter draws: a verb through the seam creates the chain.
	t.Chdir(repo)
	runCLI(t, "history", "list")
	if entries, _ := os.ReadDir(home); len(entries) == 0 {
		t.Errorf("history list created nothing in a fresh home: the seam's store chain is missing")
	}

	chapter, err := os.ReadFile(filepath.Join(repoRootFromTest(t), ".abcd", "development", "brief", "04-surfaces", "11-history.md"))
	if err != nil {
		t.Fatal(err)
	}
	prose := strings.Join(strings.Fields(string(chapter)), " ")
	for _, want := range []string{
		"`abcd history staged --all-repos` and `abcd history discard --pending` skip the seam",
		"create nothing",
	} {
		if !strings.Contains(prose, want) {
			t.Errorf("the history chapter does not state %q: it still says every history verb reaches the store through the seam", want)
		}
	}
}
