package loop

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
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
	if p, err := DriveHooksPaths(repo.Root()); len(p) != 0 || err != nil {
		t.Fatalf("no core.hooksPath is none: %q %v", p, err)
	}
	repo.Git("config", "core.hooksPath", ".githooks")
	if p, err := DriveHooksPaths(repo.Root()); !slices.Equal(p, []string{".githooks"}) || err != nil {
		t.Fatalf("a relative core.hooksPath is named: %q %v", p, err)
	}
	standInGit(t, `*"--get-all core.hooksPath"*) echo "fatal: bad config line 3 in file .git/config" >&2; exit 128 ;;`)
	p, err := DriveHooksPaths(repo.Root())
	if err == nil || len(p) != 0 || !strings.Contains(err.Error(), "git config --type=path --get-all core.hooksPath") {
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
		if p, err := DriveHooksPaths(c.dir); !slices.Equal(p, hooksList(c.want)) || err != nil {
			t.Fatalf("%s read from %s: %q %v, want %q", c.value, c.dir, p, err, c.want)
		}
	}
}

// TestAHooksPathIsPlacedAsTheFilesystemComparesNames: an absolute value that
// spells the checkout in another case names the directory inside it where the
// filesystem folds case, and a relative value is taken from the checkout's top
// and cleaned, so one that leaves the tree and comes back names the directory
// it reaches; one that stays outside is none.
func TestAHooksPathIsPlacedAsTheFilesystemComparesNames(t *testing.T) {
	parent := fsutil.RealExistingPath(t.TempDir())
	repo := filepath.Join(parent, "Repo")
	mkdirs(t, repo)
	git := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env = gittest.Env(t)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q")
	variant := ""
	if fsutil.CaseFoldingFS() {
		variant = "tools/hooks"
	}
	for _, c := range []struct{ value, want string }{
		{filepath.Join(parent, "REPO", "tools", "hooks"), variant},
		{filepath.Join("..", "Repo", "tools", "h"), "tools/h"},
		{filepath.Join("tools", "..", "tools", "h"), "tools/h"},
		{filepath.Join("..", "elsewhere", "h"), ""},
	} {
		git("config", "core.hooksPath", c.value)
		if p, err := DriveHooksPaths(repo); !slices.Equal(p, hooksList(c.want)) || err != nil {
			t.Errorf("%s: %q %v, want %q", c.value, p, err, c.want)
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
		{"tools", "tools/hooks", true},
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
		p, err := DriveHooksPaths(repo.Root())
		if err != nil || !DriveGate("internal/x.go", p...) {
			t.Fatalf("core.hooksPath %q reads as %q (%v), which leaves internal/x.go no gate", value, p, err)
		}
	}
}

// hooksList is the paths DriveHooksPaths gives for one expected place, none for "".
func hooksList(want string) []string {
	if want == "" {
		return nil
	}
	return []string{want}
}

// TestALinkedHooksDirectoryNamesTheLinkAndWhereItLeads: a hooks path that is a
// link to another directory in the tree names both, the link once however the
// value reaches it, and each is a gate; a link that leads outside the tree
// names only itself.
func TestALinkedHooksDirectoryNamesTheLinkAndWhereItLeads(t *testing.T) {
	repo := briefRepo(t, agentsMarked)
	mkdirs(t, filepath.Join(repo.Root(), "tools", "real"))
	if err := os.Symlink(filepath.Join("tools", "real"), filepath.Join(repo.Root(), "h")); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"h", "./h", filepath.Join(repo.Root(), "h")} {
		repo.Git("config", "core.hooksPath", value)
		p, err := DriveHooksPaths(repo.Root())
		if err != nil || !slices.Equal(p, []string{"h", "tools/real"}) {
			t.Fatalf("core.hooksPath %q: %q %v", value, p, err)
		}
		for _, path := range []string{"h", "tools/real/pre-push"} {
			if !DriveGate(path, p...) {
				t.Errorf("%s is no gate under %q", path, p)
			}
		}
		if DriveGate("internal/x.go", p...) {
			t.Errorf("internal/x.go is a gate under %q", p)
		}
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(repo.Root(), "out")); err != nil {
		t.Fatal(err)
	}
	repo.Git("config", "core.hooksPath", "out")
	if p, err := DriveHooksPaths(repo.Root()); err != nil || !slices.Equal(p, []string{"out"}) {
		t.Fatalf("a link out of the tree: %q %v", p, err)
	}
}

