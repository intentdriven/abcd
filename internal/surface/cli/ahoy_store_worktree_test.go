package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/gittest"
)

// TestBareAhoyNamesAnUnlinkedStoreWorktree (iss-2610050728100598): the pages
// that print the rename repair promise that `abcd ahoy` names any store
// worktree still unlinked afterwards, with the command that repairs that one.
// The gap is report-only, so the text board is where a person meets it: one
// `worktree:` line per such worktree, naming it home-relative and carrying its
// own repair, quoted so a name holding a space or an apostrophe pastes as it
// is.
func TestBareAhoyNamesAnUnlinkedStoreWorktree(t *testing.T) {
	hermeticEnv(t)
	home := os.Getenv("HOME")
	repo := gittest.NewRepo(t)
	repo.Write("README.md", "fixture\n")
	repo.Commit("init")
	sha := repo.Git("rev-list", "--max-parents=0", "HEAD")

	// A worktree made elsewhere and then moved into the store behind git's back,
	// which leaves the repository's link naming the old place, as the rename does.
	made := filepath.Join(t.TempDir(), "lane")
	repo.Git("worktree", "add", "-b", "lane", made)
	store := abcdhome.Path(home, "worktrees", sha)
	if err := os.MkdirAll(store, abcdhome.DirMode); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(made, filepath.Join(store, "it's a lane")); err != nil {
		t.Fatal(err)
	}

	t.Chdir(repo.Root())
	out, err := runCLIErr(t, "ahoy")
	if err != nil {
		t.Fatalf("ahoy: %v\n%s", err, out)
	}
	rel := abcdhome.Rel("worktrees", sha, "it's a lane")
	var line string
	for _, l := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(l, "  worktree:    ") {
			line = l
		}
	}
	if line == "" {
		t.Fatalf("bare ahoy names no unlinked store worktree:\n%s", out)
	}
	for _, want := range []string{
		"  worktree:    unlinked — ~/" + rel + ": its repository records it at ",
		"`" + abcdhome.WorktreeRepairCommand(rel) + "`",
	} {
		if !strings.Contains(line, want) {
			t.Errorf("the worktree line does not carry %q:\n%s", want, line)
		}
	}
	if strings.Contains(line, home) {
		t.Errorf("the worktree line names the home directory in full rather than as ~/:\n%s", line)
	}
}
