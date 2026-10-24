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

// hookRepo builds a repository with user.* set to name/email, the committed
// pre-commit hook installed, the machine-identity list it reads in the working
// tree (the copied hook has no checkout of its own beside it), and one file
// staged.
func hookRepo(t *testing.T, hook string, env []string, name, email string) string {
	t.Helper()
	dir := t.TempDir()
	hookGit(t, dir, env, "init")
	if name != "" {
		hookGit(t, dir, env, "config", "user.name", name)
	}
	if email != "" {
		hookGit(t, dir, env, "config", "user.email", email)
	}
	list, err := os.ReadFile(toolIdentitiesFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "internal", "core", "identity"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "internal", "core", "identity", toolIdentitiesFile), list, 0o644); err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(hook)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".git", "hooks"), 0o755); err != nil {
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

// commitThroughHook runs `git commit` with env and reports whether the hook
// refused it, and what it said.
func commitThroughHook(t *testing.T, dir string, env []string, args ...string) (bool, string) {
	t.Helper()
	cmd := exec.Command("git", append(append([]string{"-C", dir}, args...), "commit", "-m", "t")...)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	return err != nil, string(out)
}

// TestPreCommitHook_IdentityGate exercises the committed shell hook end to end
// against the person's own record on their machine (itd-2610240918547728),
// pinning down the fail-closed contract: a record the shell cannot read or
// trust must BLOCK, never fall through.
func TestPreCommitHook_IdentityGate(t *testing.T) {
	hook := locateHook(t)
	cases := []struct {
		name      string
		record    string // "" => no record on this machine
		pin       string // a committed .abcd/config/identity.json, now ignored
		gitName   string
		gitEmail  string
		wantBlock bool
		wantSay   string
	}{
		{name: "no record passes a human", gitName: "Whoever", gitEmail: "who@ever.com"},
		{name: "match passes", record: `{"name":"Alex","email":"a@b.com"}`, gitName: "Alex", gitEmail: "a@b.com"},
		{name: "mismatch blocks, naming the record and how to change it", record: `{"name":"Alex","email":"a@b.com"}`, gitName: "Alex", gitEmail: "typo@b.com", wantBlock: true, wantSay: "~/.abcd.noindex/identity.json, or remove it"},
		{name: "pretty-printed match passes", record: "{\n  \"name\": \"Alex\",\n  \"email\": \"a@b.com\"\n}\n", gitName: "Alex", gitEmail: "a@b.com"},
		{name: "non-canonical key blocks (fail closed)", record: `{"NAME":"Alex","email":"a@b.com"}`, gitName: "Alex", gitEmail: "a@b.com", wantBlock: true},
		{name: "empty value blocks (fail closed)", record: `{"name":"","email":""}`, gitName: "Alex", gitEmail: "a@b.com", wantBlock: true},
		{name: "malformed blocks (fail closed)", record: `{garbage`, gitName: "Alex", gitEmail: "a@b.com", wantBlock: true},
		// Criterion 2: a committed pin naming someone else decides nothing.
		{name: "a committed pin of the maintainer does not refuse a contributor", pin: `{"name":"A Maintainer","email":"m@example.com"}`, gitName: "Second Contributor", gitEmail: "second@example.com"},
		// Criterion 5: no record, and a machine identity, is never committed.
		{name: "an AI tool's identity blocks with no record", gitName: "Claude", gitEmail: "noreply@anthropic.com", wantBlock: true, wantSay: "machine identity"},
		{name: "a bot's identity blocks with no record", gitName: "dependabot[bot]", gitEmail: "49699333+dependabot[bot]@users.noreply.github.com", wantBlock: true},
		{name: "a machine identity blocks even when recorded", record: `{"name":"Claude","email":"noreply@anthropic.com"}`, gitName: "Claude", gitEmail: "noreply@anthropic.com", wantBlock: true},
		{name: "a person's forge privacy address passes", gitName: "Alex", gitEmail: "1234+alex@users.noreply.github.com"},
		// Not covered: a repository with no identity configured anywhere.
		// git then invents one from the account name and the host name and
		// exports it into the hook's environment, where it is
		// indistinguishable from one a launcher supplied; the Go check
		// reports it as git_identity.unset.
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := hookEnv(t)
			dir := hookRepo(t, hook, env, tc.gitName, tc.gitEmail)
			if tc.record != "" {
				writePin(t, dir, tc.record)
			}
			if tc.pin != "" {
				writeRepoPin(t, dir, tc.pin)
				hookGit(t, dir, env, "add", "-A")
			}
			blocked, out := commitThroughHook(t, dir, env)
			if blocked != tc.wantBlock {
				t.Fatalf("blocked=%v, want %v\n%s", blocked, tc.wantBlock, out)
			}
			if tc.wantSay != "" && !strings.Contains(out, tc.wantSay) {
				t.Fatalf("the refusal does not say %q:\n%s", tc.wantSay, out)
			}
			if strings.Contains(out, "Maintainer") {
				t.Fatalf("the hook named the maintainer:\n%s", out)
			}
		})
	}
}

