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
	env := hookEnv(t)

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
	env := hookEnv(t)
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

// hookEnv is gittest.Env with any ambient GIT_AUTHOR_*/GIT_COMMITTER_* removed:
// the hook reads them ahead of every config key, as git does, so a value
// inherited from whatever runs the tests would decide the case instead of the
// repository the case builds.
func hookEnv(t *testing.T) []string {
	t.Helper()
	var env []string
	for _, kv := range gittest.Env(t) {
		key, _, _ := strings.Cut(kv, "=")
		switch key {
		case "GIT_AUTHOR_NAME", "GIT_AUTHOR_EMAIL", "GIT_COMMITTER_NAME", "GIT_COMMITTER_EMAIL":
			continue
		}
		env = append(env, kv)
	}
	return env
}

// pinnedHookRepo builds a repository pinned to Alex <a@b.com>, with user.* set
// to name/email and the committed pre-commit hook installed, and one file staged.
func pinnedHookRepo(t *testing.T, hook string, env []string, name, email string) string {
	t.Helper()
	dir := t.TempDir()
	hookGit(t, dir, env, "init")
	hookGit(t, dir, env, "config", "user.name", name)
	hookGit(t, dir, env, "config", "user.email", email)
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
	return dir
}

// TestPreCommitHook_IdentityEnvOutranksConfig is iss-2609261614268567: git
// stamps GIT_AUTHOR_NAME/GIT_AUTHOR_EMAIL and GIT_COMMITTER_NAME/
// GIT_COMMITTER_EMAIL ahead of every config key, so the guard must read them
// first too. A machine identity exported by a harness or CI runner must not
// commit past a pin that the configured user.* satisfies, and an environment
// that sets the pinned identity over a different user.* must not be refused.
// git exports the author it resolved from --author and from -c author.* into
// the hook's environment, so those shapes are held by the same reading.
func TestPreCommitHook_IdentityEnvOutranksConfig(t *testing.T) {
	hook := locateHook(t)
	pinnedEnv := []string{"GIT_AUTHOR_NAME=Alex", "GIT_AUTHOR_EMAIL=a@b.com", "GIT_COMMITTER_NAME=Alex", "GIT_COMMITTER_EMAIL=a@b.com"}
	cases := []struct {
		name      string
		user      [2]string // the repository's user.name, user.email
		env       []string  // added to the commit's environment
		args      []string  // placed before "commit"
		flags     []string  // placed after "commit"
		wantBlock bool
	}{
		{name: "GIT_AUTHOR_NAME blocks", user: [2]string{"Alex", "a@b.com"}, env: []string{"GIT_AUTHOR_NAME=Claude"}, wantBlock: true},
		{name: "GIT_AUTHOR_EMAIL blocks", user: [2]string{"Alex", "a@b.com"}, env: []string{"GIT_AUTHOR_EMAIL=noreply@anthropic.com"}, wantBlock: true},
		{name: "GIT_COMMITTER_NAME blocks", user: [2]string{"Alex", "a@b.com"}, env: []string{"GIT_COMMITTER_NAME=Claude"}, wantBlock: true},
		{name: "GIT_COMMITTER_EMAIL blocks", user: [2]string{"Alex", "a@b.com"}, env: []string{"GIT_COMMITTER_EMAIL=noreply@anthropic.com"}, wantBlock: true},
		{name: "--author blocks", user: [2]string{"Alex", "a@b.com"}, flags: []string{"--author", "Claude <noreply@anthropic.com>"}, wantBlock: true},
		{name: "-c author.name blocks", user: [2]string{"Alex", "a@b.com"}, args: []string{"-c", "author.name=Claude"}, wantBlock: true},
		{name: "-c committer.email blocks", user: [2]string{"Alex", "a@b.com"}, args: []string{"-c", "committer.email=noreply@anthropic.com"}, wantBlock: true},
		{name: "pinned identity in the environment passes over a different user.*", user: [2]string{"Other", "o@example.com"}, env: pinnedEnv, wantBlock: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := hookEnv(t)
			dir := pinnedHookRepo(t, hook, env, tc.user[0], tc.user[1])
			args := append(append(append([]string{}, tc.args...), "commit", "-m", "t"), tc.flags...)
			cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
			cmd.Env = append(append([]string{}, env...), tc.env...)
			out, err := cmd.CombinedOutput()
			if blocked := err != nil; blocked != tc.wantBlock {
				t.Fatalf("blocked=%v, want %v\n%s", blocked, tc.wantBlock, out)
			}
		})
	}
}