// TestAHooksPathThroughLinksNamesEveryPlaceItPasses: the hooks value is walked
// one link at a time, so each link in a chain, a link above the hooks directory
// and the place a ".." after a link reaches are gates, and a change to any of
// them is handed back; a loop, or more links than the walk follows, ends with
// every link met a gate.
func TestAHooksPathThroughLinksNamesEveryPlaceItPasses(t *testing.T) {
	link := func(t *testing.T, root, target, name string) {
		t.Helper()
		if err := os.Symlink(filepath.FromSlash(target), filepath.Join(root, filepath.FromSlash(name))); err != nil {
			t.Fatal(err)
		}
	}
	hooksOf := func(t *testing.T, root, value string) []string {
		t.Helper()
		gittest.OpenRepo(t, root).Git("config", "core.hooksPath", value)
		p, err := DriveHooksPaths(root)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	holds := func(t *testing.T, p []string, want ...string) {
		t.Helper()
		for _, w := range want {
			if !slices.Contains(p, w) {
				t.Errorf("%q does not hold %q", p, w)
			}
		}
	}
	t.Run("a chain of links", func(t *testing.T) {
		root := briefRepo(t, agentsMarked).Root()
		mkdirs(t, filepath.Join(root, "b"))
		link(t, root, "b", "a")
		link(t, root, "a", "h")
		p := hooksOf(t, root, "h")
		holds(t, p, "h", "a", "b")
		for _, path := range []string{"h", "a", "b/pre-push"} {
			if !DriveGate(path, p...) {
				t.Errorf("%s is no gate under %q", path, p)
			}
		}
		if DriveGate("internal/x.go", p...) {
			t.Errorf("internal/x.go is a gate under %q", p)
		}
	})
	t.Run("a link above the hooks directory", func(t *testing.T) {
		root := briefRepo(t, agentsMarked).Root()
		mkdirs(t, filepath.Join(root, "real-tools", "hooks"))
		link(t, root, "real-tools", "tools")
		p := hooksOf(t, root, "tools/hooks")
		holds(t, p, "tools/hooks", "tools", "real-tools/hooks")
		for _, path := range []string{"tools", "real-tools", "real-tools/hooks/pre-push"} {
			if !DriveGate(path, p...) {
				t.Errorf("%s is no gate under %q", path, p)
			}
		}
		if !DriveGate("tools", "tools/hooks") || DriveGate("toolsx", "tools/hooks") || DriveGate("tools/other", "tools/hooks") {
			t.Error("a path above a hooks path is a gate, and no other is")
		}
	})
	t.Run("a .. after a link", func(t *testing.T) {
		root := briefRepo(t, agentsMarked).Root()
		mkdirs(t, filepath.Join(root, "sub", "deep"))
		link(t, root, "sub/deep", "x")
		p := hooksOf(t, root, "x/../h")
		holds(t, p, "x", "sub/h")
		if !DriveGate("sub/h/pre-push", p...) {
			t.Errorf("sub/h/pre-push is no gate under %q", p)
		}
	})
	t.Run("a loop", func(t *testing.T) {
		root := briefRepo(t, agentsMarked).Root()
		link(t, root, "g", "h")
		link(t, root, "h", "g")
		p := hooksOf(t, root, "h")
		holds(t, p, "h", "g")
	})
	t.Run("more links than the walk follows", func(t *testing.T) {
		root := briefRepo(t, agentsMarked).Root()
		mkdirs(t, filepath.Join(root, "real"))
		const n = maxHooksLinks + 5
		for i := 0; i < n; i++ {
			next := "real"
			if i+1 < n {
				next = fmt.Sprintf("l%d", i+1)
			}
			link(t, root, next, fmt.Sprintf("l%d", i))
		}
		p := hooksOf(t, root, "l0")
		holds(t, p, "l0", fmt.Sprintf("l%d", maxHooksLinks))
		// The walk stops at its bound; the as-resolved placement may still
		// reach the end of the chain, which only adds a gate.
		if slices.Contains(p, fmt.Sprintf("l%d", maxHooksLinks+1)) {
			t.Errorf("the walk went past its bound: %q", p)
		}
		if !DriveGate(fmt.Sprintf("l%d", maxHooksLinks), p...) {
			t.Errorf("the last link met is no gate under %q", p)
		}
	})
}
