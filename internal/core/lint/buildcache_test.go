package lint_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Without -trimpath go puts a package's directory in its compile key, so every
// checkout compiles its own copy of every package, and the build cache grows by
// a whole module per worktree. So everything that builds this tree on a person's
// machine, one checkout after another, builds under -trimpath: the script every
// abcd invocation goes through, the two commit hooks, and every Go step a Makefile
// target makes. And since those targets run tests under it, CI's check job does
// too, or preflight would judge a form of the tests CI never runs.

// The flag reaches a Makefile target through the environment rather than on each
// recipe line, so it reaches the go a script or a test starts as well, and leaves
// the argv alone: go run ./cmd/abcd implement load --wait is how every other
// waiter reads this one in the process table. A person's own GOFLAGS is kept.
func TestMakeTargetsBuildUnderTrimpath(t *testing.T) {
	if _, err := exec.LookPath("make"); err != nil {
		t.Skip("no make on PATH")
	}
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	probe := filepath.Join(t.TempDir(), "probe.mk")
	if err := os.WriteFile(probe, []byte("goflags-probe:\n\t@printf '%s\\n' \"$$GOFLAGS\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ inherited, want string }{
		{"", "-trimpath"},
		{"-mod=mod", "-mod=mod -trimpath"},
		{"-trimpath", "-trimpath"},
		{"-trimpath -mod=mod", "-mod=mod -trimpath"},
	} {
		cmd := exec.Command("make", "--no-print-directory", "-f", "Makefile", "-f", probe, "goflags-probe")
		cmd.Dir = root
		cmd.Env = append(os.Environ(), "GOFLAGS="+c.inherited)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("make with GOFLAGS=%q: %v\n%s", c.inherited, err, out)
		}
		if got := strings.TrimSpace(string(out)); got != c.want {
			t.Errorf("with GOFLAGS=%q a recipe's go runs under GOFLAGS=%q, want %q", c.inherited, got, c.want)
		}
	}
}

// TestCIBuildsUnderTrimpath: make preflight runs the check job's steps, so the job
// builds and tests in the form preflight does. The workflow's env reaches every
// step of every job, so no job saves a form of the build cache another cannot use,
// and the workflow's own test lines stay as the Makefile reads them. A job that set
// its own GOFLAGS would take itself out of that, so none does.
func TestCIBuildsUnderTrimpath(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	head, jobs, ok := strings.Cut(readRepoFile(t, root, ".github/workflows/ci.yml"), "\njobs:\n")
	if !ok {
		t.Fatal("ci.yml: no `jobs:` key; the parser or the workflow changed shape")
	}
	if !regexp.MustCompile(`(?m)^env:\n(?:  .*\n)*?  GOFLAGS: -trimpath$`).MatchString(head) {
		t.Error("ci.yml's workflow env does not set GOFLAGS: -trimpath, so CI would test a form preflight does not")
	}
	if regexp.MustCompile(`(?m)^\s+GOFLAGS:`).MatchString(jobs) {
		t.Error("a job in ci.yml sets GOFLAGS of its own; it is declared once, in the workflow's env")
	}
}

// TestCommitHooksBuildUnderTrimpath: each hook builds the checkout's own abcd on
// every commit, so a hook without the flag gives each worktree its own copy of
// every package whatever the rest does.
func TestCommitHooksBuildUnderTrimpath(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	build := regexp.MustCompile(`"\$go_bin" build ([^\n]*)\./cmd/abcd`)
	for _, hook := range []string{".githooks/commit-msg", ".githooks/pre-commit"} {
		found := build.FindAllStringSubmatch(readRepoFile(t, root, hook), -1)
		if len(found) == 0 {
			t.Errorf("%s: no build of ./cmd/abcd found; the check or the hook changed shape", hook)
		}
		for _, m := range found {
			if !strings.Contains(m[1], "-trimpath") {
				t.Errorf("%s: a build without -trimpath: %s", hook, m[0])
			}
		}
	}
}
