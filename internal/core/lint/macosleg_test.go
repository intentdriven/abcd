package lint_test

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/actionsexpr"
)

// TestMacOSLegRunsItsSuiteOnlyWhereItLands holds ci.yml's check job to where
// its macOS leg spends a macOS runner (iss-2610110938282654).
//
// macOS is the account's scarcest runner, five jobs at once, and the leg was
// the slowest in the workflow. It ran on every pull request as well as on every
// merge-queue entry, so on 2026-10-11 queue entries waited for a slot behind
// pull-request runs that repeated a local `make preflight`, passed the queue's
// timeout and were removed, and nothing merged for five hours. The leg now runs
// its suite on every event but pull_request; on a pull request it is a
// stand-in that reports `check (macos-latest)` from an ubuntu runner and runs
// nothing. Where it does run, it tests in one race-enabled pass over the whole
// module (iss-2610110938293273), while the ubuntu leg keeps its plain pass and
// its race lane.
//
// The stand-in must still REPORT, because the context is a required check in
// the ruleset: so the job itself may not be skipped on that event, only its
// steps, and the matrix must keep the macos-latest value the context is named
// from.
//
// The check is SEMANTIC, like the concurrency and range-base contracts beside
// it: every step's own `if:` and the job's `runs-on:` are evaluated per leg, per
// event the workflow triggers on, and per classifier verdict, so a condition
// rewritten in a spelling nobody anticipated passes and one that merely reads
// right fails. The event set is derived from the workflow's `on:` block, and
// an expression the evaluator cannot resolve fails the test.
func TestMacOSLegRunsItsSuiteOnlyWhereItLands(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	const rel = ".github/workflows/ci.yml"
	workflow := readRepoFile(t, root, rel)
	job, ok := workflowJobBlock(workflow, "check")
	if !ok {
		t.Fatalf("%s defines no check job", rel)
	}
	legs := checkLegTimeouts(t, rel+" job check", job)
	if !slices.ContainsFunc(legs, func(l checkLeg) bool { return l.os == "macos-latest" }) {
		t.Fatalf("%s: the check job's matrix has no macos-latest leg; the required context "+
			"`check (macos-latest)` is named from that value and would never report", rel)
	}

	runsOn, ok := jobKey(job, "runs-on")
	if !ok {
		t.Fatalf("%s: the check job declares no single-line runs-on", rel)
	}
	jobIf, _ := jobKey(job, "if")
	items := jobSteps(job)
	if len(items) < 5 {
		t.Fatalf("%s: the check job parsed into %d step(s); the step parser has lost its shape", rel, len(items))
	}
	events := workflowTriggers(workflow)
	if !slices.Contains(events, "pull_request") || !slices.Contains(events, "merge_group") {
		t.Fatalf("%s triggers on %q; the trigger parser has lost its shape, or the workflow no "+
			"longer gates pull requests and the merge queue", rel, events)
	}

	for _, leg := range legs {
		for _, event := range events {
			verdicts := []string{"false"}
			if event == "pull_request" {
				// Only a pull request can be classified inert; every other
				// event runs the full matrix (the changes job's own contract).
				verdicts = append(verdicts, "true")
			}
			for _, inert := range verdicts {
				name := fmt.Sprintf("%s/%s/inert=%s", leg.os, event, inert)
				t.Run(name, func(t *testing.T) {
					ctx := map[string]any{
						"matrix.os":                   leg.os,
						"github.event_name":           event,
						"needs.changes.outputs.inert": inert,
						"success()":                   true,
						"failure()":                   false,
						"cancelled()":                 false,
					}
					if jobIf != "" {
						run, err := actionsexpr.EvalIf(jobIf, ctx)
						if err != nil {
							t.Fatalf("cannot evaluate the job's if %q: %v", jobIf, err)
						}
						if !run {
							t.Fatalf("the check job's if %q skips the whole job here; a required "+
								"context must report from a job that runs", jobIf)
						}
					}
					runner, err := actionsexpr.EvalValue(runsOn, ctx)
					if err != nil {
						t.Fatalf("cannot evaluate runs-on %q: %v", runsOn, err)
					}
					var running []jobStep
					for _, s := range items {
						if s.cond == "" {
							running = append(running, s)
							continue
						}
						run, err := actionsexpr.EvalIf(s.cond, ctx)
						if err != nil {
							t.Fatalf("line %d, step %s: cannot evaluate if %q: %v\n\nThis gate fails "+
								"closed; simplify the condition or teach internal/actionsexpr the shape.",
								s.line, s.label, s.cond, err)
						}
						if run {
							running = append(running, s)
						}
					}
					runs := func(cmd string) bool {
						return slices.ContainsFunc(running, func(s jobStep) bool { return s.runs(cmd) })
					}

					standIn := leg.os == "macos-latest" && event == "pull_request"
					switch {
					case standIn:
						if r := actionsexpr.Stringify(runner); r != "ubuntu-latest" {
							t.Errorf("the macOS leg's stand-in on a pull request runs on %q; want "+
								"ubuntu-latest, so it takes no macOS runner", r)
						}
						if len(running) == 0 {
							t.Errorf("the stand-in runs no step at all; one step must say in the log " +
								"that this context tested nothing on macOS")
						}
						for _, s := range running {
							if s.uses != "" {
								t.Errorf("line %d, step %s runs on the macOS leg's pull-request stand-in "+
									"and uses %s; the stand-in reports the context and runs nothing",
									s.line, s.label, s.uses)
							}
							if m := toolchainCommandRe.FindString(s.script()); m != "" {
								t.Errorf("line %d, step %s runs on the macOS leg's pull-request "+
									"stand-in and runs %q; the stand-in reports the context and "+
									"runs nothing", s.line, s.label, strings.TrimSpace(m))
							}
						}
					default:
						if r := actionsexpr.Stringify(runner); r != leg.os {
							t.Errorf("the %s leg runs on %q on a %s event; want its own runner", leg.os, r, event)
						}
						for _, cmd := range []string{"go build ./...", "go vet ./..."} {
							if !runs(cmd) {
								t.Errorf("the %s leg runs no %q on a %s event", leg.os, cmd, event)
							}
						}
						var passes []string
						for _, s := range running {
							for _, l := range strings.Split(s.script(), "\n") {
								if l = strings.TrimSpace(l); strings.HasPrefix(l, "go test ") {
									passes = append(passes, l)
								}
							}
						}
						if leg.os == "macos-latest" {
							// One race-enabled pass over the whole module
							// (iss-2610110938293273): a plain pass beside it
							// reran every test outside internal/ at a third of
							// the leg's time.
							if len(passes) != 1 || !strings.HasPrefix(passes[0], "go test -race ") ||
								!strings.HasSuffix(passes[0], " "+modulePattern) {
								t.Errorf("the macOS leg runs %d test pass(es) on a %s event: %q; want one "+
									"`go test -race ... %s`", len(passes), event, passes, modulePattern)
							}
							return
						}
						// The plain pass may name its slowest packages ahead of
						// the pattern; TestPlainTestStepRunsTheWholeModule holds
						// its set to exactly the module's.
						plain := slices.ContainsFunc(passes, func(p string) bool {
							return !strings.HasPrefix(p, "go test -race ") && strings.HasSuffix(p, " "+modulePattern)
						})
						if !plain {
							t.Errorf("the %s leg runs no plain `go test ... ./...` on a %s event: %q", leg.os, event, passes)
						}
						hasRace := slices.ContainsFunc(passes, func(p string) bool { return strings.HasPrefix(p, "go test -race ") })
						if want := inert != "true"; hasRace != want {
							t.Errorf("the %s leg's race pass runs=%v on a %s event with inert=%s; want %v",
								leg.os, hasRace, event, inert, want)
						}
					}
				})
			}
		}
	}
}

