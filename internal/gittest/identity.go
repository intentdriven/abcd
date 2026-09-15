package gittest

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Person is one git author identity — the pair the redactors key on.
type Person struct{ Name, Email string }

// SplitIdentity gives the test process a caller whose git identity differs by
// scope — the everyday shape of a work checkout, and the one the identity
// redactors must cover in full: a GLOBAL identity in a per-test HOME's
// .gitconfig and a REPO-LOCAL persona in repo's .git/config.
//
// GIT_CONFIG_GLOBAL is pinned to that file and the system config to
// os.DevNull, so the production probe — which keeps global config in effect
// on purpose (gitutil.ScrubbedEnv) — resolves exactly these two people and
// nothing from the machine. repo is initialised when it is not a repository
// yet. A machine with no usable git skips the test.
func SplitIdentity(t *testing.T, repo string) (global, local Person) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	env := Env(t) // pins HOME to a test-owned temp when it is not one already
	global = Person{Name: "Personal Name", Email: "personal@private.example"}
	local = Person{Name: "Work Persona", Email: "work@corp.example"}

	gc := filepath.Join(os.Getenv("HOME"), ".gitconfig")
	cfg := "[user]\n\tname = " + global.Name + "\n\temail = " + global.Email + "\n"
	if err := os.WriteFile(gc, []byte(cfg), 0o600); err != nil {
		t.Fatalf("write global gitconfig: %v", err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", gc)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")

	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env = env
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if _, err := os.Stat(filepath.Join(repo, ".git")); err != nil {
		git("init", "-q")
	}
	git("config", "--local", "user.name", local.Name)
	git("config", "--local", "user.email", local.Email)
	return global, local
}
