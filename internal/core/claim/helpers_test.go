package claim

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/gittest"
)

// t0 is the fixtures' clock origin.
var t0 = time.Date(2026, 10, 3, 14, 0, 0, 0, time.UTC)

// clockAt is a fixed clock.
func clockAt(t time.Time) func() time.Time { return func() time.Time { return t } }

// newCheckout is a repository standing for one person's working copy: its own
// git identity in its local config and its local tier in place.
func newCheckout(t *testing.T, email string) *gittest.Repo {
	t.Helper()
	r := gittest.NewRepo(t)
	r.Git("config", "user.name", "Pat Example")
	r.Git("config", "user.email", email)
	if err := os.MkdirAll(filepath.Join(r.Root(), ".abcd", ".work.local"), 0o755); err != nil {
		t.Fatal(err)
	}
	return r
}

// machineConfig writes a home's ~/.abcd.noindex/config.json and returns the home.
func machineConfig(t *testing.T, body string) string {
	t.Helper()
	home := t.TempDir()
	dir := filepath.Join(home, ".abcd.noindex")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return home
}

// gitLog records every git command the package runs while a test runs.
type gitLog struct {
	mu   sync.Mutex
	cmds [][]string
}

func traced(t *testing.T) *gitLog {
	t.Helper()
	l := &gitLog{}
	traceGit = func(args []string) {
		l.mu.Lock()
		defer l.mu.Unlock()
		l.cmds = append(l.cmds, args)
	}
	t.Cleanup(func() { traceGit = nil })
	return l
}

// subcommands is the git subcommand of each traced command, in order.
func (l *gitLog) subcommands() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	var out []string
	for _, c := range l.cmds {
		out = append(out, c[0])
	}
	return out
}

// anyArgContains reports whether any traced argument contains s.
func (l *gitLog) anyArgContains(s string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, c := range l.cmds {
		for _, a := range c {
			if strings.Contains(a, s) {
				return true
			}
		}
	}
	return false
}

// gitEnv runs one git command in r with env appended to the fixture's
// environment, failing the test on a non-zero exit.
func gitEnv(t *testing.T, r *gittest.Repo, env []string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", r.Root()}, args...)...)
	cmd.Env = append(append([]string(nil), r.Env()...), env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}
