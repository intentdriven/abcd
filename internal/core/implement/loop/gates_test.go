package loop

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gittest"
)

// TestAHooksPathThatCannotBeReadIsNotNone: the hooks path a drive's gate
// check reads is none only where git says the key is not set; a read git
// fails is an error naming it, never "no in-tree hooks directory". Watched
// failing: the failed read answered none.
func TestAHooksPathThatCannotBeReadIsNotNone(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	if p, err := DriveHooksPath(repo.Root()); p != "" || err != nil {
		t.Fatalf("no core.hooksPath is none: %q %v", p, err)
	}
	repo.Git("config", "core.hooksPath", ".githooks")
	if p, err := DriveHooksPath(repo.Root()); p != ".githooks" || err != nil {
		t.Fatalf("a relative core.hooksPath is named: %q %v", p, err)
	}
	standInGit(t, `*"--get-all core.hooksPath"*) echo "fatal: bad config line 3 in file .git/config" >&2; exit 128 ;;`)
	p, err := DriveHooksPath(repo.Root())
	if err == nil || p != "" || !strings.Contains(err.Error(), "git config --type=path --get-all core.hooksPath") {
		t.Fatalf("a read git failed is an error naming it: %q %v", p, err)
	}
}

// TestAnAbsoluteHooksPathIsPlacedByWhereItReallyIs: a checkout reached
// through a linked parent whose core.hooksPath is written in the resolved
// form names a directory inside it, and so does one reached directly whose
// value is written through the link; a value outside it is none. Watched
// failing: the value in the other form read as outside the checkout, so its
// hooks were no gate.
func TestAnAbsoluteHooksPathIsPlacedByWhereItReallyIs(t *testing.T) {
	real := filepath.Join(fsutil.RealExistingPath(t.TempDir()), "real")
	repo := filepath.Join(real, "repo")
	mkdirs(t, repo)
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	git := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env = gittest.Env(t)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q")
	viaLink := filepath.Join(link, "repo")
	for _, c := range []struct{ dir, value, want string }{
		{viaLink, filepath.Join(repo, "tools", "hooks"), "tools/hooks"},
		{repo, filepath.Join(viaLink, "tools", "hooks"), "tools/hooks"},
		{viaLink, filepath.Join(t.TempDir(), "hooks"), ""},
	} {
		git("config", "core.hooksPath", c.value)
		if p, err := DriveHooksPath(c.dir); p != c.want || err != nil {
			t.Fatalf("%s read from %s: %q %v, want %q", c.value, c.dir, p, err, c.want)
		}
	}
}
