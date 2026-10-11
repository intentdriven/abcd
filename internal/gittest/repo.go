package gittest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Repo is a hermetic throwaway git repository for tests that must exercise real
// git objects.
//
// It exists because three packages needed the same fixture. The release
// derivation, the surface guardrail, and the ship verb all read their inputs out
// of git trees (tags, blobs, set-differences), so a stubbed git would prove
// nothing about the behaviour under test — each one has to build an actual
// history. internal/core/changelog and internal/surface/cli each grew a private
// copy of this helper; this is the promotion those copies asked for, so a fourth
// caller extends one implementation instead of writing a fourth.
//
// Every command runs under Env(t): the ambient GIT_DIR/GIT_WORK_TREE and the
// developer's global config are stripped, so a test can never be redirected onto
// the real repository.
type Repo struct {
	t    *testing.T
	root string
	env  []string
}

// NewRepo initialises an empty repository on a fixed branch name under a
// per-test temp directory. A machine with no usable git skips the test rather
// than failing it: the fixture's subject is abcd's behaviour, not git's presence.
func NewRepo(t *testing.T) *Repo {
	t.Helper()
	r := &Repo{t: t, root: t.TempDir(), env: Env(t)}
	init := exec.Command("git", "-C", r.root, "init", "--initial-branch=main")
	init.Env = r.env
	if out, err := init.CombinedOutput(); err != nil {
		t.Skipf("git init unavailable: %v (%s)", err, out)
	}
	disableMaintenance(t, r.env, r.root)
	return r
}

// maintenanceOff is the repository config every fixture repository carries.
// Background maintenance is disabled in the REPO config, not per command,
// because the code under test runs its own git against the repository: an
// auto-gc or fsmonitor child spawned by any of those commands can outlive the
// test and race t.TempDir cleanup on its objects (iss-252 — a release verify
// run lost to exactly that; iss-2610112202081803 — a bare origin's gc after a
// push). Repo-level config covers every git invocation there, whoever makes it.
var maintenanceOff = [][2]string{
	{"gc.auto", "0"},
	{"gc.autodetach", "false"},
	{"maintenance.auto", "false"},
	{"core.fsmonitor", "false"},
}

// disableMaintenance writes maintenanceOff into the local config of the
// repository at dir, a working tree or a bare git directory.
func disableMaintenance(t *testing.T, env []string, dir string) {
	t.Helper()
	for _, kv := range maintenanceOff {
		cfg := exec.Command("git", "-C", dir, "config", "--local", kv[0], kv[1])
		cfg.Env = env
		if out, err := cfg.CombinedOutput(); err != nil {
			t.Fatalf("git config %s: %v (%s)", kv[0], err, out)
		}
	}
}

// Root is the repository's absolute path — what the code under test is handed.
func (r *Repo) Root() string { return r.root }

// Env is the isolated environment the fixture's git commands run under, for a
// caller that must spawn git itself.
func (r *Repo) Env() []string { return r.env }

// Git runs one git command in the fixture and returns its trimmed output,
// failing the test on a non-zero exit. The identity is pinned per command rather
// than written into the repo's config, so a test that inspects the config sees
// only what it put there.
func (r *Repo) Git(args ...string) string {
	r.t.Helper()
	full := append([]string{
		"-C", r.root,
		"-c", "user.email=fixture@example.invalid",
		"-c", "user.name=Fixture",
		"-c", "commit.gpgsign=false",
	}, args...)
	cmd := exec.Command("git", full...)
	cmd.Env = r.env
	out, err := cmd.CombinedOutput()
	if err != nil {
		r.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// Write creates (or overwrites) a repo-relative file, creating parents. The path
// is slash-separated so tests read the same on every platform.
func (r *Repo) Write(rel, content string) {
	r.t.Helper()
	path := filepath.Join(r.root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		r.t.Fatal(err)
	}
}

// Remove deletes a repo-relative file from the working tree.
func (r *Repo) Remove(rel string) {
	r.t.Helper()
	if err := os.Remove(filepath.Join(r.root, filepath.FromSlash(rel))); err != nil {
		r.t.Fatal(err)
	}
}

// Commit stages everything and records a commit. Empty commits are allowed so a
// fixture can advance history without touching a file.
func (r *Repo) Commit(msg string) {
	r.t.Helper()
	r.Git("add", "-A")
	r.Git("commit", "--allow-empty", "-m", msg)
}

// Record writes a minimal record file carrying the given impact frontmatter —
// the shape the release derivation reads.
func (r *Repo) Record(rel, id, impact string) {
	r.t.Helper()
	r.Write(rel, "---\nid: "+id+"\nimpact: "+impact+"\n---\n# "+id+"\n")
}

// AddSubmodule commits a checked-out submodule at rel: a second fixture
// repository holding one committed file, s.txt, added by path and committed in
// this one. It returns the submodule's own git directory
// (.git/modules/<rel>), whose config is the submodule's config, for a test
// that plants something a superproject command must not reach into.
func (r *Repo) AddSubmodule(rel string) (gitDir string) {
	r.t.Helper()
	sub := NewRepo(r.t)
	sub.Write("s.txt", "s\n")
	sub.Commit("sub seed")
	r.Git("-c", "protocol.file.allow=always", "submodule", "--quiet", "add", sub.Root(), rel)
	r.Git("commit", "-q", "-m", "add submodule "+rel)
	return filepath.Join(r.root, ".git", "modules", filepath.FromSlash(rel))
}

// InitBare creates an empty bare repository at path on branch main, with the
// same maintenance config NewRepo writes: a bare fixture is usually an origin
// the test pushes into, and a push spawns the origin's auto-gc. Every test
// makes its bare repositories here or through CloneBare, never with its own
// git init --bare.
func InitBare(t *testing.T, path string) {
	t.Helper()
	env := Env(t)
	cmd := exec.Command("git", "init", "-q", "--bare", "--initial-branch=main", path)
	cmd.Env = env
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %v (%s)", err, out)
	}
	disableMaintenance(t, env, path)
}

// CloneBare clones src into a new bare repository at path, with the same
// maintenance config as InitBare.
func CloneBare(t *testing.T, src, path string) {
	t.Helper()
	env := Env(t)
	cmd := exec.Command("git", "clone", "-q", "--bare", src, path)
	cmd.Env = env
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git clone --bare: %v (%s)", err, out)
	}
	disableMaintenance(t, env, path)
}
