package lint_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/actionsexpr"
)

// TestMacOSLegRunsItsSuiteOnlyNightlyAndOnDemand holds ci.yml to where it spends a
// macOS runner, and to what each of its test jobs runs (iss-2610110938282654).
//
// macOS is the account's scarcest runner, five jobs at once, and its suite is the
// slowest thing in the workflow. It ran on every pull request as well as on every
// merge-queue entry, so on 2026-10-11 queue entries waited for a slot behind
// pull-request runs that repeated a local `make preflight`, passed the queue's
// timeout and were removed, and nothing merged for five hours. It then ran on
// every event but pull_request, and on 2026-11-04 queue entries kept dropping
// because GitHub had no macOS runner to give, with no test failing. So it stands
// down on a merge-queue entry too, and runs as shard jobs (iss-2610181757540736)
// on the nightly schedule and by hand; the pre-push hook holds a pushed commit
// to `make preflight` locally. `check (macos-latest)` is an aggregator on an ubuntu
// runner: on a pull request and on a queue entry it reports with no shard having
// run, and says so. Where the suite runs, every shard makes one race-enabled pass
// over its packages (iss-2610110938293273), and the build, vet and the tests that
// assert nothing under -race (iss-2610180542030806) run once, in the first shard.
// The ubuntu job keeps its plain pass and its race lane.
//
// The check is SEMANTIC, like the concurrency and range-base contracts beside
// it: every step's own `if:` and each job's `if:` are evaluated per shard, per
// event the workflow triggers on, and per classifier verdict, so a condition
// rewritten in a spelling nobody anticipated passes and one that merely reads
// right fails. The event set is derived from the workflow's `on:` block. A context
// the job does not have fails the test, which is what catches a step left reading
// `matrix.os` in a job with no such matrix: GitHub would read it as empty and skip
// the step in silence.
func TestMacOSLegRunsItsSuiteOnlyNightlyAndOnDemand(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	const rel = ".github/workflows/ci.yml"
	workflow := readRepoFile(t, root, rel)
	events := workflowTriggers(workflow)
	if !slices.Contains(events, "pull_request") || !slices.Contains(events, "merge_group") {
		t.Fatalf("%s triggers on %q; the trigger parser has lost its shape, or the workflow no "+
			"longer gates pull requests and the merge queue", rel, events)
	}
	// The two events the suite exists for: without them it runs nowhere at all.
	if !slices.Contains(events, "schedule") || !slices.Contains(events, "workflow_dispatch") {
		t.Errorf("%s triggers on %q; the macOS suite runs on the schedule and by hand, so it needs both", rel, events)
	}

	t.Run("contexts", func(t *testing.T) { checkRequiredTestContexts(t, root, workflow) })
	t.Run("ubuntu", func(t *testing.T) { checkUbuntuJobPasses(t, workflow, events) })
	t.Run("suite", func(t *testing.T) { checkMacOSSuitePasses(t, workflow, events) })
}

// checkRequiredTestContexts: the ruleset's two test contexts are each reported by
// exactly the job that should, under a name written out, since a context nobody
// reports wedges the queue and a shard reporting one would read as the whole suite.
func checkRequiredTestContexts(t *testing.T, root, workflow string) {
	t.Helper()
	mirror := readRepoFile(t, root, rulesetMirror)
	for job, context := range map[string]string{"check": "check (ubuntu-latest)", macOSCheckJob: macOSContext} {
		if !strings.Contains(mirror, `"context": "`+context+`"`) {
			t.Errorf("%s does not require %q", rulesetMirror, context)
		}
		block, ok := workflowJobBlock(workflow, job)
		if !ok {
			t.Errorf("ci.yml has no %s job, which reports the required context %q", job, context)
			continue
		}
		if got, _ := jobKey(block, "name"); got != context {
			t.Errorf("ci.yml's %s job is named %q; want %q, the context the ruleset requires", job, got, context)
		}
		if n := strings.Count(workflow, "\n    name: "+context+"\n"); n != 1 {
			t.Errorf("ci.yml names %d jobs %q; want one", n, context)
		}
	}
	suite, ok := workflowJobBlock(workflow, macOSSuiteJob)
	if !ok {
		t.Fatalf("ci.yml has no %s job", macOSSuiteJob)
	}
	if got, ok := jobKey(suite, "name"); ok {
		t.Errorf("ci.yml's %s job is named %q; a skipped matrix job reports its name unexpanded, so it "+
			"keeps the name its id and shard give it", macOSSuiteJob, got)
	}
}

