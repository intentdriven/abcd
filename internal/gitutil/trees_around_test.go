//go:build unix

package gitutil_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestEveryWorkingTreeAboveADirectoryIsNamed pins WorkingTreesAround: from a
// subdirectory it names the tree the directory is in and every tree that
// holds that one, nearest first, and it reaches them through a link to the
// subdirectory as well; the account's own home is left out, and a directory
// in no working tree names none.
func TestEveryWorkingTreeAboveADirectoryIsNamed(t *testing.T) {
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	outer := filepath.Join(base, "outer")
	inner := filepath.Join(outer, "vendor", "inner")
	sub := filepath.Join(inner, "pkg", "deep")
	plain := filepath.Join(base, "plain")
	for _, d := range []string{filepath.Join(outer, ".git"), sub, plain} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	// A submodule or linked worktree leaves a .git file, not a directory.
	if err := os.WriteFile(filepath.Join(inner, ".git"), []byte("gitdir: elsewhere\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "into-sub")
	if err := os.Symlink(sub, link); err != nil {
		t.Fatal(err)
	}

	accountHomeIs(t, plain, nil)
	want := []string{inner, outer}
	if got := gitutil.WorkingTreesAround(sub); !slices.Equal(got, want) {
		t.Errorf("WorkingTreesAround(subdirectory) = %q, want %q", got, want)
	}
	if got := gitutil.WorkingTreesAround(link); !slices.Equal(got, want) {
		t.Errorf("WorkingTreesAround(link to the subdirectory) = %q, want %q", got, want)
	}
	if got := gitutil.WorkingTreesAround(plain); len(got) != 0 {
		t.Errorf("WorkingTreesAround(a directory in no working tree) = %q, want none", got)
	}
	if got := gitutil.WorkingTreesAround(""); len(got) != 0 {
		t.Errorf("WorkingTreesAround(\"\") = %q, want none", got)
	}

	accountHomeIs(t, outer, nil)
	if got := gitutil.WorkingTreesAround(sub); !slices.Equal(got, []string{inner}) {
		t.Errorf("WorkingTreesAround with the outer tree as the account home = %q, want only %q", got, inner)
	}
}

// TestALinkedWorktreesMainCheckoutIsNamed: from inside a linked worktree,
// WorkingTreesAround names the worktree and the main checkout its .git file
// leads to (the gitdir it names, then that gitdir's commondir), read from
// those two files alone. A submodule's git directory carries no commondir
// and adds nothing, and a main checkout that is the account's own home is
// left out as any other tree is.
func TestALinkedWorktreesMainCheckoutIsNamed(t *testing.T) {
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	main := filepath.Join(base, "main")
	admin := filepath.Join(main, ".git", "worktrees", "wt")
	wt := filepath.Join(base, "elsewhere", "wt")
	sub := filepath.Join(wt, "pkg")
	module := filepath.Join(main, ".git", "modules", "vendor")
	submodule := filepath.Join(base, "detached-submodule")
	for _, d := range []string{admin, sub, module, submodule} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	write := func(p, s string) {
		t.Helper()
		if err := os.WriteFile(p, []byte(s), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(wt, ".git"), "gitdir: "+admin+"\n")
	write(filepath.Join(admin, "commondir"), "../..\n")
	write(filepath.Join(admin, "gitdir"), filepath.Join(wt, ".git")+"\n")
	write(filepath.Join(submodule, ".git"), "gitdir: "+module+"\n")

	accountHomeIs(t, filepath.Join(base, "nobody"), nil)
	if got, want := gitutil.WorkingTreesAround(sub), []string{wt, main}; !slices.Equal(got, want) {
		t.Errorf("WorkingTreesAround(inside a linked worktree) = %q, want %q", got, want)
	}
	if got, want := gitutil.WorkingTreesAround(submodule), []string{submodule}; !slices.Equal(got, want) {
		t.Errorf("WorkingTreesAround(a submodule's tree) = %q, want %q", got, want)
	}
	// A relative gitdir is read against the worktree.
	rel, err := filepath.Rel(wt, admin)
	if err != nil {
		t.Fatal(err)
	}
	write(filepath.Join(wt, ".git"), "gitdir: "+rel+"\n")
	if got, want := gitutil.WorkingTreesAround(sub), []string{wt, main}; !slices.Equal(got, want) {
		t.Errorf("WorkingTreesAround(a relative gitdir) = %q, want %q", got, want)
	}

	accountHomeIs(t, main, nil)
	if got, want := gitutil.WorkingTreesAround(sub), []string{wt}; !slices.Equal(got, want) {
		t.Errorf("WorkingTreesAround with the main checkout as the account home = %q, want %q", got, want)
	}
}

// TestEveryLinkedWorktreeOfTheRepositoryIsNamed: from inside one linked
// worktree, and from the main checkout, WorkingTreesAround names every linked
// worktree of the same repository, read from the shared git directory's
// worktrees/<name>/gitdir files without starting git. An entry whose working
// tree is gone, or whose gitdir does not name a .git, adds nothing, and the
// account's own home is not read for its worktrees.
func TestEveryLinkedWorktreeOfTheRepositoryIsNamed(t *testing.T) {
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	main := filepath.Join(base, "main")
	common := filepath.Join(main, ".git")
	write := func(p, s string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(s), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	worktree := func(name, dir string) {
		t.Helper()
		admin := filepath.Join(common, "worktrees", name)
		write(filepath.Join(dir, ".git"), "gitdir: "+admin+"\n")
		write(filepath.Join(admin, "commondir"), "../..\n")
		write(filepath.Join(admin, "gitdir"), filepath.Join(dir, ".git")+"\n")
	}
	a := filepath.Join(base, "trees", "a")
	b := filepath.Join(base, "elsewhere", "b")
	worktree("a", a)
	worktree("b", b)
	sub := filepath.Join(a, "pkg")
	if err := os.MkdirAll(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	// A worktree removed without pruning, and an entry naming no .git.
	write(filepath.Join(common, "worktrees", "gone", "gitdir"), filepath.Join(base, "gone", ".git")+"\n")
	write(filepath.Join(common, "worktrees", "odd", "gitdir"), base+"\n")

	accountHomeIs(t, filepath.Join(base, "nobody"), nil)
	if got, want := gitutil.WorkingTreesAround(sub), []string{a, main, b}; !slices.Equal(got, want) {
		t.Errorf("WorkingTreesAround(inside linked worktree a) = %q, want %q", got, want)
	}
	if got, want := gitutil.WorkingTreesAround(main), []string{main, a, b}; !slices.Equal(got, want) {
		t.Errorf("WorkingTreesAround(the main checkout) = %q, want %q", got, want)
	}

	// A bare repository's worktrees name each other, with no main checkout.
	bare := filepath.Join(base, "repo.git")
	c, d := filepath.Join(base, "bare", "c"), filepath.Join(base, "bare", "d")
	for name, dir := range map[string]string{"c": c, "d": d} {
		admin := filepath.Join(bare, "worktrees", name)
		write(filepath.Join(dir, ".git"), "gitdir: "+admin+"\n")
		write(filepath.Join(admin, "commondir"), "../..\n")
		write(filepath.Join(admin, "gitdir"), filepath.Join(dir, ".git")+"\n")
	}
	if got, want := gitutil.WorkingTreesAround(c), []string{c, d}; !slices.Equal(got, want) {
		t.Errorf("WorkingTreesAround(a bare repository's worktree) = %q, want %q", got, want)
	}

	accountHomeIs(t, main, nil)
	if got := gitutil.WorkingTreesAround(main); len(got) != 0 {
		t.Errorf("WorkingTreesAround(the main checkout as the account home) = %q, want none", got)
	}
}
