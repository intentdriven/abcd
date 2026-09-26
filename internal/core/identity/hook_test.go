package identity

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// locateHook finds the committed .githooks/pre-commit by walking up from the
// test's working directory. Skips when not run from a checkout (e.g. a build
// tarball) or when bash/git are unavailable.
func locateHook(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash unavailable")
	}
	topLevel := exec.Command("git", "rev-parse", "--show-toplevel")
	topLevel.Env = gittest.Env(t)
	out, err := topLevel.Output()
	if err != nil {
		t.Skip("not in a git checkout")
	}
	hook := filepath.Join(strings.TrimSpace(string(out)), ".githooks", "pre-commit")
	if _, err := os.Stat(hook); err != nil {
		t.Skipf("hook not found: %v", err)
	}
	return hook
}

func hookGit(t *testing.T, dir string, env []string, args ...string) error {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil && !strings.Contains(strings.Join(args, " "), "commit") {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return err
}

// TestPreCommitHook_IdentityGate exercises the committed shell hook end to end,
// pinning down the fail-closed contract: a pin the shell cannot read must BLOCK,
// never fall through (the F1 regression this test guards).
func TestPreCommitHook_IdentityGate(t *testing.T) {
	hook := locateHook(t)
	// Isolate from the machine's git global/system config and ~/.abcd corpus so
	// the hook only sees the temp repo (gittest.Env pins HOME/XDG to a temp dir
	// and neutralises global/system config).
	env := gittest.Env(t)

	cases := []struct {
		name      string
		pin       string // "" => no identity.json
		gitName   string
		gitEmail  string
		wantBlock bool
	}{
		{"no pin passes", "", "Whoever", "who@ever.com", false},
		{"match passes", `{"name":"Alex","email":"a@b.com"}`, "Alex", "a@b.com", false},
		{"mismatch blocks", `{"name":"Alex","email":"a@b.com"}`, "Test User", "test@example.com", true},
		{"pretty-printed match passes", "{\n  \"name\": \"Alex\",\n  \"email\": \"a@b.com\"\n}\n", "Alex", "a@b.com", false},
		{"non-canonical key blocks (fail closed)", `{"NAME":"Alex","email":"a@b.com"}`, "Alex", "a@b.com", true},
		{"empty value blocks (fail closed)", `{"name":"","email":""}`, "Alex", "a@b.com", true},
		{"malformed blocks (fail closed)", `{garbage`, "Alex", "a@b.com", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			hookGit(t, dir, env, "init")
			hookGit(t, dir, env, "config", "user.name", tc.gitName)
			hookGit(t, dir, env, "config", "user.email", tc.gitEmail)
			if tc.pin != "" {
				if err := os.MkdirAll(filepath.Join(dir, ".abcd", "config"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, ".abcd", "config", "identity.json"), []byte(tc.pin), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			hooksDir := filepath.Join(dir, ".git", "hooks")
			if err := os.MkdirAll(hooksDir, 0o755); err != nil {
				t.Fatal(err)
			}
			src, err := os.ReadFile(hook)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(hooksDir, "pre-commit"), src, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}
			hookGit(t, dir, env, "add", "-A")
			err = hookGit(t, dir, env, "commit", "-m", "t")
			blocked := err != nil
			if blocked != tc.wantBlock {
				t.Fatalf("blocked=%v, want %v", blocked, tc.wantBlock)
			}
		})
	}
}

// TestPreCommitHook_AuthorRoleConfigBlocks is iss-2609261454332615 at the shell
// guard: a repo-local author.name/author.email outranks user.name/user.email
// for the author git stamps, so a matching user.* must not wave through a commit
// that author.* attributes to someone else. The committer is held to the pin the
// same way (itd-131), so committer.* is refused on the same terms.
func TestPreCommitHook_AuthorRoleConfigBlocks(t *testing.T) {
	hook := locateHook(t)
	env := gittest.Env(t)
	for _, key := range []string{"author.name", "author.email", "committer.name", "committer.email"} {
		t.Run(key, func(t *testing.T) {
			dir := t.TempDir()
			hookGit(t, dir, env, "init")
			hookGit(t, dir, env, "config", "user.name", "Alex")
			hookGit(t, dir, env, "config", "user.email", "a@b.com")
			hookGit(t, dir, env, "config", key, "Test User")
			if err := os.MkdirAll(filepath.Join(dir, ".abcd", "config"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, ".abcd", "config", "identity.json"), []byte(`{"name":"Alex","email":"a@b.com"}`), 0o644); err != nil {
				t.Fatal(err)
			}
			src, err := os.ReadFile(hook)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, ".git", "hooks", "pre-commit"), src, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}
			hookGit(t, dir, env, "add", "-A")
			if err := hookGit(t, dir, env, "commit", "-m", "t"); err == nil {
				t.Fatalf("the hook let %s = Test User make a commit under a pin of Alex", key)
			}
		})
	}
}
