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
	for _, b := range []string{"lane/top", "lane/base"} {
		if h.Holder(b) != common {
			t.Errorf("Holder(%s) = %q, want %q", b, h.Holder(b), common)
		}
	}
	if h.Holder("main") != "" {
		t.Errorf("main is held, though no rebase or bisect names it: %+v", h)
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
