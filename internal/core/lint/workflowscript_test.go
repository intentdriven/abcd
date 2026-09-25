package lint_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// The helpers in this file EXECUTE a workflow step's own `run:` script, read
// out of the committed workflow, against fake tools placed first on PATH. A
// shape assertion can say a flag is present; only running the script can say
// what the step does with the values an event hands it. The fakes stand in for
// the network-facing tools (gh, gitleaks) the runner would call, and record or
// answer exactly what the test needs.

// stepScript returns a step's `run:` script: a one-line value, or the
// dedented body of a `run: |` block.
func stepScript(t *testing.T, step workflowStep) string {
	t.Helper()
	lines := strings.Split(step.body, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "run: ") && trimmed != "run: |" {
			return strings.TrimPrefix(trimmed, "run: ") + "\n"
		}
		if trimmed != "run: |" {
			continue
		}
		keyIndent := lineIndent(line)
		var body []string
		bodyIndent := -1
		for _, l := range lines[i+1:] {
			if strings.TrimSpace(l) == "" {
				body = append(body, "")
				continue
			}
			if lineIndent(l) <= keyIndent {
				break
			}
			if bodyIndent < 0 {
				bodyIndent = lineIndent(l)
			}
			body = append(body, l[bodyIndent:])
		}
		return strings.Join(body, "\n") + "\n"
	}
	t.Fatalf("step %q (line %d) has no `run:` script", step.name, step.line)
	return ""
}

// findStep returns the one step in a workflow whose body contains needle,
// failing when there is none or more than one: an assertion that silently
// picked the first of two would leave the second unchecked.
func findStep(t *testing.T, workflow, rel, needle string) workflowStep {
	t.Helper()
	steps, err := workflowSteps(workflow)
	if err != nil {
		t.Fatalf("%s: %v", rel, err)
	}
	var found []workflowStep
	for _, s := range steps {
		if strings.Contains(s.body, needle) {
			found = append(found, s)
		}
	}
	if len(found) != 1 {
		t.Fatalf("%s: %d steps contain %q; want exactly one", rel, len(found), needle)
	}
	return found[0]
}

// fakeTool writes an executable shell script named name into dir.
func fakeTool(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/usr/bin/env bash\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
}

// runScript runs script under bash in dir, with bin first on PATH and env
// added to a minimal environment. It returns combined output and the exit
// code; a failure to start bash at all fails the test.
func runScript(t *testing.T, script, dir, bin string, env map[string]string) (string, int) {
	t.Helper()
	cmd := exec.Command("bash", "-c", script)
	cmd.Dir = dir
	cmd.Env = []string{
		"PATH=" + bin + string(os.PathListSeparator) + os.Getenv("PATH"),
		"HOME=" + dir,
		"RUNNER_TEMP=" + t.TempDir(),
		"GIT_CONFIG_NOSYSTEM=1",
	}
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	out, err := cmd.CombinedOutput()
	if err == nil {
		return string(out), 0
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return string(out), ee.ExitCode()
	}
	t.Fatalf("run script: %v\n%s", err, out)
	return "", -1
}

// scratchRepo makes a throwaway repository with n commits on its current branch
// and returns its directory and the sha of each commit, oldest first.
func scratchRepo(t *testing.T, n int) (string, []string) {
	t.Helper()
	dir := t.TempDir()
	git := func(args ...string) string {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(gittest.Env(t),
			"GIT_AUTHOR_NAME=Test Person", "GIT_AUTHOR_EMAIL=person@example.com",
			"GIT_COMMITTER_NAME=Test Person", "GIT_COMMITTER_EMAIL=person@example.com")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q")
	var shas []string
	for i := 0; i < n; i++ {
		if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte(strings.Repeat("x", i+1)), 0o644); err != nil {
			t.Fatal(err)
		}
		git("add", "f.txt")
		git("commit", "-q", "-m", "commit")
		shas = append(shas, git("rev-parse", "HEAD"))
	}
	return dir, shas
}
