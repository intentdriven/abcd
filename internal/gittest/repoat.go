package gittest

import (
	"os/exec"
	"testing"
)

// NewRepoAt is NewRepo at root, a directory the caller made and owns, for a
// fixture that keeps its repository at a path of its own choosing.
func NewRepoAt(t *testing.T, root string) *Repo {
	t.Helper()
	r := &Repo{t: t, root: root, env: Env(t)}
	init := exec.Command("git", "-C", r.root, "init", "--initial-branch=main")
	init.Env = r.env
	if out, err := init.CombinedOutput(); err != nil {
		t.Skipf("git init unavailable: %v (%s)", err, out)
	}
	disableMaintenance(t, r.env, r.root)
	return r
}
