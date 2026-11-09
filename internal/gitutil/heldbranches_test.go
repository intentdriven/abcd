package gitutil_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// stoppedRebase builds main and two stacked branches, lane/base under
// lane/top, then rebases lane/top onto a main that conflicts with lane/base's
// commit, with --update-refs, in the main checkout itself: git stops on the
// conflict with the checkout detached, head-name naming lane/top and the
// update-refs list naming lane/base.
func stoppedRebase(t *testing.T) *gittest.Repo {
	t.Helper()
	r := gittest.NewRepo(t)
	r.Write("f", "a\n")
	r.Commit("a")
	r.Git("switch", "-q", "-c", "lane/base")
	r.Write("f", "b\n")
	r.Commit("b")
	r.Git("switch", "-q", "-c", "lane/top")
	r.Write("g", "c\n")
	r.Commit("c")
	r.Git("switch", "-q", "main")
	r.Write("f", "z\n")
	r.Commit("z")
	r.Git("switch", "-q", "lane/top")
	cmd := exec.Command("git", "-C", r.Root(), "-c", "user.email=fixture@example.invalid", "-c", "user.name=Fixture",
		"-c", "commit.gpgsign=false", "rebase", "--update-refs", "main")
	cmd.Env = r.Env()
	if out, err := cmd.CombinedOutput(); err == nil {
		t.Fatalf("the rebase did not stop on its conflict:\n%s", out)
	}
	if got := r.Git("worktree", "list", "--porcelain"); !slices.Contains(strings.Split(got, "\n"), "detached") {
		t.Fatalf("git does not list the stopped checkout detached:\n%s", got)
	}
	return r
}

// A rebase git itself stopped in the main checkout holds the branch it started
// from (head-name) and each branch its --update-refs list will rewrite, though
// git lists the checkout detached; both read from the common dir.
func TestHeldBranchesReadsARebaseGitStoppedInTheMainCheckout(t *testing.T) {
	r := stoppedRebase(t)
	common, err := gitutil.CommonDir(r.Root())
	if err != nil {
		t.Fatal(err)
	}
	h := gitutil.HeldBranches(common)
	if h.Unreadable != "" {
		t.Fatalf("Unreadable = %q, want every state file read", h.Unreadable)
	}
	if got := h.Branches["lane/top"]; got != common {
		t.Errorf("Branches[lane/top] = %q, want the common dir %q (all: %v)", got, common, h.Branches)
	}
	if got := h.Rewrites["lane/base"]; got != common {
		t.Errorf("Rewrites[lane/base] = %q, want the common dir %q (all: %v)", got, common, h.Rewrites)
	}
	if _, ok := h.Branches["lane/base"]; ok {
		t.Errorf("lane/base is counted started from, though only the update-refs list names it: %v", h.Branches)
	}
	if _, ok := h.Branches["main"]; ok {
		t.Errorf("main is held, though no rebase or bisect names it: %+v", h)
	}
	if _, ok := h.Rewrites["main"]; ok {
		t.Errorf("main is to be rewritten, though no update-refs list names it: %+v", h)
	}
}

// An update-refs list git would call invalid (a record cut short, or an
// object name missing) names no branch for certain, so it is Unreadable, as a
// symlinked head-name is: a caller about to act counts every branch held.
func TestHeldBranchesFailsClosedOnAStateFileItCannotRead(t *testing.T) {
	r := stoppedRebase(t)
	common, err := gitutil.CommonDir(r.Root())
	if err != nil {
		t.Fatal(err)
	}
	list := filepath.Join(common, "rebase-merge", "update-refs")
	for name, body := range map[string]string{
		"cut short":      "refs/heads/lane/base\n0000000000000000000000000000000000000000\n",
		"no object name": "refs/heads/lane/base\nnot-an-object-name\n0000000000000000000000000000000000000000\n",
	} {
		if err := os.WriteFile(list, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if h := gitutil.HeldBranches(common); h.Unreadable != list {
			t.Errorf("%s: Unreadable = %q, want %q", name, h.Unreadable, list)
		}
	}
	if err := os.Remove(list); err != nil {
		t.Fatal(err)
	}
	head := filepath.Join(common, "rebase-merge", "head-name")
	if err := os.Remove(head); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(t.TempDir(), "elsewhere"), head); err != nil {
		t.Fatal(err)
	}
	if h := gitutil.HeldBranches(common); h.Unreadable != head {
		t.Errorf("a symlinked head-name: Unreadable = %q, want %q", h.Unreadable, head)
	}
}

