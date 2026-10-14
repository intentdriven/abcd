package site

// The project-site switch as the release workflow reads it
// (itd-2610091010348892): the switch step setup writes is run with bash in a
// checkout shaped the way actions/checkout leaves one (the released tag
// checked out, fetch-depth 0, so every branch is there as origin/<branch>),
// and its output decides whether the deploy job runs at all. Nothing here
// reaches a network.

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/hosting"
	"github.com/intentdriven/abcd/internal/adapter/hosting/githubpages"
	"github.com/intentdriven/abcd/internal/gittest"
)

// switchStep is the run block of the workflow's switch step, unindented.
func switchStep(t *testing.T, wf string) string {
	t.Helper()
	lines := strings.Split(wf, "\n")
	start := -1
	for i, l := range lines {
		if strings.TrimSpace(l) == "id: switch" {
			start = i
			break
		}
	}
	if start < 0 {
		t.Fatal("the workflow has no switch step")
	}
	var script []string
	runIndent := -1
	for _, l := range lines[start+1:] {
		indent := len(l) - len(strings.TrimLeft(l, " "))
		if runIndent < 0 {
			if strings.TrimSpace(l) == "run: |" {
				runIndent = indent
			}
			continue
		}
		if strings.TrimSpace(l) != "" && indent <= runIndent {
			break
		}
		script = append(script, strings.TrimPrefix(l, strings.Repeat(" ", runIndent+2)))
	}
	if len(script) == 0 {
		t.Fatal("the switch step has no run block")
	}
	return strings.Join(script, "\n") + "\n"
}

func TestTheSwitchIsReadAtTheTagAndTheDefaultBranchAndOnlyWhilePublic(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq is not installed; the workflow's runner carries it")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is not installed")
	}
	wf, err := renderSiteWorkflow("trunk", githubpages.Adapter{}, hosting.Site{Name: "widgets", Repo: "acme/widgets"})
	if err != nil {
		t.Fatal(err)
	}
	script := switchStep(t, string(wf))

	const (
		on        = `{"hosting": {"provider": "github-pages", "name": "widgets", "publish": true}}`
		off       = `{"hosting": {"provider": "github-pages", "name": "widgets", "publish": false}}`
		malformed = `{"hosting": {"publish": true`
		missing   = ""
		unset     = "<unset>"
	)
	for name, c := range map[string]struct {
		atTag, atHead, private, want string
	}{
		"on at the tag and the head, public":           {on, on, "false", "true"},
		"switched off on the default branch since":     {on, off, "false", "false"},
		"no switch file on the default branch":         {on, missing, "false", "false"},
		"a malformed switch file on the branch":        {on, malformed, "false", "false"},
		"off at the tag, on at the head":               {off, on, "false", "false"},
		"made private since setup":                     {on, on, "true", "false"},
		"a visibility the event does not carry":        {on, on, "", "false"},
		"a visibility the step was handed no value of": {on, on, unset, "false"},
	} {
		t.Run(name, func(t *testing.T) {
			origin := gittest.NewRepo(t)
			origin.Git("checkout", "-q", "-b", "trunk")
			origin.Write(ManifestRelPath, c.atTag)
			origin.Commit("the released commit")
			origin.Git("tag", "v1.0.0")
			if c.atHead == missing {
				origin.Remove(ManifestRelPath)
			} else {
				origin.Write(ManifestRelPath, c.atHead)
			}
			origin.Commit("the default branch since the release")

			clone := filepath.Join(t.TempDir(), "checkout")
			origin.Git("clone", "-q", "--no-checkout", origin.Root(), clone)
			git := func(args ...string) {
				t.Helper()
				cmd := exec.Command("git", append([]string{"-C", clone, "-c", "gc.auto=0", "-c", "maintenance.auto=false"}, args...)...)
				cmd.Env = origin.Env()
				if out, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("git %v: %v\n%s", args, err, out)
				}
			}
			git("checkout", "-q", "--detach", "v1.0.0")

			output := filepath.Join(t.TempDir(), "github_output")
			if err := os.WriteFile(output, nil, 0o644); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("bash", "-c", script)
			cmd.Dir = clone
			cmd.Env = append(origin.Env(), "GITHUB_OUTPUT="+output)
			if c.private != unset {
				cmd.Env = append(cmd.Env, "REPO_PRIVATE="+c.private)
			}
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("the switch step failed: %v\n%s", err, out)
			}
			raw, err := os.ReadFile(output)
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.TrimSpace(string(raw)); got != "on="+c.want {
				t.Fatalf("the switch step wrote %q, want on=%s", got, c.want)
			}
		})
	}
}

// TestTheDeployJobRunsOnlyForAPublicRepository pins the deploy job's gate:
// the switch the render job read, and the repository's visibility as the
// event carries it, compared as JSON so a missing field never reads as public.
func TestTheDeployJobRunsOnlyForAPublicRepository(t *testing.T) {
	wf, err := renderSiteWorkflow("main", githubpages.Adapter{}, hosting.Site{Name: "widgets", Repo: "acme/widgets"})
	if err != nil {
		t.Fatal(err)
	}
	gate := regexp.MustCompile(`(?m)^  deploy:\n    needs: render\n    if: needs\.render\.outputs\.publish == 'true' && toJSON\(github\.event\.repository\.private\) == 'false'\n`)
	if !gate.Match(wf) {
		t.Error("the deploy job is not gated on the switch and on a public repository")
	}
	if !strings.Contains(string(wf), "          REPO_PRIVATE: ${{ github.event.repository.private }}\n") {
		t.Error("the switch step is not handed the repository's visibility")
	}
}

// TestThePagesWorkflowIsPinned pins the workflow setup writes for the
// github-pages provider byte for byte.
func TestThePagesWorkflowIsPinned(t *testing.T) {
	wf, err := renderSiteWorkflow("main", githubpages.Adapter{}, hosting.Site{Name: "widgets", Repo: "acme/widgets"})
	if err != nil {
		t.Fatal(err)
	}
	golden(t, "site-workflow-github-pages.yml", wf)
}