// TestPreCommitHook_RefusesAnUntrustedRecord: the hook reads the record under
// the rules loader's guards and blocks on every one it fails; a symlinked
// ~/.abcd.noindex with no record in it reads as absent.
func TestPreCommitHook_RefusesAnUntrustedRecord(t *testing.T) {
	hook := locateHook(t)
	const rec = `{"name":"Alex","email":"a@b.com"}`
	cases := map[string]struct {
		setup     func(t *testing.T, home string)
		wantBlock bool
	}{
		"group-writable record": {setup: func(t *testing.T, home string) {
			p := writeHomeFile(t, MachineRecordRef().Rel, rec)
			if err := os.Chmod(p, 0o660); err != nil {
				t.Fatal(err)
			}
		}, wantBlock: true},
		"world-writable folder": {setup: func(t *testing.T, home string) {
			p := writeHomeFile(t, MachineRecordRef().Rel, rec)
			if err := os.Chmod(filepath.Dir(p), 0o777); err != nil {
				t.Fatal(err)
			}
		}, wantBlock: true},
		"record is a symlink": {setup: func(t *testing.T, home string) {
			p := writeHomeFile(t, MachineRecordRef().Rel, rec)
			if err := os.Rename(p, p+".real"); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(p+".real", p); err != nil {
				t.Fatal(err)
			}
		}, wantBlock: true},
		"record behind a symlinked home": {setup: func(t *testing.T, home string) {
			real := filepath.Join(t.TempDir(), "elsewhere")
			if err := os.MkdirAll(real, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(real, "identity.json"), []byte(rec), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(real, filepath.Join(home, ".abcd.noindex")); err != nil {
				t.Fatal(err)
			}
		}, wantBlock: true},
		"symlinked home with no record reads as absent": {setup: func(t *testing.T, home string) {
			real := filepath.Join(t.TempDir(), "elsewhere")
			if err := os.MkdirAll(real, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(real, filepath.Join(home, ".abcd.noindex")); err != nil {
				t.Fatal(err)
			}
		}},
		"oversized record": {setup: func(t *testing.T, home string) {
			writeHomeFile(t, MachineRecordRef().Rel, `{"name":"Alex","email":"a@b.com","pad":"`+strings.Repeat("x", 5000)+`"}`)
		}, wantBlock: true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			env := hookEnv(t)
			dir := hookRepo(t, hook, env, "Alex", "a@b.com")
			tc.setup(t, os.Getenv("HOME"))
			if blocked, out := commitThroughHook(t, dir, env); blocked != tc.wantBlock {
				t.Fatalf("blocked=%v, want %v\n%s", blocked, tc.wantBlock, out)
			}
		})
	}
}

// TestPreCommitHook_RepoRecordOutranksTheMachineRecord: the hook reads the
// record for this repository, keyed on its root commit, ahead of the
// machine-wide default, exactly as the Go check does.
func TestPreCommitHook_RepoRecordOutranksTheMachineRecord(t *testing.T) {
	hook := locateHook(t)
	for _, tc := range []struct {
		name      string
		repoRec   string
		wantBlock bool
	}{
		{name: "the repository's record matches", repoRec: `{"name":"Alex at Work","email":"alex@work.example"}`},
		{name: "the repository's record differs", repoRec: `{"name":"Alex at Work","email":"other@work.example"}`, wantBlock: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := hookEnv(t)
			dir := hookRepo(t, hook, env, "Alex at Work", "alex@work.example")
			hookGit(t, dir, env, "commit", "--no-verify", "-m", "root")
			sha := gittest.RootCommit(t, dir)
			writePin(t, dir, `{"name":"Alex Reppel","email":"alex@example.com"}`)
			ref, err := RepoRecordRef(sha)
			if err != nil {
				t.Fatal(err)
			}
			writeHomeFile(t, ref.Rel, tc.repoRec)
			if err := os.WriteFile(filepath.Join(dir, "g.txt"), []byte("y"), 0o644); err != nil {
				t.Fatal(err)
			}
			hookGit(t, dir, env, "add", "-A")
			if blocked, out := commitThroughHook(t, dir, env); blocked != tc.wantBlock {
				t.Fatalf("blocked=%v, want %v\n%s", blocked, tc.wantBlock, out)
			}
		})
	}
}