// A holder's git dir is not where the person works: a linked worktree's is
// <common>/worktrees/<id>, and git picks a new id when the folder name is
// taken (a second "look" becomes look1). Held.Checkout names the checkout itself,
// through the gitdir file git keeps there, for a refusal to point at.
func TestHeldCheckoutNamesTheCheckoutNotTheAdminDir(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("f", "a\n")
	r.Commit("a")
	first := filepath.Join(t.TempDir(), "look")
	second := filepath.Join(t.TempDir(), "look")
	r.Git("worktree", "add", "-q", "--detach", first)
	r.Git("worktree", "add", "-q", "--detach", second)
	common, err := gitutil.CommonDir(r.Root())
	if err != nil {
		t.Fatal(err)
	}
	h := gitutil.HeldBranches(common)
	admin := strings.TrimSpace(r.Git("-C", second, "rev-parse", "--path-format=absolute", "--git-dir"))
	if filepath.Base(admin) != "look1" {
		t.Fatalf("premise: the second worktree's admin dir is %s, want the id look1", admin)
	}
	wts, err := gitutil.ListWorktrees(r.Root(), gitutil.MaxWorktreeListing)
	if err != nil {
		t.Fatal(err)
	}
	same := func(a, b string) bool {
		ra, _ := filepath.EvalSymlinks(a)
		rb, _ := filepath.EvalSymlinks(b)
		return ra == rb
	}
	if got := h.Checkout(admin, wts); !same(got, second) {
		t.Errorf("Checkout(%s) = %q, want the checkout %q", admin, got, second)
	}
	if got := h.Checkout(common, wts); !same(got, r.Root()) {
		t.Errorf("Checkout(common) = %q, want the main checkout %q", got, r.Root())
	}
	// git 2.48's worktree.useRelativePaths writes the gitdir file relative
	// to the admin dir.
	rel, err := filepath.Rel(admin, filepath.Join(second, ".git"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(admin, "gitdir"), []byte(rel+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := h.Checkout(admin, nil); !same(got, second) {
		t.Errorf("Checkout(%s) with a relative gitdir = %q, want %q", admin, got, second)
	}
	// With no gitdir file to read, the admin dir is all there is to name.
	if err := os.Remove(filepath.Join(admin, "gitdir")); err != nil {
		t.Fatal(err)
	}
	if got := h.Checkout(admin, wts); got != admin {
		t.Errorf("Checkout(%s) with no gitdir file = %q, want the admin dir itself", admin, got)
	}
}

// The gitdir file is text anyone who can write the admin dir controls, and
// Checkout's answer is printed in a refusal: text holding a control character
// or a line break names no checkout a person could open, so the admin dir is
// named instead, and no such rune reaches the terminal.
func TestHeldCheckoutNamesTheAdminDirForAGitdirHoldingAControlCharacter(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("f", "a\n")
	r.Commit("a")
	wt := filepath.Join(t.TempDir(), "look")
	r.Git("worktree", "add", "-q", "--detach", wt)
	common, err := gitutil.CommonDir(r.Root())
	if err != nil {
		t.Fatal(err)
	}
	h := gitutil.HeldBranches(common)
	admin := filepath.Join(common, "worktrees", "look")
	for _, text := range []string{
		wt + "\x1b[2J/.git\n",
		filepath.Join(wt, "a\nforged line", ".git") + "\n",
		filepath.Join(wt, "a\rb", ".git") + "\n",
		filepath.Join(wt, "a\u202eb", ".git") + "\n",
	} {
		if err := os.WriteFile(filepath.Join(admin, "gitdir"), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := h.Checkout(admin, nil); got != admin {
			t.Errorf("Checkout with gitdir %q = %q, want the admin dir %q", text, got, admin)
		}
	}
}

// git writes a relative gitdir (worktree.useRelativePaths) against the admin
// dir's real path, and reads it the same way, so a common dir reached through
// a symlink is resolved before the join: joined to the path as given, the
// climb out of the admin dir lands beside the symlink, not the checkout.
func TestHeldCheckoutJoinsARelativeGitdirToTheRealAdminDir(t *testing.T) {
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	real := filepath.Join(base, "a", "b", "real")
	gd := filepath.Join(base, "gd")
	for _, d := range []string{real, gd} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	link := filepath.Join(gd, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	r := gittest.NewRepoAt(t, real)
	r.Write("f", "a\n")
	r.Commit("a")
	wt := filepath.Join(base, "wt")
	r.Git("-C", link, "worktree", "add", "-q", "--detach", wt)
	h := gitutil.HeldBranches(filepath.Join(link, ".git"))
	admin := filepath.Join(link, ".git", "worktrees", "wt")
	realAdmin := filepath.Join(real, ".git", "worktrees", "wt")
	rel, err := filepath.Rel(realAdmin, filepath.Join(wt, ".git"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(admin, "gitdir"), []byte(rel+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := h.Checkout(admin, nil); got != wt {
		t.Errorf("Checkout(%s) with gitdir %q = %q, want the checkout %q", admin, rel, got, wt)
	}
}
