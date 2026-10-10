package loop

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// TestPickGitStartsNoAutoMaintenance is iss-2610091935334207. pickGit keeps
// the repository's config (ScrubbedEnv), so a repository with a low gc.auto
// makes the pick commit start `git maintenance run --auto` and `gc --auto`,
// which repacks the loose objects and runs gc.recentObjectsHook, a program the
// repository names. pickGit pins gc.auto=0 and maintenance.auto=false on the
// command line, where they outrank the repository's config, and sets
// GIT_NO_LAZY_FETCH=1 so a merge in a partial clone fetches nothing.
func TestPickGitStartsNoAutoMaintenance(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX sentinel script")
	}
	r := gittest.NewRepo(t)
	// Two packs over a gc.autoPackLimit of 1 is what makes `gc --auto` act,
	// whatever the loose-object sample says.
	for i := 0; i < 2; i++ {
		r.Write("f"+string(rune('a'+i))+".md", strings.Repeat("x", i+1)+"\n")
		r.Commit("seed")
		r.Git("repack", "-q", "-d")
	}
	hookMark := filepath.Join(t.TempDir(), "recent-objects-hook-ran")
	hook := filepath.Join(t.TempDir(), "recent-hook.sh")
	if err := os.WriteFile(hook, []byte("#!/bin/sh\ntouch "+hookMark+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	envMark := filepath.Join(t.TempDir(), "filter-env")
	filter := filepath.Join(t.TempDir(), "env-filter.sh")
	if err := os.WriteFile(filter, []byte("#!/bin/sh\necho \"lazy=$GIT_NO_LAZY_FETCH\" > "+envMark+"\ncat\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	for k, v := range map[string]string{
		"user.name": "Fixture", "user.email": "fixture@example.invalid",
		"gc.auto": "1", "gc.autoDetach": "false", "gc.autoPackLimit": "1",
		"maintenance.auto": "true", "maintenance.autoDetach": "false",
		"gc.recentObjectsHook": hook, "gc.cruftPacks": "true",
		"filter.envspy.clean": filter,
	} {
		r.Git("config", k, v)
	}
	if err := os.WriteFile(filepath.Join(r.Root(), ".git", "info", "attributes"), []byte("*.md filter=envspy\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	packs := func() int {
		m, _ := filepath.Glob(filepath.Join(r.Root(), ".git", "objects", "pack", "*.pack"))
		return len(m)
	}
	if packs() != 2 {
		t.Fatalf("fixture: want two packs, got %d", packs())
	}

	r.Write("pick.md", "picked\n")
	if _, err := pickGit(r.Root(), "add", "--", "pick.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := pickGit(r.Root(), "commit", "-q", "-m", "pick", "--", "pick.md"); err != nil {
		t.Fatal(err)
	}
	if n := packs(); n != 2 {
		t.Fatalf("the pick commit started an automatic gc: %d pack(s) where there were 2", n)
	}
	if _, err := os.Stat(hookMark); err == nil {
		t.Fatal("the pick commit ran the repository's gc.recentObjectsHook")
	}
	got, err := os.ReadFile(envMark)
	if err != nil {
		t.Fatalf("fixture: the clean filter did not run: %v", err)
	}
	if strings.TrimSpace(string(got)) != "lazy=1" {
		t.Fatalf("pickGit must run with GIT_NO_LAZY_FETCH=1, the filter saw %q", strings.TrimSpace(string(got)))
	}

	// The fixture is live: the same commit under the repository's own config,
	// without pickGit's pins, starts the automatic gc.
	r.Write("again.md", "again\n")
	for _, args := range [][]string{{"add", "--", "again.md"}, {"commit", "-q", "-m", "again"}} {
		c := exec.Command("git", append([]string{"-c", "core.hooksPath=/dev/null", "-C", r.Root()}, args...)...)
		c.Env = gitutil.ScrubbedEnv()
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("fixture git %v: %v\n%s", args, err, out)
		}
	}
	if packs() == 2 {
		t.Fatal("fixture: a plain commit did not start an automatic gc, so this test proves nothing")
	}
}