// checkUbuntuJobPasses evaluates the check job per event and classifier verdict.
func checkUbuntuJobPasses(t *testing.T, workflow string, events []string) {
	t.Helper()
	job, ok := workflowJobBlock(workflow, "check")
	if !ok {
		t.Fatal("ci.yml defines no check job")
	}
	if strings.Contains(job, "matrix:") {
		t.Error("the check job carries a matrix; it reports one context, on one runner")
	}
	if r, _ := jobKey(job, "runs-on"); r != "ubuntu-latest" {
		t.Errorf("the check job runs on %q; want ubuntu-latest", r)
	}
	jobIf, _ := jobKey(job, "if")
	items := jobSteps(job)
	if len(items) < 5 {
		t.Fatalf("the check job parsed into %d step(s); the step parser has lost its shape", len(items))
	}
	for _, event := range events {
		// "" is a classifier that wrote no output (it failed, or timed out under
		// the check job's !cancelled()): it reads as not inert, and it decides
		// whether the job tests internal/ under the race detector at all.
		verdicts := []string{"false", ""}
		if event == "pull_request" {
			// Only a pull request can be classified inert; every other event
			// runs the full matrix (the changes job's own contract).
			verdicts = append(verdicts, "true")
		}
		for _, inert := range verdicts {
			t.Run(fmt.Sprintf("%s/inert=%s", event, inert), func(t *testing.T) {
				ctx := map[string]any{
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
				running := runningSteps(t, items, ctx)
				for _, cmd := range []string{"go build ./...", "go vet ./..."} {
					if !slices.ContainsFunc(running, func(s jobStep) bool { return s.runs(cmd) }) {
						t.Errorf("the check job runs no %q on a %s event", cmd, event)
					}
				}
				passes := goTestPasses(running)
				// Where the race lane stands down (an inert change) the plain pass
				// is the job's only one and runs the whole module;
				// TestPlainTestStepRunsTheWholeModule holds its set. Where the race
				// lane runs, it runs every internal test, so the plain passes run
				// only what it cannot assert (iss-2610171803053186): the packages
				// outside internal/, and the tests that assert only without -race,
				// by name.
				plainWhole := slices.ContainsFunc(passes, func(p string) bool {
					return !strings.HasPrefix(p, "go test -race ") && strings.HasSuffix(p, " "+modulePattern)
				})
				hasRace := slices.ContainsFunc(passes, func(p string) bool { return strings.HasPrefix(p, "go test -race ") })
				outside := slices.Contains(passes, "go test "+outsideInternalPattern)
				guards := slices.ContainsFunc(passes, func(p string) bool {
					return !strings.HasPrefix(p, "go test -race ") && strings.HasPrefix(p, "go test -run ")
				})
				if inert == "true" {
					if !plainWhole || hasRace || outside || guards {
						t.Errorf("the check job on an inert %s event runs %q; want one plain "+
							"`go test ... %s` and no race pass", event, passes, modulePattern)
					}
				} else if plainWhole || !hasRace || !outside || !guards {
					t.Errorf("the check job on a %s event that is not inert runs %q; want the race "+
						"pass, `go test %s` and the plain-only guards by name, and no plain "+
						"rerun of the whole module", event, passes, outsideInternalPattern)
				}
			})
		}
	}
}

// macOSStandsDown reports whether the suite takes no macOS runner on event. Every
// other event runs it, an event added later included, so a new trigger cannot read
// as a green for a suite nobody ran.
func macOSStandsDown(event string) bool {
	return event == "pull_request" || event == "merge_group"
}

// checkMacOSSuitePasses evaluates the macOS suite per event and shard: no runner
// on a pull request or a queue entry, and otherwise every shard's race pass and
// each one-off once.
func checkMacOSSuitePasses(t *testing.T, workflow string, events []string) {
	t.Helper()
	job, ok := workflowJobBlock(workflow, macOSSuiteJob)
	if !ok {
		t.Fatalf("ci.yml defines no %s job", macOSSuiteJob)
	}
	if r, _ := jobKey(job, "runs-on"); r != "macos-latest" {
		t.Errorf("the %s job runs on %q; want macos-latest", macOSSuiteJob, r)
	}
	// The classifier has no say here: no pull request reaches a shard, and every
	// other event is never inert. A need on it would only add a way to skip a
	// shard, on the classifier's failure, on the only runs the suite has.
	if _, ok := jobKey(job, "needs"); ok {
		t.Errorf("the %s job carries a needs; a failed dependency would skip the suite", macOSSuiteJob)
	}
	if f, _ := regexpFirst(job, `(?m)^      fail-fast: (\S+)\s*$`); f != "true" {
		t.Errorf("the %s job's fail-fast is %q; want true: a shard that failed has decided the run, and "+
			"the rest would hold scarce runners for an answer already known", macOSSuiteJob, f)
	}
	jobIf, ok := jobKey(job, "if")
	if !ok {
		t.Fatalf("the %s job carries no if, so a pull request and a queue entry would take macOS runners", macOSSuiteJob)
	}
	shards := macOSShards(t, job)
	items := jobSteps(job)
	for _, event := range events {
		t.Run(event, func(t *testing.T) {
			ctx := map[string]any{
				"github.event_name": event,
				"success()":         true,
				"failure()":         false,
				"cancelled()":       false,
			}
			run, err := actionsexpr.EvalIf(jobIf, ctx)
			if err != nil {
				t.Fatalf("cannot evaluate the job's if %q: %v", jobIf, err)
			}
			if want := !macOSStandsDown(event); run != want {
				t.Fatalf("the %s job's if %q reads %v on a %s event; want %v", macOSSuiteJob, jobIf, run, event, want)
			}
			if !run {
				return
			}
			count := map[string]int{}
			for _, shard := range shards {
				ctx["matrix.shard"] = float64(shard)
				running := runningSteps(t, items, ctx)
				passes := goTestPasses(running)
				if len(passes) == 0 || passes[0] != "go test -race -timeout 40m "+macOSShardPackages {
					t.Errorf("shard %d on a %s event runs %q; want its race pass first, so the slowest "+
						"packages start with the shard's testing", shard, event, passes)
				}
				for i, p := range passes {
					switch {
					case i == 0:
					case strings.HasPrefix(p, "go test -run "):
						count["guards"]++
					default:
						t.Errorf("shard %d on a %s event also runs %q; a shard makes one race pass, "+
							"and the plain-only guards by name", shard, event, p)
					}
				}
				for _, cmd := range []string{"go build ./...", "go vet ./..."} {
					if slices.ContainsFunc(running, func(s jobStep) bool { return s.runs(cmd) }) {
						count[cmd]++
					}
				}
			}
			// Everything the single job did besides its race pass, once.
			for _, what := range []string{"go build ./...", "go vet ./...", "guards"} {
				if count[what] != 1 {
					t.Errorf("the macOS suite runs %s in %d shards on a %s event; want exactly one", what, count[what], event)
				}
			}
		})
	}
}

// runningSteps evaluates each step's if: in ctx, failing the test on one it cannot.
func runningSteps(t *testing.T, items []jobStep, ctx map[string]any) []jobStep {
	t.Helper()
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
	return running
}

// goTestPasses returns the `go test` lines the steps run, in order.
func goTestPasses(steps []jobStep) []string {
	var passes []string
	for _, s := range steps {
		for _, l := range strings.Split(s.script(), "\n") {
			if l = strings.TrimSpace(l); strings.HasPrefix(l, "go test ") {
				passes = append(passes, l)
			}
		}
	}
	return passes
}

// regexpFirst returns the first capture of pattern in text.
func regexpFirst(text, pattern string) (string, bool) {
	m := regexp.MustCompile(pattern).FindStringSubmatch(text)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// macOSContext is the required context the macOS suite's aggregator reports.
const macOSContext = "check (macos-latest)"

// macOSShards reads the suite's matrix, failing unless it is 1..n: the step asks
// cmd/ci-shard for shard matrix.shard of strategy.job-total, so a gap or a repeat
// here would leave a shard's packages tested nowhere.
func macOSShards(t *testing.T, suite string) []int {
	t.Helper()
	raw, ok := regexpFirst(suite, `(?m)^        shard: \[([^\]]*)\]\s*$`)
	if !ok {
		t.Fatalf("the %s job declares no `shard: [...]` matrix", macOSSuiteJob)
	}
	var shards []int
	for i, f := range strings.Split(raw, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(f))
		if err != nil || n != i+1 {
			t.Fatalf("the %s job's matrix is [%s]; want 1, 2, ... n, with no gap", macOSSuiteJob, raw)
		}
		shards = append(shards, n)
	}
	if len(shards) != macOSShardCount {
		t.Fatalf("the %s job runs %d shards; macOSShardCount says %d, the count its budget is sized on",
			macOSSuiteJob, len(shards), macOSShardCount)
	}
	return shards
}

// macOSShardCount is how many shards the suite runs: a third of the slowest
// passing run fits a slow runner comfortably, and one run's three shards fit the
// account's five macOS runners (the comment on the macos-suite job in ci.yml has
// the sums).
const macOSShardCount = 3

// macOSRaceStepScript returns the shard race step's job step, its env values held
// to what asks cmd/ci-shard for this shard of as many as the matrix has.
func macOSRaceStepScript(t *testing.T, suite string) jobStep {
	t.Helper()
	for _, s := range jobSteps(suite) {
		if s.label != macOSRaceStep {
			continue
		}
		for key, want := range map[string]string{
			"ABCD_CI_SHARD":  "${{ matrix.shard }}",
			"ABCD_CI_SHARDS": "${{ strategy.job-total }}",
		} {
			if got, _ := regexpFirst(s.body, `(?m)^\s+`+key+`: (.+?)\s*$`); got != want {
				t.Errorf("step %q: %s is %q; want %q, so the count cannot drift from the matrix", macOSRaceStep, key, got, want)
			}
		}
		if s.cond != "" {
			t.Errorf("step %q carries `if: %s`; every shard runs its race pass", macOSRaceStep, s.cond)
		}
		return s
	}
	t.Fatalf("the %s job has no %q step", macOSSuiteJob, macOSRaceStep)
	return jobStep{}
}

// TestMacOSSuiteShardsCoverTheModule runs what the workflow itself asks for, shard
// by shard from its own matrix, and holds the union to the module's packages and
// the shards to disjoint (iss-2610181757540736). cmd/ci-shard's own tests hold its
// assignment; this one holds the matrix and the step to asking for all of it.
func TestMacOSSuiteShardsCoverTheModule(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	suite, ok := workflowJobBlock(readRepoFile(t, root, ".github/workflows/ci.yml"), macOSSuiteJob)
	if !ok {
		t.Fatalf("ci.yml defines no %s job", macOSSuiteJob)
	}
	shards := macOSShards(t, suite)
	step := macOSRaceStepScript(t, suite)
	const assign = `pkgs=$(go run ./cmd/ci-shard -shard "$ABCD_CI_SHARD" -of "$ABCD_CI_SHARDS")`
	if !strings.Contains(step.script(), assign) {
		t.Fatalf("step %q does not read %s:\n%s", macOSRaceStep, assign, step.script())
	}

	// The packages as a shard's go test -race resolves them.
	list := exec.Command("go", "list", "-tags", "race", modulePattern)
	list.Dir = root
	out, err := list.Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	want := strings.Fields(string(out))
	seen := map[string]int{}
	for _, shard := range shards {
		cmd := exec.Command("go", "run", "./cmd/ci-shard", "-shard", strconv.Itoa(shard), "-of", strconv.Itoa(len(shards)))
		cmd.Dir = root
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("shard %d: %v", shard, err)
		}
		got := strings.Fields(string(out))
		if len(got) == 0 {
			t.Errorf("shard %d names no packages", shard)
		}
		for _, p := range got {
			seen[p]++
		}
	}
	for _, p := range want {
		if seen[p] != 1 {
			t.Errorf("%s is in %d shards; want exactly one", p, seen[p])
		}
	}
	for p := range seen {
		if !slices.Contains(want, p) {
			t.Errorf("a shard names %s, which go list does not", p)
		}
	}
}

