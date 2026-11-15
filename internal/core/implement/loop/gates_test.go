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

// TestDriveGateNamesTheWorkflowFolderAndFoldsCase: the forge's workflow folder
// is a gate, and every gate, the configured hooks path among them, is named
// whatever the case of each part, on any filesystem; an ordinary path, or a
// name that only starts with a gate's letters, is none.
func TestDriveGateNamesTheWorkflowFolderAndFoldsCase(t *testing.T) {
	for _, c := range []struct {
		path, hooks string
		gate        bool
	}{
		{".github/workflows/ci.yml", "", true},
		{".github", "", true},
		{".github/", "", true},
		{"Scripts/x.sh", "", true},
		{"MAKEFILE", "", true},
		{".GitHooks/pre-push", "", true},
		{"Tools/Hooks/x", "tools/hooks", true},
		{"tools/HOOKS", "tools/hooks", true},
		{"internal/x.go", "", false},
		{"scriptsx/a", "", false},
		{".githubx/workflows/ci.yml", "", false},
		{"Makefile.bak", "", false},
		{"makefile/x", "", false},
		{"docs/.github/x.yml", "", false},
		{"tools/hooksx/a", "tools/hooks", false},
		{"tools", "tools/hooks", false},
		{"Make\ufb01le", "", true},
		{"GNUmake\ufb01le", "", true},
		{"MAKE\ufb01LE", "", true},
		{"\u017fcripts/x.sh", "", true},
		{"hoo\u212as/pre-push", "", true},
		{"tools/cla\u00df-hooks/x", "tools/class-hooks", true},
		{"tools/CLA\u1e9e-HOOKS/x", "tools/class-hooks", true},
		{"tools/\ufb01xups/a", "tools/fixups", true},
		{"tools/class-hooks/x", "tools/cla\u00df-hooks", true},
		{"Make\ufb02le", "", false},
		{"tools/h\u00e9ks/x", "tools/he\u0301ks", true},
		{"tools/he\u0301ks/x", "tools/h\u00e9ks", true},
		{"tools/H\u00c9KS/x", "tools/he\u0301ks", true},
		{"scr\u00efpts/x.sh", "", false},
		{"Makef\u00edle", "", false},
		{"tools/h\u00e9ks/x", "tools/heks", false},
		{".abcd/rules.json", "", true},
		{".ABCD/Guard.json", "", true},
		{".abcd/config.json", "", true},
		{".abcd/config/patterns", "", true},
		{".abcd/.work.local/approvals/abc", "", true},
		{".abcd/.WORK.local/preflight-receipts/abc", "", true},
		{".abcd/work/issues/open/a.md", "", false},
		{".abcd/rules.json.bak", "", false},
		{".abcd/configx/a", "", false},
		{"", "", false},
	} {
		if got := DriveGate(c.path, c.hooks); got != c.gate {
			t.Errorf("DriveGate(%q, %q) = %v, want %v", c.path, c.hooks, got, c.gate)
		}
	}
}

// TestAHooksPathAtTheTopMakesEveryPathAGate: a configured hooks path that is
// the tree's own top, however written, makes every path a gate, as a hooks
// directory counts with all below it; an empty path still names nothing.
func TestAHooksPathAtTheTopMakesEveryPathAGate(t *testing.T) {
	for _, hooks := range []string{".", "./", "a/.."} {
		for _, path := range []string{"pre-push", "internal/x.go"} {
			if !DriveGate(path, hooks) {
				t.Errorf("DriveGate(%q, %q) is false", path, hooks)
			}
		}
		if DriveGate("", hooks) {
			t.Errorf("an empty path is a gate under hooks path %q", hooks)
		}
	}
	repo := briefRepo(t, agentsMarked)
	for _, value := range []string{".", "./"} {
		repo.Git("config", "core.hooksPath", value)
		p, err := DriveHooksPath(repo.Root())
		if err != nil || !DriveGate("internal/x.go", p) {
			t.Fatalf("core.hooksPath %q reads as %q (%v), which leaves internal/x.go no gate", value, p, err)
		}
	}
}