// jobStep is one item of a job's `steps:` list, named or not: the checkout and
// set-up steps carry only `uses:`, and they occupy a runner like any other.
type jobStep struct {
	label string // its name, or its uses: value
	line  int    // 1-based within the job block
	cond  string // its if: value, "" when it has none
	uses  string // its uses: value, "" for a run: step
	body  string
}

// runs reports whether the step's script runs cmd.
func (s jobStep) runs(cmd string) bool { return strings.Contains(s.script(), cmd) }

// script is the step's run: value, inline or block, or "" for a uses: step.
func (s jobStep) script() string {
	lines := strings.Split(s.body, "\n")
	for i, l := range lines {
		trimmed := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(l), "- "))
		if !strings.HasPrefix(trimmed, "run:") {
			continue
		}
		rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "run:"))
		if rest != "|" && rest != ">" {
			return rest
		}
		return strings.Join(lines[i+1:], "\n")
	}
	return ""
}

// toolchainCommandRe finds a line of a script that invokes the Go toolchain or
// make: the work a stand-in may not do.
var toolchainCommandRe = regexp.MustCompile(`(?m)^\s*(go|make)\s.*$`)

var (
	jobStepStartRe = regexp.MustCompile(`^      - (.*)$`)
	jobStepKeyRe   = regexp.MustCompile(`^(?:      - |        )(name|uses|if):\s*(.*)$`)
)

// jobSteps splits a job block's steps list into its items: each starts at a
// `      - ` line and runs to the next one or to the end of the block. Comment
// lines between steps belong to neither.
func jobSteps(job string) []jobStep {
	lines := strings.Split(job, "\n")
	var out []jobStep
	cur := -1
	for i, l := range lines {
		if jobStepStartRe.MatchString(l) {
			out = append(out, jobStep{line: i + 1})
			cur = len(out) - 1
		}
		if cur < 0 {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(l), "#") {
			continue
		}
		out[cur].body += l + "\n"
		if m := jobStepKeyRe.FindStringSubmatch(l); m != nil {
			v := stripComment(m[2])
			switch m[1] {
			case "name":
				out[cur].label = v
			case "uses":
				out[cur].uses = v
				if out[cur].label == "" {
					out[cur].label = v
				}
			case "if":
				out[cur].cond = v
			}
		}
	}
	return out
}

// jobKey returns a job-level key's single-line value (four-space indent).
func jobKey(job, key string) (string, bool) {
	m := regexp.MustCompile(`(?m)^    ` + regexp.QuoteMeta(key) + `:[ \t]*(\S.*)$`).FindStringSubmatch(job)
	if m == nil {
		return "", false
	}
	return stripComment(m[1]), true
}

// workflowTriggers returns the event names under a workflow's block-form `on:`
// key, in the order written.
func workflowTriggers(workflow string) []string {
	lines := strings.Split(workflow, "\n")
	key := regexp.MustCompile(`^  ([a-z_]+):`)
	var out []string
	for i, l := range lines {
		if l != "on:" {
			continue
		}
		for _, n := range lines[i+1:] {
			if topLevelKeyRe.MatchString(n) {
				break
			}
			if m := key.FindStringSubmatch(n); m != nil {
				out = append(out, m[1])
			}
		}
		break
	}
	return out
}