// TestAShardThatCannotReadItsPackagesFails runs the race step's own script against
// a stub go: a helper that failed must fail the step before go test runs, because
// go test with no packages tests the working directory and reports success. On the
// other side, the packages it printed must reach go test as separate words.
func TestAShardThatCannotReadItsPackagesFails(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	suite, ok := workflowJobBlock(readRepoFile(t, root, ".github/workflows/ci.yml"), macOSSuiteJob)
	if !ok {
		t.Fatalf("ci.yml defines no %s job", macOSSuiteJob)
	}
	script := macOSRaceStepScript(t, suite).script()
	dir := t.TempDir()
	stub := "#!/bin/sh\nprintf '%s\\n' \"$*\" >>\"$STUB_LOG\"\n" +
		"[ \"$1\" = run ] || exit 0\n[ -z \"$STUB_FAIL\" ] || exit 1\nprintf 'p/a\\np/b\\n'\n"
	if err := os.WriteFile(filepath.Join(dir, "go"), []byte(stub), 0o755); err != nil {
		t.Fatal(err)
	}
	run := func(fail string) (string, error) {
		log := filepath.Join(dir, "log"+fail)
		// The flags the step's own shell gives bash, ci-watchdog aside.
		cmd := exec.Command("bash", "--noprofile", "--norc", "-eo", "pipefail", "-c", script)
		cmd.Env = append(os.Environ(), "PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"),
			"STUB_LOG="+log, "STUB_FAIL="+fail, "ABCD_CI_SHARD=1", "ABCD_CI_SHARDS=3")
		err := cmd.Run()
		got, _ := os.ReadFile(log)
		return string(got), err
	}
	if log, err := run("yes"); err == nil {
		t.Errorf("the step passed with cmd/ci-shard failing; it ran:\n%s", log)
	} else if strings.Contains(log, "test") {
		t.Errorf("the step ran go test with cmd/ci-shard failing:\n%s", log)
	}
	log, err := run("")
	if err != nil {
		t.Fatalf("the step failed with a working cmd/ci-shard: %v\n%s", err, log)
	}
	if !strings.Contains(log, "test -race -timeout 40m p/a p/b\n") {
		t.Errorf("the step did not hand go test the shard's packages as separate words:\n%s", log)
	}
}

