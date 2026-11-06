package lint_test

import (
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

// A hang in a job that runs go test fails with a goroutine dump before the job's
// cap cancels it.
//
// go test's -timeout counts from each test binary's own start, so it cannot
// bound the job's clock: on a passing merge-group run the ubuntu job's last
// package binary started 23.8 minutes into the job, so a hang there would have
// met the job's 60-minute cap before its 40m -timeout fired, and a
// cancellation at the cap carries no dump. No single -timeout fits every package
// a job starts at a different time.
//
// So every step that runs go test, in ci.yml's check job and in each shard of the
// macOS suite (macos-suite.yml), runs it under cmd/ci-watchdog, the job clock's watchdog. It starts
// the step's shell as a process group it owns and, at the job's cap less the
// declared margin on the clock the "Record the job's start" step sets, sends the
// test binaries under go test SIGQUIT (each prints its goroutines and fails), then
// the group SIGKILL after a grace period. The package -timeout stays the
// per-package budget make preflight shares
// (TestRaceLaneBudgetIsDeclaredAndFitsItsJob).
//
// The margin is declared once in each workflow's env, and the cap the watchdog
// reads is held to each job's timeout-minutes, so the deadline sits inside every
// job's cap by the margin. No job outside ciTestJobs may run go test at all: it
// would have no watchdog and this test would not read it.
func TestCheckJobTestStepsRunUnderTheJobClockWatchdog(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	// A caller's workflow-level env does not reach a called workflow, so each file
	// declares the margin once, and the two declare one value.
	margins := map[string]time.Duration{}
	for _, file := range []string{".github/workflows/ci.yml", macOSSuiteWorkflow} {
		head, jobsPart, ok := strings.Cut(readRepoFile(t, root, file), "\njobs:\n")
		if !ok {
			t.Fatalf("%s: no `jobs:` key; the parser or the workflow changed shape", file)
		}
		if m := regexp.MustCompile(`(?m)^  ` + watchdogMarginVar + `: (.+?)\s*$`).FindStringSubmatch(head); m == nil {
			t.Errorf("%s: the workflow's env declares no %s, which the watchdog reads", file, watchdogMarginVar)
		} else {
			n, err := strconv.Atoi(strings.Trim(m[1], `"`))
			margins[file] = time.Duration(n) * time.Minute
			switch {
			case err != nil:
				t.Errorf("%s: %s is %q, not a whole number of minutes", file, watchdogMarginVar, m[1])
			case margins[file] < watchdogMarginFloor:
				t.Errorf("%s: %s is %s, under %s: the dumps, the grace before SIGKILL and "+
					"the job's remaining steps must finish inside it", file, watchdogMarginVar, margins[file], watchdogMarginFloor)
			}
		}
		if strings.Contains(jobsPart, watchdogMarginVar+":") {
			t.Errorf("%s: a job declares %s of its own; it is declared once, in the workflow's env", file, watchdogMarginVar)
		}
	}
	if a, b := margins[".github/workflows/ci.yml"], margins[macOSSuiteWorkflow]; a != b {
		t.Errorf("ci.yml's %s is %s and %s's is %s; want one margin", watchdogMarginVar, a, macOSSuiteWorkflow, b)
	}

	for _, testJob := range ciTestJobs {
		name := testJob.job
		margin := margins[testJob.file]
		where := testJob.file + " job " + name
		job, ok := workflowJobBlock(readRepoFile(t, root, testJob.file), name)
		if !ok {
			t.Fatalf("%s: no such job; the parser or the workflow changed shape", where)
		}
		env := func(key string) (string, bool) {
			m := regexp.MustCompile(`(?m)^      ` + regexp.QuoteMeta(key) + `: (.+?)\s*$`).FindStringSubmatch(job)
			if m == nil {
				t.Errorf("%s: the job's env declares no %s, which the watchdog reads", where, key)
				return "", false
			}
			return m[1], true
		}
		capMin := jobTimeoutMinutes(t, where, job)
		if raw, ok := env(watchdogCapVar); ok {
			n, err := strconv.Atoi(strings.Trim(raw, `"`))
			if got := time.Duration(n) * time.Minute; err != nil || got != capMin {
				t.Errorf("%s: %s is %s, but its timeout-minutes is %s; the watchdog times its deadline "+
					"from the cap the runner enforces", where, watchdogCapVar, raw, capMin)
			}
		}
		if margin >= capMin {
			t.Errorf("%s: %s (%s) leaves its cap of %s no deadline", where, watchdogMarginVar, margin, capMin)
		}
		env(watchdogLegVar)

		steps := jobSteps(job)
		clockLine := 0
		for _, step := range steps {
			if step.label != checkClockStep {
				continue
			}
			clockLine = step.line
			if !regexp.MustCompile(`(?m)\b` + watchdogStartedVar + `=.*>>\s*"\$GITHUB_ENV"\s*$`).MatchString(step.body) {
				t.Errorf("%s: the `%s` step does not export %s to $GITHUB_ENV, so the watchdog has no job clock",
					where, checkClockStep, watchdogStartedVar)
			}
		}
		if clockLine == 0 {
			t.Fatalf("%s: no `%s` step", where, checkClockStep)
		}

		// Every step whose script names go test, named or not, wherever on a line.
		shell := regexp.MustCompile(`(?m)^\s+shell: ` + regexp.QuoteMeta(watchdogShell) + `\s*$`)
		found := 0
		for _, step := range steps {
			if !goTestRe.MatchString(step.body) {
				continue
			}
			found++
			label := step.label
			if label == "" {
				label = "the step at line " + strconv.Itoa(step.line) + " of the job"
			}
			if step.line < clockLine {
				t.Errorf("%s: step %q runs before `%s`, so the job clock is not set for it", where, label, checkClockStep)
			}
			if !shell.MatchString(step.body) {
				t.Errorf("%s: step %q runs go test outside the job clock's watchdog; give it `shell: %s`, "+
					"or a hang there meets the job's cap before its -timeout and is cancelled with no dump",
					where, label, watchdogShell)
			}
		}
		if found == 0 {
			t.Fatalf("%s: found no step that runs go test; the parser or the workflow changed shape", where)
		}
	}

	// Comments between steps belong to no step, so only a step's own text counts.
	for _, file := range []string{".github/workflows/ci.yml", macOSSuiteWorkflow} {
		workflow := readRepoFile(t, root, file)
		for _, id := range workflowJobIDs(workflow) {
			if slices.Contains(ciTestJobs, ciTestJob{file, id}) {
				continue
			}
			job, _ := workflowJobBlock(workflow, id)
			for _, step := range jobSteps(job) {
				if goTestRe.MatchString(step.body) {
					t.Errorf("%s job %s: step %q runs go test, in a job with no job clock "+
						"for the watchdog; add the job to ciTestJobs and give it the clock", file, id, step.label)
				}
			}
		}
	}
}

// goTestRe finds a step that runs go test.
var goTestRe = regexp.MustCompile(`\bgo test\b`)

// The watchdog settings a test job's env declares for cmd/ci-watchdog; the margin
// is the workflow's, one value for every job.
const (
	watchdogLegVar     = "ABCD_CI_LEG"
	watchdogCapVar     = "ABCD_CI_JOB_CAP_MINUTES"
	watchdogMarginVar  = "ABCD_CI_WATCHDOG_MARGIN_MINUTES"
	watchdogStartedVar = "ABCD_CI_JOB_STARTED"
)

// watchdogShell is the shell every step of a test job that runs go test declares:
// the runner's own bash, started by the job clock's watchdog.
const watchdogShell = "go run ./cmd/ci-watchdog bash --noprofile --norc -eo pipefail {0}"

// watchdogMarginFloor is the least margin the watchdog may leave before the
// cap: its 30-second grace between SIGQUIT and SIGKILL, then the steps that
// run whatever the test step's outcome (the warning step) and the runner's
// post-job steps.
const watchdogMarginFloor = 3 * time.Minute
