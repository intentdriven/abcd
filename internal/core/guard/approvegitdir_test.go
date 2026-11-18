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

// TestApproveJudgesTheDirectoryGitRunsInAfterAnUnfixedCdAWriteOrAWrapper:
// git finds its repository from the directory it runs in, so a git command
// is refused under git-dir-not-own when that directory is one the line does
// not fix (after a cd the line leaves unknown), when an earlier command of
// the line writes and the directory or the git directory named stands inside
// the working tree away from its root and its own git directories (the write
// may have made a repository there), and when a wrapper's change of directory
// (env -C or --chdir, sudo -D or --chdir) moves git to a second git directory
// or to a directory the line does not fix. A launch directory the reading
// knows, with no cd, a write followed by git at the root, and a wrapper's
// change to a plain subdirectory are left alone. Watched failing before:
// each refused line here was approved, or refused under path-not-fixed for
// git's -C alone.
func TestApproveJudgesTheDirectoryGitRunsInAfterAnUnfixedCdAWriteOrAWrapper(t *testing.T) {
	r, outside := gitDirRig(t)
	// out is a link in the tree to the repository outside it, which a write
	// earlier in a line may point elsewhere.
	if err := os.Symlink(outside, filepath.Join(r.dir, "out")); err != nil {
		t.Fatal(err)
	}
	refused := []string{
		// The directory git runs in is unknown.
		`cd nested; git log -1`,
		`cd "$D" && git log -1`,
		`cd "$(echo nested)" && git status`,
		// An earlier command of the line writes.
		`cp -R bare.git sub/copy && git -C sub/copy log -1`,
		`cp -R bare.git sub/copy && cd sub/copy && git log -1`,
		`mkdir -p sub/x && git -C sub/x log -1`,
		`touch prog && git --git-dir=out/.git log -1`,
		`touch prog && GIT_DIR=out/.git git log -1`,
		// A wrapper changes the directory.
		`env -C nested git log -1`,
		`env -Cnested git log -1`,
		`env --chdir=nested git log -1`,
		`env --chdir nested git log -1`,
		`env -i -C bare.git git log -1`,
		`env -C sub git -C ../nested log -1`,
		`env -C sub env -C ../bare.git git log -1`,
		`env -u X -C nested git log -1`,
		`timeout 5 env --chd=nested git log -1`,
		`env --unset=$(echo X) -C nested git log -1`,
		`sudo -D nested git log -1`,
		`sudo --chdir=bare.git git log -1`,
		`env -C "$D" git log -1`,
		`sudo -D "$D" git log -1`,
		`env -C "" git log -1`,
	}
	for _, line := range refused {
		a := approveShell(clean(), r.files(), line)
		if a.Allow || a.EntryID != gitDirEntryID {
			t.Errorf("%q: want a refusal under %s, got %+v", line, gitDirEntryID, a)
		}
	}
	r.run(t, []approveCase{
		{`git log -1`, nil},
		{`git status`, nil},
		{`touch prog && git log -1`, nil},
		{`touch prog && git --git-dir=.git log -1`, nil},
		{`env -C sub git log -1`, nil},
		{`env --chdir=sub git log -1`, nil},
		{`env -C mod git log -1`, nil},
	})
	// Left to the approver's other rules, as git's own -C is.
	for _, line := range []string{
		`git --git-dir=out/.git log -1`,
		`env -C ` + outside + ` git log -1`,
		`sudo -D sub git log -1`,
		`sudo -u prog git log -1`,
	} {
		if a := approveShell(clean(), r.files(), line); a.EntryID == gitDirEntryID {
			t.Errorf("%q: refused under %s, which leaves it alone: %s", line, gitDirEntryID, a.Reason)
		}
	}
}
