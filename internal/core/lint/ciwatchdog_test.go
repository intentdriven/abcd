package lint_test

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// A hang in the check job fails with a goroutine dump before the job's cap
// cancels it.
//
// go test's -timeout counts from each test binary's own start, so it cannot
// bound the job's clock: on a passing merge-group run the ubuntu leg's last
// package binary started 23.8 minutes into the job and the macOS leg's 66.0, so
// a hang there would have met the 45- and 85-minute caps long before its 40m
// -timeout fired, and a cancellation at the cap carries no dump. No single
// -timeout fits: on the macOS leg a late package would need 19 minutes or less,
// and internal/core/implement/loop alone ran 19.1 there.
//
// So every check-job step that runs go test runs it under cmd/ci-watchdog, the
// job clock's watchdog. It starts the step's shell as a process group it owns
// and, at the leg's cap less the job's declared margin on the clock the
// "Record the job's start" step sets, sends the test binaries under go test
// SIGQUIT (each prints its goroutines and fails), then the group SIGKILL after
// a grace period. The package -timeout stays the per-package budget make
// preflight shares (TestRaceLaneBudgetIsDeclaredAndFitsItsJob).
//
// The margin is declared once, as the job's ABCD_CI_WATCHDOG_MARGIN_MINUTES,
// and the cap the watchdog reads is held to each leg's timeout-minutes, so the
// deadline sits inside every leg's cap by the margin.
func TestCheckJobTestStepsRunUnderTheJobClockWatchdog(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	const where = ".github/workflows/ci.yml job check"
	job, ok := workflowJobBlock(readRepoFile(t, root, ".github/workflows/ci.yml"), "check")
	if !ok {
		t.Fatalf("%s: no such job; the parser or the workflow changed shape", where)
	}

	env := func(name string) (string, bool) {
		m := regexp.MustCompile(`(?m)^      ` + regexp.QuoteMeta(name) + `: (.+?)\s*$`).FindStringSubmatch(job)
		if m == nil {
			t.Errorf("%s: the job's env declares no %s, which the watchdog reads", where, name)
			return "", false
		}
		return m[1], true
	}

	legs := checkLegTimeouts(t, where, job)
	var oses []string
	for _, leg := range legs {
		oses = append(oses, leg.os)
	}
	if raw, ok := env(watchdogCapVar); ok {
		caps := legValues(t, where, watchdogCapVar, raw, oses)
		for _, leg := range legs {
			if got := time.Duration(caps[leg.os]) * time.Minute; got != leg.minutes {
				t.Errorf("%s: %s gives the %s leg %s, but its timeout-minutes is %s; the watchdog "+
					"times its deadline from the cap the runner enforces", where, watchdogCapVar, leg.os, got, leg.minutes)
			}
		}
	}
	if raw, ok := env(watchdogMarginVar); ok {
		n, err := strconv.Atoi(strings.Trim(raw, `"`))
		margin := time.Duration(n) * time.Minute
		switch {
		case err != nil:
			t.Errorf("%s: %s is %q, not a whole number of minutes", where, watchdogMarginVar, raw)
		case margin < watchdogMarginFloor:
			t.Errorf("%s: %s is %s, under %s: the dumps, the grace before SIGKILL and the job's "+
				"remaining steps must finish inside it", where, watchdogMarginVar, margin, watchdogMarginFloor)
		default:
			for _, leg := range legs {
				if margin >= leg.minutes {
					t.Errorf("%s: %s (%s) leaves the %s leg (cap %s) no deadline", where, watchdogMarginVar, margin, leg.os, leg.minutes)
				}
			}
		}
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
	goTest := regexp.MustCompile(`\bgo test\b`)
	shell := regexp.MustCompile(`(?m)^\s+shell: ` + regexp.QuoteMeta(watchdogShell) + `\s*$`)
	found := 0
	for _, step := range steps {
		if !goTest.MatchString(step.body) {
			continue
		}
		found++
		name := step.label
		if name == "" {
			name = "the step at line " + strconv.Itoa(step.line) + " of the job"
		}
		if step.line < clockLine {
			t.Errorf("%s: step %q runs before `%s`, so the job clock is not set for it", where, name, checkClockStep)
		}
		if !shell.MatchString(step.body) {
			t.Errorf("%s: step %q runs go test outside the job clock's watchdog; give it `shell: %s`, "+
				"or a hang there meets the job's cap before its -timeout and is cancelled with no dump",
				where, name, watchdogShell)
		}
	}
	if found == 0 {
		t.Fatalf("%s: found no step that runs go test; the parser or the workflow changed shape", where)
	}
}

// The check job's watchdog settings, as its env declares them for
// cmd/ci-watchdog.
const (
	watchdogLegVar     = "ABCD_CI_LEG"
	watchdogCapVar     = "ABCD_CI_JOB_CAP_MINUTES"
	watchdogMarginVar  = "ABCD_CI_WATCHDOG_MARGIN_MINUTES"
	watchdogStartedVar = "ABCD_CI_JOB_STARTED"
)

// watchdogShell is the shell every check-job step that runs go test declares:
// the runner's own bash, started by the job clock's watchdog.
const watchdogShell = "go run ./cmd/ci-watchdog bash --noprofile --norc -eo pipefail {0}"

// watchdogMarginFloor is the least margin the watchdog may leave before the
// cap: its 30-second grace between SIGQUIT and SIGKILL, then the steps that
// run whatever the test step's outcome (the warning step) and the runner's
// post-job steps.
const watchdogMarginFloor = 3 * time.Minute
