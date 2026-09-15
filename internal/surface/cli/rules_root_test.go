package cli

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// gitInitAt turns dir into a git working tree under the hermetic test env, so
// the root resolver has a toplevel to bound at. A machine with no usable git
// skips the test: the subject is abcd's resolution, not git's presence.
func gitInitAt(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "-C", dir, "init", "--initial-branch=main")
	cmd.Env = gittest.Env(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("git init unavailable: %v (%s)", err, out)
	}
}

// realPath resolves symlinks so macOS /var -> /private/var cannot defeat a
// compare between a path the test built and one git reported.
func realPath(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q): %v", p, err)
	}
	return r
}

// TestRulesRootWalksUpToDotAbcd is the attack/behaviour test for the rules-loader
// root fix: run from a subdirectory of a git working tree, rulesRoot must return
// the nearest directory inside that tree holding a .abcd directory, not cwd.
// Handing rules.Load a subdirectory silently ignored the per-repo overrides AND
// the kill switch, so a repo that had disabled a domain would still inject it
// from any nested directory.
func TestRulesRootWalksUpToDotAbcd(t *testing.T) {
	repo := t.TempDir()
	gitInitAt(t, repo)
	if err := os.MkdirAll(filepath.Join(repo, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(repo, "internal", "deep", "pkg")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	wantRepo := realPath(t, repo)
	if got := realPath(t, rulesRoot(sub, io.Discard)); got != wantRepo {
		t.Errorf("rulesRoot(%q) = %q, want the .abcd-bearing ancestor %q", sub, got, wantRepo)
	}
	// From the repo root itself, rulesRoot returns it unchanged.
	if got := realPath(t, rulesRoot(repo, io.Discard)); got != wantRepo {
		t.Errorf("rulesRoot(repo root) = %q, want %q", got, wantRepo)
	}
}

// TestRulesRootStopsAtGitToplevel is the stop-test for GHSA-vvqc-3mv2-5p49: a
// .abcd directory planted ABOVE the git working tree must never govern a session
// inside it. The walk is bounded at the toplevel git reports — a repo with no
// .abcd of its own resolves to its own root, never to an ancestor's plant — and
// a .abcd inside the tree still wins by nearest-first.
func TestRulesRootStopsAtGitToplevel(t *testing.T) {
	outer := t.TempDir()
	if err := os.MkdirAll(filepath.Join(outer, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	inner := filepath.Join(outer, "inner-repo")
	gitInitAt(t, inner)
	sub := filepath.Join(inner, "pkg")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	wantInner := realPath(t, inner)

	// No .abcd inside the tree: the toplevel, not the planted ancestor.
	if got := realPath(t, rulesRoot(sub, io.Discard)); got != wantInner {
		t.Errorf("rulesRoot(%q) = %q, escaped the git working tree %q", sub, got, wantInner)
	}
	if got := realPath(t, rulesRoot(inner, io.Discard)); got != wantInner {
		t.Errorf("rulesRoot(%q) = %q, escaped the git working tree %q", inner, got, wantInner)
	}

	// A .abcd at the repo root is the nearest one inside the tree.
	if err := os.MkdirAll(filepath.Join(inner, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := realPath(t, rulesRoot(sub, io.Discard)); got != wantInner {
		t.Errorf("rulesRoot(%q) = %q, want the repo's own .abcd at %q", sub, got, wantInner)
	}

	// Nearest-first still holds inside the tree: a .abcd in the subdirectory
	// itself beats the one at the root.
	if err := os.MkdirAll(filepath.Join(sub, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, want := realPath(t, rulesRoot(sub, io.Discard)), realPath(t, sub); got != want {
		t.Errorf("rulesRoot(%q) = %q, want the nearest .abcd inside the tree %q", sub, got, want)
	}
}

// TestRulesRootNonGitDoesNotWalk pins the other half of the bound: outside any
// git working tree there is no toplevel to stop at, so there is no walk at all.
// cwd is the root — never an ancestor, never $HOME, never the shared temp dir.
func TestRulesRootNonGitDoesNotWalk(t *testing.T) {
	outer := t.TempDir()
	if err := os.MkdirAll(filepath.Join(outer, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	plain := filepath.Join(outer, "plain")
	if err := os.MkdirAll(plain, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := rulesRoot(plain, io.Discard); got != plain {
		t.Errorf("rulesRoot(%q) = %q, want cwd itself (a non-git directory must not walk to an ancestor's .abcd)", plain, got)
	}
}

// TestHookPromptRouterIgnoresAncestorRules is GHSA-vvqc-3mv2-5p49 at the hook
// front door: a rules.json planted above the git working tree must inject
// nothing into a session inside it.
func TestHookPromptRouterIgnoresAncestorRules(t *testing.T) {
	t.Setenv("ABCD_RULES_STATE_DIR", t.TempDir())
	outer := t.TempDir()
	if err := os.MkdirAll(filepath.Join(outer, ".abcd"), 0o755); err != nil {
		t.Fatal(err)
	}
	planted := `{"schema_version":1,"domains":{"ANCESTOR":{"state":"active","recall":["ancestor"],"rules":["rules planted OUTSIDE the git repo"]}}}`
	if err := os.WriteFile(filepath.Join(outer, ".abcd", "rules.json"), []byte(planted), 0o644); err != nil {
		t.Fatal(err)
	}
	inner := filepath.Join(outer, "inner-repo")
	gitInitAt(t, inner)

	out, errlog := runHook(t, hookInputJSON(t, "ancestor", inner, "ancestor"), "hook", "prompt-router")
	if strings.Contains(out, "## ANCESTOR") {
		t.Fatalf("a .abcd planted above the git working tree governed the session:\n%s", out)
	}
	if out != "" {
		t.Fatalf("expected a no-match (zero model-facing stdout), got:\n%s\nstderr:\n%s", out, errlog)
	}
}
