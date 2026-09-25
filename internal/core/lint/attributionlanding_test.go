package lint_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAttributionGatesTheCommitsThatLandOnMain holds attribution.yml's commit
// half to the commits that actually land (iss-280).
//
// The commit check read only a pull request's base..head, and exempted the
// merge-queue entry. So the commit a merge creates on main — a squash commit
// composed from the pull request's title and body, a web-UI merge — was
// checked by nothing: `make check-attribution` walks origin/main..HEAD, whose
// base is main itself, and CI never looked again. The queue entry is the run
// that judges the would-be merge result, and a push to main is the one that
// sees what landed however it got there, so both run the commit half now.
//
// Three parts: the triggers; the range each event's own BASE_SHA/HEAD_SHA
// expressions resolve to (evaluated, not pattern-matched); and the steps'
// scripts, executed against fake gate scripts, so an exemption that swallows
// an event is caught by what the step does rather than by how it reads.
func TestAttributionGatesTheCommitsThatLandOnMain(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	const rel = ".github/workflows/attribution.yml"
	wf := readRepoFile(t, root, rel)

	if !strings.Contains(wf, "\n  push:\n    branches: [main]\n") {
		t.Error("attribution.yml must trigger on push to main, so a squash or web merge is judged once it lands")
	}
	if !strings.Contains(wf, "\n  merge_group:\n") {
		t.Error("attribution.yml must keep its merge_group trigger")
	}

	commits := findStep(t, wf, rel, "check-attribution.sh commits")
	const (
		prBase   = "3e5d9a1c7f2b4086d3a5c9e1b7f4d2a6c8e0b5d9"
		prHead   = "5a7c9e1b3d5f7092b4d6f8a0c2e4b6d8f0a2c4e6"
		pushHead = "8b0d2f4a6c8e0193d5f7b9a1c3e5d7f9b1d3f5a7"
	)
	pull := withoutSubtree(contextFor(map[string]any{
		"github.event_name":                  "pull_request",
		"github.event.pull_request.number":   float64(620),
		"github.event.pull_request.base.sha": prBase,
		"github.event.pull_request.head.sha": prHead,
		"github.sha":                         "not-the-head-a-pull-request-checks",
	}), "github.event.merge_group.")
	queue := contextFor(map[string]any{"github.event.pull_request.head.sha": nil})
	push := withoutSubtree(contextFor(map[string]any{
		"github.event_name":                  "push",
		"github.ref":                         "refs/heads/main",
		"github.event.before":                pushBeforeSHA,
		"github.sha":                         pushHead,
		"github.event.pull_request.head.sha": nil,
	}), "github.event.merge_group.")
	for _, ev := range []struct {
		name       string
		ctx        map[string]any
		base, head string
	}{
		{"pull_request", pull, prBase, prHead},
		{"merge_group", queue, mergeGroupBaseSHA, stringify(mergeGroupContext["github.event.merge_group.head_sha"])},
		{"push", push, pushBeforeSHA, pushHead},
	} {
		for key, want := range map[string]string{"BASE_SHA": ev.base, "HEAD_SHA": ev.head} {
			raw, ok := commits.env[key]
			if !ok {
				t.Errorf("the commit step declares no %s env", key)
				continue
			}
			got, err := evalWorkflowValue(raw, ev.ctx)
			if err != nil {
				t.Errorf("%s on a %s event: cannot evaluate %s: %v", key, ev.name, raw, err)
				continue
			}
			if stringify(got) != want {
				t.Errorf("%s on a %s event resolves to %q, want %q (%s)", key, ev.name, stringify(got), want, raw)
			}
		}
	}

	// The scripts, run in a scratch repository whose scripts/ holds fakes that
	// record how they were called.
	repo, shas := scratchRepo(t, 2)
	if err := os.MkdirAll(filepath.Join(repo, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	calls := filepath.Join(t.TempDir(), "calls")
	for _, name := range []string{"check-attribution.sh", "check-issue-resolution.sh"} {
		fakeTool(t, filepath.Join(repo, "scripts"), name, `echo "`+name+` $*" >> "`+calls+`"`+"\n")
	}
	run := func(step workflowStep, env map[string]string) (string, int, string) {
		_ = os.Remove(calls)
		out, code := runScript(t, stepScript(t, step), repo, t.TempDir(), env)
		got, _ := os.ReadFile(calls)
		return out, code, strings.TrimSpace(string(got))
	}

	commitScript := func(event, base, head string) map[string]string {
		return map[string]string{"EVENT_NAME": event, "BASE_SHA": base, "HEAD_SHA": head, "ABCD_OUTBOUND_BIN": "/nonexistent"}
	}
	for _, event := range []string{"pull_request", "merge_group", "push"} {
		out, code, got := run(commits, commitScript(event, shas[0], shas[1]))
		want := "check-attribution.sh commits " + shas[0] + " " + shas[1]
		if code != 0 || got != want {
			t.Errorf("%s: the commit step must run %q; exit %d, called %q\n%s", event, want, code, got, out)
		}
	}
	for _, bad := range []string{"", absentSHA, strings.Repeat("ab", 20)} {
		out, code, got := run(commits, commitScript("push", bad, shas[1]))
		if code == 0 || got != "" {
			t.Errorf("push with base %q: the commit step must refuse loudly without a range (exit %d, called %q)\n%s", bad, code, got, out)
		}
	}

	// The pull-request FORM steps have nothing to read off a queue entry or a
	// push, and must say so and pass rather than judge an empty body.
	for _, needle := range []string{"check-attribution.sh body", "check-issue-resolution.sh pr"} {
		step := findStep(t, wf, rel, needle)
		for _, event := range []string{"merge_group", "push"} {
			out, code, got := run(step, map[string]string{"EVENT_NAME": event, "PR_AUTHOR_TYPE": "User", "ABCD_OUTBOUND_BIN": "/nonexistent"})
			if code != 0 || got != "" || !strings.Contains(out, "nothing was checked") {
				t.Errorf("step %q on %s must exempt itself loudly (exit %d, called %q)\n%s", step.name, event, code, got, out)
			}
		}
	}
}
