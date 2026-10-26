package gitutil_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

func realOf(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// TestCommonDirIsAbsoluteFromEveryWorkingTree: the main working tree's answer
// (".git", relative) and a linked worktree's (absolute) name the same
// directory, absolute either way, so the store's proof of belonging and the
// peers reader compare them as paths.
func TestCommonDirIsAbsoluteFromEveryWorkingTree(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Write("a.txt", "a\n")
	r.Commit("one")
	wt := filepath.Join(t.TempDir(), "wt")
	r.Git("worktree", "add", "-q", "--detach", wt)

	main, err := gitutil.CommonDir(r.Root())
	if err != nil {
		t.Fatal(err)
	}
	linked, err := gitutil.CommonDir(wt)
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(main) || !filepath.IsAbs(linked) {
		t.Fatalf("want absolute answers: %q, %q", main, linked)
	}
	if realOf(t, main) != realOf(t, linked) || realOf(t, main) != realOf(t, filepath.Join(r.Root(), ".git")) {
		t.Fatalf("one common dir from both trees: %q, %q", main, linked)
	}
}

// TestCommonDirRefusesAnythingButOneLine: a git older than 2.31 echoes an
// option it does not know and answers on a second line, so anything but one
// line is refused rather than compared; an empty answer is refused too.
func TestCommonDirRefusesAnythingButOneLine(t *testing.T) {
	r := gittest.NewRepo(t)
	for name, out := range map[string]string{
		"two lines": "--path-format=absolute\n/srv/repo/.git\n",
		"a CR":      "/srv/repo/.git\r/elsewhere\n",
		"empty":     "\n",
	} {
		t.Run(name, func(t *testing.T) {
			gittest.ShimGit(t, []string{"--git-common-dir"}, out, 0)
			got, err := gitutil.CommonDir(r.Root())
			if err == nil {
				t.Fatalf("want a refusal, got %q", got)
			}
			if !strings.Contains(err.Error(), "no single common dir") {
				t.Fatalf("the refusal says why: %v", err)
			}
		})
	}
}

// TestCommonDirOutsideARepositoryFails: git's own refusal is returned.
func TestCommonDirOutsideARepositoryFails(t *testing.T) {
	gittest.Env(t)
	if got, err := gitutil.CommonDir(t.TempDir()); err == nil {
		t.Fatalf("want git's refusal outside a repository, got %q", got)
	}
}