// TestMacOSCheckFailsUnlessEveryShardSucceeded runs the aggregator's own script
// over every result its need can read, per event (loud staging: a green may only
// stand for work that happened). It passes in exactly two cases: every shard
// succeeded on an event the suite runs on, and the shards were deliberately skipped
// on a pull request or a queue entry, where it says that no macOS suite ran and
// where one does. A shard skipped anywhere else, failed, cancelled or abandoned, or
// a result that reads as nothing at all, fails it. needs.<job>.result is a matrix's
// result as a whole: GitHub reads it as success only when every leg succeeded, and
// as abandoned when a shard never got a runner (run 41901793293).
//
// It must also report whatever happened to the shards and the run, so the job and
// its step both read always(): a job's step with the implicit success() is skipped
// when the run has been cancelled, and a job whose steps are all skipped concludes
// success, which is a green for a suite that never finished.
func TestMacOSCheckFailsUnlessEveryShardSucceeded(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	workflow := readRepoFile(t, root, ".github/workflows/ci.yml")
	job, ok := workflowJobBlock(workflow, macOSCheckJob)
	if !ok {
		t.Fatalf("ci.yml defines no %s job, which reports the required %s", macOSCheckJob, macOSContext)
	}
	for key, want := range map[string]string{
		"needs":       macOSSuiteJob,
		"if":          "always()",
		"runs-on":     "ubuntu-latest",
		"permissions": "{}",
	} {
		if got, _ := jobKey(job, key); got != want {
			t.Errorf("the %s job's %s is %q; want %q", macOSCheckJob, key, got, want)
		}
	}
	steps := jobSteps(job)
	if len(steps) != 1 {
		t.Fatalf("the %s job has %d steps; want the one that reads the shards' result", macOSCheckJob, len(steps))
	}
	step := steps[0]
	if step.cond != "always()" {
		t.Errorf("the %s job's step reads `if: %s`; want always(): a step skipped on a cancelled run leaves "+
			"a job that concludes success", macOSCheckJob, step.cond)
	}
	if step.uses != "" {
		t.Errorf("the %s job's step uses %s; it reads one result and takes no action", macOSCheckJob, step.uses)
	}
	script := step.script()
	if m := toolchainCommandRe.FindString(script); m != "" {
		t.Errorf("the %s job's step runs %q; it runs nothing but its verdict", macOSCheckJob, strings.TrimSpace(m))
	}
	if strings.Contains(script, "${{") {
		t.Errorf("the %s job's step interpolates an expression inside run:; pass values through env", macOSCheckJob)
	}
	for key, want := range map[string]string{
		"EVENT_NAME":   "${{ github.event_name }}",
		"SUITE_RESULT": "${{ needs." + macOSSuiteJob + ".result }}",
	} {
		if got, _ := regexpFirst(step.body, `(?m)^\s+`+key+`: (.+?)\s*$`); got != want {
			t.Errorf("the %s job's step: %s is %q; want %q", macOSCheckJob, key, got, want)
		}
	}

	results := []string{"success", "failure", "cancelled", "skipped", "abandoned", "", "neutral"}
	// What each stand-in has to say, in the log and the step summary: a green here
	// tested nothing on macOS, and the reader is told where the suite does run.
	standIn := map[string][]string{
		"pull_request": {"this run tested nothing on macOS", "nightly", "on demand"},
		"merge_group":  {"No macOS suite ran on this entry", "The nightly run covers the default branch"},
	}
	for _, event := range append(workflowTriggers(workflow), "an-event-added-later") {
		for _, result := range results {
			t.Run(fmt.Sprintf("%s/%s", event, result), func(t *testing.T) {
				want := result == "success"
				if macOSStandsDown(event) {
					want = result == "skipped"
				}
				summary := filepath.Join(t.TempDir(), "summary")
				// The flags the runner gives bash for a step that names no shell.
				cmd := exec.Command("bash", "-e", "-c", script)
				cmd.Env = append(os.Environ(), "EVENT_NAME="+event, "SUITE_RESULT="+result, "GITHUB_STEP_SUMMARY="+summary)
				out, err := cmd.CombinedOutput()
				if pass := err == nil; pass != want {
					t.Errorf("exit %v on a %s event with result %q; want pass=%v:\n%s", err, event, result, want, out)
				}
				if !want && !strings.Contains(string(out), "::error") {
					t.Errorf("a refusal says nothing on the log:\n%s", out)
				}
				if !macOSStandsDown(event) || !want {
					return
				}
				if !strings.Contains(string(out), "::notice") {
					t.Errorf("the %s stand-in raises no notice in the log:\n%s", event, out)
				}
				written, err := os.ReadFile(summary)
				if err != nil {
					t.Errorf("the %s stand-in writes no step summary: %v", event, err)
				}
				for _, says := range standIn[event] {
					if !strings.Contains(string(out), says) || !strings.Contains(string(written), says) {
						t.Errorf("the %s stand-in does not say %q in both the log and the summary:\n%s\nsummary:\n%s",
							event, says, out, written)
					}
				}
			})
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
