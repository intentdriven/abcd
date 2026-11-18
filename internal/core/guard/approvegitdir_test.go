package guard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// gitDirRig is a repository's working tree holding a second repository with
// its own .git (nested), a bare one (bare.git), a plain subdirectory (sub), a
// submodule (mod), and a third repository outside the tree (outside).
func gitDirRig(t *testing.T) (approveRig, string) {
	t.Helper()
	r, repo := gitRig(t)
	repo.AddSubmodule("mod")
	for _, d := range []string{"nested", "sub", filepath.Join("nested", "deeper")} {
		if err := os.MkdirAll(filepath.Join(r.dir, d), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	gittest.NewRepoAt(t, filepath.Join(r.dir, "nested"))
	gittest.InitBare(t, filepath.Join(r.dir, "bare.git"))
	outside := t.TempDir()
	gittest.NewRepoAt(t, outside)
	return r, outside
}

// TestApproveRefusesAGitCommandAimedAtASecondGitDirectoryInTheTree: git
// takes its configuration from the git directory it runs against, so a git
// command aimed at a second git directory standing inside the working tree
// (by -C, --git-dir, GIT_DIR set on the command or exported before it,
// GIT_COMMON_DIR, --bare, a change of directory, or git init
// --separate-git-dir) reads a configuration the guard never read, and is
// refused, naming the directory. A plain subdirectory, a submodule (whose git
// file points into the tree's own git directory) and a target outside the
// tree are left alone; a target the line does not fix is refused. Watched
// failing before: every refused line here was approved.
func TestApproveRefusesAGitCommandAimedAtASecondGitDirectoryInTheTree(t *testing.T) {
	r, outside := gitDirRig(t)
	refused := []struct{ line, names string }{
		{`git -C nested log -1`, "nested"},
		{`git -C bare.git log -1`, "bare.git"},
		{`git -C nested/deeper log -1`, "nested"},
		{`git -C sub -C ../nested log -1`, "nested"},
		{`git --git-dir=nested/.git log -1`, "nested"},
		{`git --git-dir nested/.git log -1`, "nested"},
		{`git --git-dir=bare.git log -1`, "bare.git"},
		{`git --git-dir bare.git log -1`, "bare.git"},
		{`git -C sub --git-dir=../bare.git log -1`, "bare.git"},
		{`git --git-dir=` + filepath.Join(r.dir, "bare.git") + ` log -1`, "bare.git"},
		{`GIT_DIR=nested/.git git log -1`, "nested"},
		{`GIT_DIR=bare.git git log -1`, "bare.git"},
		{`env GIT_DIR=bare.git git log -1`, "bare.git"},
		{`export GIT_DIR=bare.git && git log -1`, "bare.git"},
		{`export GIT_DIR=nested/.git; git log -1`, "nested"},
		{`GIT_DIR=bare.git; export GIT_DIR; git log -1`, "bare.git"},
		{`GIT_COMMON_DIR=bare.git git log -1`, "bare.git"},
		{`git -C bare.git --bare log -1`, "bare.git"},
		{`cd nested && git log -1`, "nested"},
		{`cd bare.git && git log -1`, "bare.git"},
		{`git init --separate-git-dir bare.git`, "bare.git"},
		{`git init --separate-git-dir=nested/.git`, "nested"},
		{`git init --separate-git-dir sub/prog`, "prog"},
		{`git -C sub init --separate-git-dir prog`, "prog"},
	}
	for _, c := range refused {
		a := approveShell(clean(), r.files(), c.line)
		if a.Allow || a.EntryID != gitDirEntryID || !strings.Contains(a.Reason, c.names) {
			t.Errorf("%q: want a refusal under %s naming %s, got %+v", c.line, gitDirEntryID, c.names, a)
		}
	}
	unfixed := []string{
		`git -C "$D" log -1`,
		`git -C "$(pwd)/nested" log -1`,
		`git --git-dir="$D" log -1`,
		`git --git-dir "$D" log -1`,
		`GIT_DIR=$D git log -1`,
		`export GIT_DIR="$D"; git log -1`,
		`read -r GIT_DIR; git log -1`,
		`cd "$D" && git -C sub log -1`,
		`cd "$D" && git --git-dir=.git log -1`,
		`git init --separate-git-dir "$D"`,
	}
	for _, line := range unfixed {
		a := approveShell(clean(), r.files(), line)
		if a.Allow || a.EntryID != gitDirEntryID {
			t.Errorf("%q: want a refusal under %s, got %+v", line, gitDirEntryID, a)
		}
	}
	r.run(t, []approveCase{
		{`git log -1`, nil},
		{`cd sub && git log -1`, nil},
		{`git --git-dir=.git log -1`, nil},
		{`cd mod && git log -1`, nil},
		{`git --git-dir=mod/.git log -1`, nil},
		{`git --git-dir=.git/modules/mod log -1`, nil},
		{`git --git-dir=` + filepath.Join(outside, ".git") + ` log -1`, nil},
		{`GIT_DIR=` + filepath.Join(outside, ".git") + ` git log -1`, nil},
		{`git init --separate-git-dir ` + filepath.Join(outside, "prog") + ` ` + filepath.Join(outside, "w"), nil},
		{`echo GIT_DIR=bare.git`, nil},
	})
	// A git command with -C is judged by the approver's other rules as
	// before (git's writes there are not placed); this rule leaves these
	// alone.
	for _, line := range []string{
		`git -C . log -1`,
		`git -C sub log -1`,
		`git -C "" log -1`,
		`git -C mod log -1`,
		`git -C .git log -1`,
		`git -C ` + outside + ` log -1`,
	} {
		if a := approveShell(clean(), r.files(), line); a.EntryID == gitDirEntryID {
			t.Errorf("%q: refused under %s, which leaves it alone: %s", line, gitDirEntryID, a.Reason)
		}
	}
}
