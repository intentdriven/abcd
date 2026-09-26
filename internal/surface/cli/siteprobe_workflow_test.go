package cli

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// siteprobe_workflow_test.go holds iss-2609260709386741: site.yml probes for the
// `site` verb before it renders, and the probe read `abcd --help`. Once the root
// help was grouped (itd-146), that listing names only the person's verbs and `site`
// sits under `--help --agent`, so every site run refused a binary that has the
// verb. The test EXECUTES each probing step's own `run:` script out of the
// committed workflow against a stand-in `./abcd` that answers the help calls with
// the real command tree's output, so it judges what the step does, not its text.

const siteWorkflowPath = ".github/workflows/site.yml"

// siteProbeRefusal is the line every probing step prints when it refuses a
// binary; it is also how the test finds those steps.
const siteProbeRefusal = "::error::this binary has no site verb"

// siteProbeSteps returns the dedented `run: |` body of every site.yml step that
// probes for the site verb.
func siteProbeSteps(t *testing.T) []string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(testRepoRoot(), filepath.FromSlash(siteWorkflowPath)))
	if err != nil {
		t.Fatalf("read %s: %v", siteWorkflowPath, err)
	}
	lines := strings.Split(string(b), "\n")
	indent := func(s string) int { return len(s) - len(strings.TrimLeft(s, " ")) }
	var steps []string
	for i, line := range lines {
		if strings.TrimSpace(line) != "run: |" {
			continue
		}
		key := indent(line)
		body, bodyIndent := []string{}, -1
		for _, l := range lines[i+1:] {
			if strings.TrimSpace(l) == "" {
				body = append(body, "")
				continue
			}
			if indent(l) <= key {
				break
			}
			if bodyIndent < 0 {
				bodyIndent = indent(l)
			}
			body = append(body, l[bodyIndent:])
		}
		script := strings.Join(body, "\n") + "\n"
		if strings.Contains(script, siteProbeRefusal) {
			steps = append(steps, script)
		}
	}
	return steps
}

// siteEntryRe is a listing line naming the site verb, the shape the probe greps.
var siteEntryRe = regexp.MustCompile(`(?m)^[ \t]+site[ \t].*\n`)

// siteProbeStub is the stand-in binary. agent.txt present means the binary knows
// `--agent`; absent, it refuses the flag the way a binary from before the grouped
// help does (verified against v0.10.0: exit 2, one line on stderr, nothing on
// stdout). Any call that is not a help call is a verb the step ran, logged.
const siteProbeStub = `#!/usr/bin/env bash
dir="$(cd "$(dirname "$0")" && pwd)"
case " $* " in
  *" lint --help "*) cat "$dir/stub/lint.txt" ;;
  *" --agent "*)
    if [ -f "$dir/stub/agent.txt" ]; then cat "$dir/stub/agent.txt"
    else echo "abcd: unknown flag: --agent" >&2; exit 2; fi ;;
  " --help ") cat "$dir/stub/flat.txt" ;;
  *) echo "$*" >> "$dir/stub/ran.log" ;;
esac
`

// TestSiteWorkflowProbeFindsTheSiteVerb runs every probing step against four
// binaries: this tree's (the verb only in the agent listing), one from before the
// grouped help (the verb in the flat listing, `--agent` unknown), one from before
// the site slice (no verb, `--agent` unknown), and a grouped one with the verb
// withdrawn. The first two must reach the verb; the last two must refuse in one
// line without running anything.
func TestSiteWorkflowProbeFindsTheSiteVerb(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash unavailable")
	}
	steps := siteProbeSteps(t)
	// Four probing steps: render and its check (production), build and its check
	// (preview). A different count means a step gained or lost a probe, and this
	// test would silently judge the wrong set.
	if len(steps) != 4 {
		t.Fatalf("%s: found %d steps printing %q, want 4", siteWorkflowPath, len(steps), siteProbeRefusal)
	}

	flat, _ := executedHelp(t, "--help")
	agent, _ := executedHelp(t, "--help", "--agent")
	lint, _ := executedHelp(t, "lint", "--help")
	if siteEntryRe.MatchString(flat) || !siteEntryRe.MatchString(agent) {
		t.Fatalf("fixture premise broken: the site verb must be in the agent listing only\n--- --help:\n%s\n--- --help --agent:\n%s", flat, agent)
	}
	// Before the grouped help the flat listing named every verb; one line under a
	// flat heading stands in for it.
	legacyFlat := flat + "\nAvailable Commands:\n  site        Render the website\n"

	for _, bin := range []struct {
		name        string
		flat, agent string // agent "" = the binary refuses --agent
		wantVerb    bool
	}{
		{"this tree (grouped help)", flat, agent, true},
		{"grouped help predates, site present (v0.6.2 to v0.10.0)", legacyFlat, "", true},
		{"site slice predates (v0.6.1 and earlier)", flat, "", false},
		{"grouped help, site withdrawn", flat, siteEntryRe.ReplaceAllString(agent, ""), false},
	} {
		for i, step := range steps {
			t.Run(bin.name+"/step"+strconv.Itoa(i), func(t *testing.T) {
				repo := gittest.NewRepo(t)
				repo.Commit("fixture")
				root := repo.Root()
				repo.Write("stub/flat.txt", bin.flat)
				repo.Write("stub/lint.txt", lint)
				if bin.agent != "" {
					repo.Write("stub/agent.txt", bin.agent)
				}
				if err := os.WriteFile(filepath.Join(root, "abcd"), []byte(siteProbeStub), 0o755); err != nil {
					t.Fatal(err)
				}
				repo.Write("step.sh", step)

				cmd := exec.Command("bash", "--noprofile", "--norc", "-eo", "pipefail", "step.sh")
				cmd.Dir = root
				cmd.Env = append(repo.Env(), "TAG=v0.11.0")
				out, err := cmd.CombinedOutput()
				ran, _ := os.ReadFile(filepath.Join(root, "stub", "ran.log"))

				var exit *exec.ExitError
				switch {
				case bin.wantVerb && err != nil:
					t.Errorf("step %d refused a binary that has the site verb: %v\n%s\n--- step:\n%s", i, err, out, step)
				case bin.wantVerb && !strings.Contains(string(ran), "site"):
					t.Errorf("step %d exited 0 without running a site verb (ran %q)\n--- step:\n%s", i, ran, step)
				case !bin.wantVerb && !errors.As(err, &exit):
					t.Errorf("step %d did not refuse a binary with no site verb: err=%v\n%s", i, err, out)
				case !bin.wantVerb && (!strings.Contains(string(out), siteProbeRefusal) || len(ran) > 0):
					t.Errorf("step %d must refuse in its one line before running anything; ran %q, output:\n%s", i, ran, out)
				}
			})
		}
	}
}
