package identity

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

func mustGitID(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = gittest.Env(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// unsetEnv removes key for the rest of the test and restores it afterwards. An
// empty value is not the same as unset to git (it refuses an empty ident), so the
// git var cross-checks below need the variable gone, not blank.
func unsetEnv(t *testing.T, key string) {
	t.Helper()
	t.Setenv(key, "")
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}
}

// envRepo is a hermetic repository with user.* set to Real Name <real@example.com>
// and every ambient identity and command-line config variable cleared, so each
// case decides the identity through the one source it sets.
func envRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	for _, k := range []string{
		"GIT_AUTHOR_NAME", "GIT_AUTHOR_EMAIL", "GIT_COMMITTER_NAME", "GIT_COMMITTER_EMAIL",
		"GIT_CONFIG_PARAMETERS", "GIT_CONFIG_COUNT", "GIT_CONFIG_KEY_0", "GIT_CONFIG_VALUE_0",
		"GIT_CONFIG", "GIT_DIR",
	} {
		unsetEnv(t, k)
	}
	dir := t.TempDir()
	mustGitID(t, dir, "init")
	mustGitID(t, dir, "config", "user.email", "real@example.com")
	mustGitID(t, dir, "config", "user.name", "Real Name")
	return dir
}

// gitVarIdent is what git itself would stamp for role in dir under the test's
// current environment: `git var GIT_<ROLE>_IDENT`, without the timestamp.
func gitVarIdent(t *testing.T, dir string, role Role) string {
	t.Helper()
	cmd := exec.Command("git", "-C", dir, "var", "GIT_"+strings.ToUpper(string(role))+"_IDENT")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git var: %v", err)
	}
	ident := strings.TrimSpace(string(out))
	if i := strings.LastIndex(ident, ">"); i >= 0 {
		ident = ident[:i+1]
	}
	return ident
}

// TestIdentityReadsTheCommandLineConfigGitCommitsWith is iss-2609261614306830:
// git commit honours configuration handed to it on the command line (`git -c`,
// which reaches a hook as GIT_CONFIG_PARAMETERS) and through the
// GIT_CONFIG_COUNT/KEY_n/VALUE_n form, so the identity reader must see it too.
// Scrubbing it made `abcd ahoy --identity` report the configured identity as ok
// while git stamped another. Each case is cross-checked against git var, so the
// test states git's behaviour, not an assumption about it.
func TestIdentityReadsTheCommandLineConfigGitCommitsWith(t *testing.T) {
	cases := []struct {
		name string
		env  [][2]string
		role Role
		want Effective
	}{
		{
			name: "GIT_CONFIG_PARAMETERS committer.name",
			env:  [][2]string{{"GIT_CONFIG_PARAMETERS", "'committer.name'='Claude'"}},
			role: RoleCommitter,
			want: Effective{Name: "Claude", Email: "real@example.com"},
		},
		{
			name: "GIT_CONFIG_PARAMETERS author.email",
			env:  [][2]string{{"GIT_CONFIG_PARAMETERS", "'author.email'='noreply@anthropic.com'"}},
			role: RoleAuthor,
			want: Effective{Name: "Real Name", Email: "noreply@anthropic.com"},
		},
		{
			name: "GIT_CONFIG_COUNT user.email",
			env:  [][2]string{{"GIT_CONFIG_COUNT", "1"}, {"GIT_CONFIG_KEY_0", "user.email"}, {"GIT_CONFIG_VALUE_0", "forged@example.com"}},
			role: RoleAuthor,
			want: Effective{Name: "Real Name", Email: "forged@example.com"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := envRepo(t)
			for _, kv := range tc.env {
				t.Setenv(kv[0], kv[1])
			}
			if got, want := gitVarIdent(t, dir, tc.role), tc.want.Name+" <"+tc.want.Email+">"; got != want {
				t.Fatalf("the case does not state git's behaviour: git var stamps %q, the case expects %q", got, want)
			}
			got, err := effective(dir, tc.role)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("the %s read %+v, but git commits with %+v", tc.role, got, tc.want)
			}
		})
	}
}

// TestCheckHoldsTheCommitterGitCommitsWith is iss-2609261614306830 at the gate:
// under `git -c committer.name=Claude commit`, a hook running `abcd ahoy
// --identity` in a pinned repository whose user.* matches the pin must refuse,
// because git does not export the committer it resolves into the hook's
// environment and the command-line config is the only place it shows.
func TestCheckHoldsTheCommitterGitCommitsWith(t *testing.T) {
	dir := envRepo(t)
	writePin(t, dir, `{"name":"Real Name","email":"real@example.com"}`)
	t.Setenv("GIT_CONFIG_PARAMETERS", "'committer.name'='Claude'")
	res, err := Check(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !res.CommitterDiverges || !res.Blocks() {
		t.Fatalf("the gate passed a commit git stamps with committer Claude: %+v", res)
	}
}

// TestIdentityReadIgnoresWhatGitCommitDoesNot keeps the rest of the scrub: an
// inherited GIT_DIR must not redirect the read at another repository, and the
// legacy GIT_CONFIG file variable (read by `git config` alone, ignored by
// `git commit`) must not change the identity the gate reports.
func TestIdentityReadIgnoresWhatGitCommitDoesNot(t *testing.T) {
	t.Run("GIT_DIR", func(t *testing.T) {
		dir := envRepo(t)
		other := t.TempDir()
		mustGitID(t, other, "init")
		mustGitID(t, other, "config", "user.email", "other@example.com")
		t.Setenv("GIT_DIR", filepath.Join(other, ".git"))
		eff, err := EffectiveIdentity(dir)
		if err != nil {
			t.Fatal(err)
		}
		if eff.Email != "real@example.com" {
			t.Errorf("an inherited GIT_DIR redirected the identity read: got %q, want real@example.com", eff.Email)
		}
	})
	t.Run("GIT_CONFIG", func(t *testing.T) {
		dir := envRepo(t)
		legacy := filepath.Join(t.TempDir(), "legacy.cfg")
		if err := os.WriteFile(legacy, []byte("[user]\n\temail = legacy@example.com\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Setenv("GIT_CONFIG", legacy)
		if got := gitVarIdent(t, dir, RoleAuthor); got != "Real Name <real@example.com>" {
			t.Fatalf("the case does not state git's behaviour: git var stamps %q under GIT_CONFIG", got)
		}
		eff, err := EffectiveIdentity(dir)
		if err != nil {
			t.Fatal(err)
		}
		if eff.Email != "real@example.com" {
			t.Errorf("the legacy GIT_CONFIG file changed the identity read: got %q, want real@example.com", eff.Email)
		}
	})
}