// TestPreCommitHook_ACopyWithNoListStillChecksTheRecord: a hook copied out of
// the checkout, with no machine-identity list anywhere near it, says so in one
// line and still holds the person to their own record; a list that is there
// but lacks a key blocks.
func TestPreCommitHook_ACopyWithNoListStillChecksTheRecord(t *testing.T) {
	hook := locateHook(t)
	env := hookEnv(t)
	dir := hookRepo(t, hook, env, "Alex", "typo@b.com")
	list := filepath.Join(dir, "internal", "core", "identity", toolIdentitiesFile)
	if err := os.Remove(list); err != nil {
		t.Fatal(err)
	}
	hookGit(t, dir, env, "add", "-A")
	writePin(t, dir, `{"name":"Alex","email":"a@b.com"}`)
	blocked, out := commitThroughHook(t, dir, env)
	if !blocked || !strings.Contains(out, "no machine-identity list") {
		t.Fatalf("want a one-line warning and the record still checked; blocked=%v\n%s", blocked, out)
	}
	env = hookEnv(t)
	dir = hookRepo(t, hook, env, "Alex", "a@b.com")
	if err := os.WriteFile(filepath.Join(dir, "internal", "core", "identity", toolIdentitiesFile), []byte("ai_name=x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	hookGit(t, dir, env, "add", "-A")
	if blocked, out := commitThroughHook(t, dir, env); !blocked || !strings.Contains(out, "has no ai_mail pattern") {
		t.Fatalf("a list missing a key must block; blocked=%v\n%s", blocked, out)
	}
}

// TestPreCommitHook_AuthorRoleConfigBlocks is iss-2609261454332615 at the shell
// guard: a repo-local author.name/author.email outranks user.name/user.email
// for the author git stamps, so a matching user.* must not wave through a commit
// that author.* attributes to someone else. The committer is held to the
// record the same way (itd-131), so committer.* is refused on the same terms.
func TestPreCommitHook_AuthorRoleConfigBlocks(t *testing.T) {
	hook := locateHook(t)
	for _, key := range []string{"author.name", "author.email", "committer.name", "committer.email"} {
		t.Run(key, func(t *testing.T) {
			env := hookEnv(t)
			dir := hookRepo(t, hook, env, "Alex", "a@b.com")
			writePin(t, dir, `{"name":"Alex","email":"a@b.com"}`)
			hookGit(t, dir, env, "config", key, "Test User")
			if blocked, _ := commitThroughHook(t, dir, env); !blocked {
				t.Fatalf("the hook let %s = Test User make a commit under a record of Alex", key)
			}
		})
	}
}

// hookEnv is gittest.Env with any ambient GIT_AUTHOR_*/GIT_COMMITTER_* removed:
// the hook reads them ahead of every config key, as git does, so a value
// inherited from whatever runs the tests would decide the case instead of the
// repository the case builds.
//
// Each call starts from a HOME of its own, so a record one case writes under
// ~/.abcd.noindex is never read by the next.
func hookEnv(t *testing.T) []string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
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

// TestPreCommitHook_IdentityEnvOutranksConfig is iss-2609261614268567: git
// stamps GIT_AUTHOR_NAME/GIT_AUTHOR_EMAIL and GIT_COMMITTER_NAME/
// GIT_COMMITTER_EMAIL ahead of every config key, so the guard must read them
// first too. A machine identity exported by a harness or CI runner must not
// commit past a record that the configured user.* satisfies, and an
// environment that sets the recorded identity over a different user.* must not
// be refused.
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
		{name: "recorded identity in the environment passes over a different user.*", user: [2]string{"Other", "o@example.com"}, env: pinnedEnv, wantBlock: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := hookEnv(t)
			dir := hookRepo(t, hook, env, tc.user[0], tc.user[1])
			writePin(t, dir, `{"name":"Alex","email":"a@b.com"}`)
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
