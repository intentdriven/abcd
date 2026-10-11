package lint_test

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/gittest"
)

// TestCIRunsOncePerChangeAndNightly holds ci.yml's triggers to the shape that
// runs the full gate once per change (iss-2610110938294826).
//
// With the merge queue, every commit on the default branch is a merge result
// its queue entry already tested, so a push trigger on main ran the whole gate
// a second time on a tree nothing could change any more: 68 push runs in the
// six days to 2026-10-11, against 71 merged pull requests, each taking a macOS
// slot the queue was waiting for. What a push run did that the queue cannot is
// done by a nightly schedule instead: it checks the default branch against what
// moves outside the tree, and it saves the default branch's Go build cache,
// which pull-request and queue runs restore. A dispatch trigger runs the same
// thing by hand.
//
// The schedule's minute is not the top of an hour, when scheduled runs across
// GitHub pile up and are delayed or dropped, and the run is once a day.
func TestCIRunsOncePerChangeAndNightly(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	const rel = ".github/workflows/ci.yml"
	workflow := readRepoFile(t, root, rel)

	triggers := workflowTriggers(workflow)
	if slices.Contains(triggers, "push") {
		t.Errorf("%s triggers on push; every commit on the default branch is a merge result the "+
			"queue already tested, so a push run repeats the full gate on it", rel)
	}
	for _, want := range []string{"pull_request", "merge_group", "schedule", "workflow_dispatch"} {
		if !slices.Contains(triggers, want) {
			t.Errorf("%s does not trigger on %s (it triggers on %q)", rel, want, triggers)
		}
	}

	crons := regexp.MustCompile(`(?m)^    - cron: "([^"]*)"\s*$`).FindAllStringSubmatch(workflow, -1)
	if len(crons) != 1 {
		t.Fatalf("%s declares %d cron schedule(s); want the one nightly run", rel, len(crons))
	}
	fields := strings.Fields(crons[0][1])
	if len(fields) != 5 {
		t.Fatalf("%s: cron %q is not five fields", rel, crons[0][1])
	}
	minute, errM := strconv.Atoi(fields[0])
	hour, errH := strconv.Atoi(fields[1])
	if errM != nil || errH != nil || minute < 1 || minute > 59 || hour < 0 || hour > 23 ||
		fields[2] != "*" || fields[3] != "*" || fields[4] != "*" {
		t.Errorf("%s: cron %q is not one run a day at a fixed hour and a minute past the top "+
			"of the hour", rel, crons[0][1])
	}
}

// TestCIClassifierRunsTheFullMatrixOffPullRequests runs the changes job's own
// script, read from ci.yml, on each event the workflow triggers on. Only a pull
// request can be classified inert: the merge-queue entry, the nightly run and
// a dispatched one carry no base to diff against, and each must resolve to the
// full matrix. The pull-request case on a docs-only diff is the control that
// proves the same script can answer inert at all.
func TestCIClassifierRunsTheFullMatrixOffPullRequests(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	const rel = ".github/workflows/ci.yml"
	workflow := readRepoFile(t, root, rel)
	script := stepScript(t, findStep(t, workflow, rel, "is_inert_path()"))

	r := gittest.NewRepo(t)
	r.Write("cmd/x.go", "package main\n")
	r.Commit("baseline")
	base := r.Git("rev-parse", "HEAD")
	r.Write("docs/guide.md", "# Guide\n")
	r.Commit("docs only")
	bin := t.TempDir()

	for _, event := range workflowTriggers(workflow) {
		want := "false"
		if event == "pull_request" {
			want = "true"
		}
		t.Run(event, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "github-output")
			log, code := runScript(t, script, r.Root(), bin, map[string]string{
				"EVENT_NAME":    event,
				"BASE_SHA":      base,
				"GITHUB_OUTPUT": out,
			})
			if code != 0 {
				t.Fatalf("the classifier exited %d; it must exit zero on every path:\n%s", code, log)
			}
			raw, err := os.ReadFile(out)
			if err != nil {
				t.Fatalf("the classifier recorded no verdict:\n%s", log)
			}
			if got := strings.TrimSpace(string(raw)); got != "inert="+want {
				t.Errorf("on a %s event over a docs-only diff the classifier recorded %q; want inert=%s\n%s",
					event, got, want, log)
			}
		})
	}
}

// TestRangeGatesAnnounceANoBaseRun runs ci.yml's two shell range gates —
// record-lint's `-agent-diff` arm and the issue-resolution range check — with
// the empty base the nightly and dispatched runs hand them, against a fake
// toolchain. Each must exit zero (the tree-shaped rules still run, and RS003
// already ran) and say so as an annotation naming the event: a notice where no
// base is ever named, a warning on the two events that should always name one.
// A skipped gate that says nothing is a gate nobody knows is off.
func TestRangeGatesAnnounceANoBaseRun(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	const rel = ".github/workflows/ci.yml"
	workflow := readRepoFile(t, root, rel)
	repo, _ := scratchRepo(t, 1)
	bin := t.TempDir()
	argsFile := filepath.Join(t.TempDir(), "go-args")
	fakeTool(t, bin, "go", `printf '%s\n' "$*" >> "`+argsFile+`"`+"\n")

	gates := []struct {
		needle string
		ranAs  string // what the unarmed run must still invoke, "" for none
	}{
		{`-agent-diff "${BASE_SHA}...HEAD"`, "run ./cmd/record-lint"},
		{"scripts/check-issue-resolution.sh commits", ""},
	}
	for _, g := range gates {
		step := findStep(t, workflow, rel, g.needle)
		script := stepScript(t, step)
		for _, c := range []struct{ event, level string }{
			{"schedule", "::notice"},
			{"workflow_dispatch", "::notice"},
			{"merge_group", "::warning"},
			{"pull_request", "::warning"},
		} {
			t.Run(step.name+"/"+c.event, func(t *testing.T) {
				_ = os.Remove(argsFile)
				out, code := runScript(t, script, repo, bin, map[string]string{
					"BASE_SHA":          "",
					"GITHUB_EVENT_NAME": c.event,
				})
				if code != 0 {
					t.Fatalf("exit %d with no base; want 0:\n%s", code, out)
				}
				if !strings.Contains(out, c.level) || !strings.Contains(out, c.event) {
					t.Errorf("with no base on a %s run the step printed:\n%s\nwant a %s annotation naming the event",
						c.event, out, c.level)
				}
				raw, _ := os.ReadFile(argsFile)
				if g.ranAs != "" && strings.TrimSpace(string(raw)) != g.ranAs {
					t.Errorf("with no base the step ran go %q; want the unarmed %q, so the tree-shaped rules still run",
						strings.TrimSpace(string(raw)), g.ranAs)
				}
				if g.ranAs == "" && len(raw) != 0 {
					t.Errorf("with no base the step ran go %q; it must skip the range check", raw)
				}
			})
		}
	}
}
