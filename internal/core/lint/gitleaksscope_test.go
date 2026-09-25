package lint_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGitleaksScansOnlyTheHistoryThisRunWouldLand holds ci.yml's secret scan
// to the history of the commit under test, never every ref the checkout
// fetched (iss-2608282038283692).
//
// The job checks out with fetch-depth: 0, which fetches every remote branch,
// and a bare `gitleaks git .` walks `git log --all`. A secret literal on a
// stale branch whose pull request had closed therefore failed every open pull
// request until someone deleted the branch — observed 2026-08-28, when one
// superseded branch blocked seven unrelated pull requests — and the truncated
// finding named neither the branch nor the file.
//
// The step's own script is run against a fake gitleaks that records its
// arguments. On a pull request the scan is the pull request's own range,
// base..HEAD. On a push or a merge-queue entry it is the full history
// reachable from HEAD — the history that lands — which keeps the full-history
// property without walking other refs. A pull request whose base cannot be
// resolved scans wider (HEAD), never narrower, and says so.
func TestGitleaksScansOnlyTheHistoryThisRunWouldLand(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	const rel = ".github/workflows/ci.yml"
	step := findStep(t, readRepoFile(t, root, rel), rel, "gitleaks git ")
	script := stepScript(t, step)

	repo, shas := scratchRepo(t, 2)
	base := shas[0]
	bin := t.TempDir()
	argsFile := filepath.Join(t.TempDir(), "gitleaks-args")
	fakeTool(t, bin, "gitleaks", `printf '%s\n' "$@" > "`+argsFile+`"`+"\n")

	cases := []struct {
		name  string
		env   map[string]string
		scope string
	}{
		{"pull_request", map[string]string{"EVENT_NAME": "pull_request", "BASE_SHA": base}, base + "..HEAD"},
		{"push", map[string]string{"EVENT_NAME": "push", "BASE_SHA": ""}, "HEAD"},
		{"merge_group", map[string]string{"EVENT_NAME": "merge_group", "BASE_SHA": ""}, "HEAD"},
		{"pull_request, base absent from the checkout", map[string]string{
			"EVENT_NAME": "pull_request", "BASE_SHA": strings.Repeat("ab", 20)}, "HEAD"},
		{"pull_request, no base in the payload", map[string]string{"EVENT_NAME": "pull_request", "BASE_SHA": ""}, "HEAD"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_ = os.Remove(argsFile)
			out, code := runScript(t, script, repo, bin, c.env)
			if code != 0 {
				t.Fatalf("the scan step exited %d:\n%s", code, out)
			}
			raw, err := os.ReadFile(argsFile)
			if err != nil {
				t.Fatalf("the step never ran gitleaks:\n%s", out)
			}
			args := strings.Split(strings.TrimSpace(string(raw)), "\n")
			if len(args) == 0 || args[0] != "git" {
				t.Fatalf("gitleaks args %q: want the git subcommand", args)
			}
			want := "--log-opts=" + c.scope
			found := false
			for _, a := range args {
				if a == want {
					found = true
				}
				if strings.Contains(a, "--all") {
					t.Errorf("the scan passes %q; it must never walk every fetched ref", a)
				}
			}
			if !found {
				t.Errorf("gitleaks args %q: want %q, so the scan is scoped to the history this run lands", args, want)
			}
		})
	}
}
