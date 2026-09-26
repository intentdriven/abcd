package gitutil_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// fakeGitAnswering plants a git on PATH whose every answer is out, so the shape
// check on git's toplevel answer can be driven with answers real git never gives.
func fakeGitAnswering(t *testing.T, out string) {
	t.Helper()
	bin := t.TempDir()
	script := "#!/bin/sh\nprintf '%s' '" + out + "'\n"
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
}

// TestToplevelNamesTheRootOfARealCheckout is the ordinary answer: from a
// subdirectory, the working-tree root.
func TestToplevelNamesTheRootOfARealCheckout(t *testing.T) {
	repo := t.TempDir()
	if out, err := runGit(t, repo, "init", "-q"); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	sub := filepath.Join(repo, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	top, err := gitutil.Toplevel(sub)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.EvalSymlinks(repo)
	got, _ := filepath.EvalSymlinks(top)
	if got != want {
		t.Fatalf("Toplevel = %q, want %q", top, repo)
	}
	if _, err := gitutil.Toplevel(t.TempDir()); err == nil {
		t.Fatal("a directory outside any checkout was given a toplevel")
	}
}

// TestToplevelRefusesAnAnswerOfTheWrongShape is iss-2608292038186663: git's
// toplevel is always one absolute line naming a directory that contains the
// one asked about, and any other answer (relative, several lines, a directory
// elsewhere) is refused rather than handed on as a root.
func TestToplevelRefusesAnAnswerOfTheWrongShape(t *testing.T) {
	dir := t.TempDir()
	elsewhere := t.TempDir()
	for name, answer := range map[string]string{
		"relative":  "some/where",
		"two lines": dir + "\n" + dir,
		"elsewhere": elsewhere,
		"empty":     "",
	} {
		t.Run(name, func(t *testing.T) {
			fakeGitAnswering(t, answer)
			if top, err := gitutil.Toplevel(dir); err == nil {
				t.Fatalf("the answer %q was accepted as the toplevel %q", answer, top)
			}
		})
	}
}

// TestShowToplevelIsAskedOnlyThroughToplevel keeps the rule in one place: no
// production code outside this package runs `rev-parse --show-toplevel` itself,
// so no caller can skip the shape check. The one exception is the banlist
// worktree probe, which asks in git's absolute path format, requires the answer
// to equal the directory it asked from, and checks it with ToplevelShaped.
func TestShowToplevelIsAskedOnlyThroughToplevel(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	allowed := map[string]bool{
		filepath.Join("internal", "gitutil", "repo.go"):             true,
		filepath.Join("internal", "core", "banlist", "worktree.go"): true,
	}
	err = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if n := d.Name(); p != root && (strings.HasPrefix(n, ".") || n == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		if allowed[rel] {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if strings.Contains(string(data), `"--show-toplevel"`) {
			t.Errorf("%s runs rev-parse --show-toplevel itself; call gitutil.Toplevel", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestCheckoutRootRefusalNamesAWorktreeSettingPointingElsewhere is
// iss-2609260057112155: with core.worktree naming a working tree that does not
// contain the caller's directory, git answers with a toplevel Toplevel refuses
// for its shape, and CheckoutRoot refuses in turn. The refusal listed three
// causes (git absent, the repository unreadable, its ownership refused), none of
// which is this one, so it must name the fourth.
func TestCheckoutRootRefusalNamesAWorktreeSettingPointingElsewhere(t *testing.T) {
	repo := t.TempDir()
	if out, err := runGit(t, repo, "init", "-q"); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	elsewhere := t.TempDir()
	if out, err := runGit(t, repo, "config", "core.worktree", elsewhere); err != nil {
		t.Fatalf("git config: %v: %s", err, out)
	}
	sub := filepath.Join(repo, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := gitutil.CheckoutRoot(sub, "the issue ledger")
	if err == nil {
		t.Fatal("a checkout whose core.worktree points elsewhere was given a root")
	}
	if !strings.Contains(err.Error(), "core.worktree") {
		t.Errorf("the refusal does not name the core.worktree cause: %v", err)
	}
	if !strings.Contains(err.Error(), "git could not name the repository root") {
		t.Errorf("the refusal lost the phrase its front-door tests match: %v", err)
	}
}
