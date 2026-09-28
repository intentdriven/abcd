package lint_test

import (
	"encoding/json"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The race lane's time budget is declared, not inherited (iss-2609260319483365).
//
// `go test` kills a test binary that runs past -timeout, and with no flag the
// ceiling is ten minutes per package. internal/surface/cli under -race on the
// macOS runner measured 417.9s, 472.0s and 567.9s on three passing runs, and
// a merge-group run on a slow runner crossed 600s with no test hung and ejected
// an unrelated pull request from the queue. The lane's ceiling was the default
// rather than a budget anyone had sized, so it is written down here, in three
// places that must agree: the merge gate (ci.yml), the release gate
// (release.yml's verify job) and the local pre-push gate (make preflight).
//
// The enclosing ceilings are the other half, and they differ by job.
//
// ci.yml's check job is a required context, and on a merge-group run the merge
// queue fails the group when a required check has not concluded within the
// ruleset's check_response_timeout_minutes. That cap is the outer ceiling on
// the event the lane was ejected on, and a job ceiling above it is unreachable
// there: the queue gives up first, with no job failure to read. So no leg of
// the check job may carry a timeout-minutes above the cap the ruleset mirror
// records. Under ruling Z (2026-09-28, the technical facilitator) the cap is 45
// minutes and the macOS leg follows it, while the ubuntu leg keeps 30; the job
// declares that as one matrix expression, which checkLegTimeouts reads leg by
// leg. The step's -timeout still earns its place under that cap: it lifts the
// ten-minute per-package default, so a slow package fails on the job's clock
// rather than at ten minutes. Raising the queue's cap is an admin act on the
// live ruleset, left to the technical facilitator; this test reads the mirror,
// so a job ceiling can rise only after the mirror records the raise.
//
// release.yml's verify job is not a merge-queue job, so its own timeout-minutes
// is its ceiling. A package timeout that reaches past it never fires: the
// runner cancels the job first, and a cancellation carries no goroutine dump,
// so a real hang would read as a slow runner. verify therefore holds the
// package timeout plus the time its slowest package waits before it starts:
// the steps ahead of the race step, and the packages `go test` runs ahead of
// it inside the step.
func TestRaceLaneBudgetIsDeclaredAndFitsItsJob(t *testing.T) {
	root := filepath.Join("..", "..", "..")

	recipe, ok := makeRecipe(readRepoFile(t, root, "Makefile"), "preflight")
	if !ok {
		t.Fatal("Makefile declares no `preflight:` recipe")
	}
	local := raceTimeout(t, "Makefile preflight", recipe)

	// raceJob returns a workflow job's block and the -timeout its race step
	// carries, holding that timeout to the local gate's.
	raceJob := func(file, job string) (string, time.Duration, bool) {
		where := file + " job " + job
		block, ok := workflowJobBlock(readRepoFile(t, root, file), job)
		if !ok {
			t.Errorf("%s: no such job; the parser or the workflow changed shape", where)
			return "", 0, false
		}
		step, ok := workflowStepBlock(block, "Test (race, internal)")
		if !ok {
			t.Errorf("%s: no `Test (race, internal)` step", where)
			return "", 0, false
		}
		pkg := raceTimeout(t, where, step)
		if pkg != local {
			t.Errorf("%s runs the race lane under -timeout %s but make preflight uses %s; "+
				"local and CI must judge the lane against one budget", where, pkg, local)
		}
		return block, pkg, true
	}

	if check, _, ok := raceJob(".github/workflows/ci.yml", "check"); ok {
		const where = ".github/workflows/ci.yml job check"
		queue := mergeQueueResponseTimeout(t, root)
		for _, leg := range checkLegTimeouts(t, where, check) {
			if leg.minutes > queue {
				t.Errorf("%s: the %s leg's timeout-minutes is %s, above the merge queue's %s check "+
					"response timeout in %s; on a merge-group run the queue fails the group first, "+
					"so the job ceiling is unreachable there and any budget sized against it is "+
					"false. Raise the live ruleset's cap (an admin act) before the job's",
					where, leg.os, leg.minutes, queue, rulesetMirror)
			}
			// Ruling Z raised the macOS leg alone; every other leg keeps 30.
			if leg.os != "macos-latest" && leg.minutes != 30*time.Minute {
				t.Errorf("%s: the %s leg's timeout-minutes is %s; ruling Z (2026-09-28) raised only "+
					"the macOS leg, and every other leg of the check job stays at 30m",
					where, leg.os, leg.minutes)
			}
		}
	}

	// The last release (run 35963282477, ubuntu, uncached toolchain): 2.8
	// minutes before the race step, 4.8 inside it before internal/surface/cli
	// started, 0.5 after it; the ubuntu race step has grown by about two
	// minutes since. Rounded up to 10.
	const verifyHeadroom = 10 * time.Minute
	if verify, pkg, ok := raceJob(".github/workflows/release.yml", "verify"); ok {
		const where = ".github/workflows/release.yml job verify"
		capMin := jobTimeoutMinutes(t, where, verify)
		if need := pkg + verifyHeadroom; capMin < need {
			t.Errorf("%s: timeout-minutes is %s, below the package timeout %s plus %s of headroom (%s); "+
				"the runner would cancel the job before go test could report a hang",
				where, capMin, pkg, verifyHeadroom, need)
		}
	}
}

// checkWarnAfterMinutes is where the macOS leg of ci.yml's check job warns that
// it is nearing its ceiling: ruling Z's cancel policy (2026-09-28) sets it at 35
// minutes, ten under the 45-minute cap, so a speed lane opens before the queue
// cancels anything.
const checkWarnAfterMinutes = 35

// checkWarnStep is the name of that warning step in the check job.
const checkWarnStep = "Warn when the check nears the queue's cap"

// The macOS leg warns before the merge queue's cap cancels it (ruling Z's
// cancel policy, 2026-09-28). A cancellation at the cap carries nothing but the
// cancellation, and the rule for it is rerun once, then stop that pull request
// and open a speed lane; the warning moves that signal ahead of the first
// cancellation. It is a warning and nothing else: it may not fail the job, it
// runs however the steps before it ended, and its threshold sits below the
// leg's ceiling, or it could never fire before the cancellation it announces.
func TestCheckJobWarnsBeforeTheQueueCap(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	const where = ".github/workflows/ci.yml job check"
	job, ok := workflowJobBlock(readRepoFile(t, root, ".github/workflows/ci.yml"), "check")
	if !ok {
		t.Fatalf("%s: no such job; the parser or the workflow changed shape", where)
	}
	step, ok := workflowStepBlock(job, checkWarnStep)
	if !ok {
		t.Fatalf("%s: no %q step; the macOS leg would reach the merge queue's cap with no "+
			"warning ahead of the cancellation", where, checkWarnStep)
	}

	m := regexp.MustCompile(`(?m)^\s+WARN_AFTER_MINUTES: "?(\d+)"?\s*$`).FindStringSubmatch(step)
	if m == nil {
		t.Fatalf("%s step %q: no WARN_AFTER_MINUTES in its env", where, checkWarnStep)
	}
	warn, _ := strconv.Atoi(m[1])
	if warn != checkWarnAfterMinutes {
		t.Errorf("%s step %q warns after %d minutes; the cancel policy of ruling Z sets %d",
			where, checkWarnStep, warn, checkWarnAfterMinutes)
	}
	warnAfter := time.Duration(warn) * time.Minute
	for _, leg := range checkLegTimeouts(t, where, job) {
		if leg.os == "macos-latest" && warnAfter >= leg.minutes {
			t.Errorf("%s step %q warns after %s, not below the macos-latest leg's %s ceiling; "+
				"the job is cancelled before the warning can fire", where, checkWarnStep, warnAfter, leg.minutes)
		}
	}
	if queue := mergeQueueResponseTimeout(t, root); warnAfter >= queue {
		t.Errorf("%s step %q warns after %s, not below the merge queue's %s cap in %s",
			where, checkWarnStep, warnAfter, queue, rulesetMirror)
	}

	ifLine := regexp.MustCompile(`(?m)^\s+if: (.*)$`).FindStringSubmatch(step)
	if ifLine == nil || !strings.Contains(ifLine[1], "always()") || !strings.Contains(ifLine[1], "macos-latest") {
		t.Errorf("%s step %q: want an `if:` naming always() and the macos-latest leg, so it runs "+
			"on that leg however the steps before it ended", where, checkWarnStep)
	}
	if !regexp.MustCompile(`(?m)^\s+continue-on-error: true\s*$`).MatchString(step) {
		t.Errorf("%s step %q: want `continue-on-error: true`; a warning may never fail the job", where, checkWarnStep)
	}
	at := strings.Index(step, "run:")
	if at < 0 {
		t.Fatalf("%s step %q has no run: script", where, checkWarnStep)
	}
	run := step[at:]
	for _, want := range []string{"::warning", "GITHUB_STEP_SUMMARY", "rerun", "speed lane"} {
		if !strings.Contains(run, want) {
			t.Errorf("%s step %q: its script never mentions %q; the warning names the rerun-once "+
				"rule and the speed lane, in the log and the step summary", where, checkWarnStep, want)
		}
	}
	if strings.Contains(run, "${{") {
		t.Errorf("%s step %q interpolates an expression inside run:; pass values through env", where, checkWarnStep)
	}
}

// checkLeg is one matrix leg's job ceiling.
type checkLeg struct {
	os      string
	minutes time.Duration
}

// checkLegTimeouts reads the check job's `timeout-minutes:` per matrix leg. The
// value is a plain number, which every leg shares, or the one expression
// `${{ matrix.os == '<os>' && <n> || <m> }}`, which gives <os> its own ceiling
// and every other leg <m>. Legs come from the job's `os: [...]` matrix line.
func checkLegTimeouts(t *testing.T, where, job string) []checkLeg {
	t.Helper()
	osLine := regexp.MustCompile(`(?m)^\s+os: \[([^\]]*)\]\s*$`).FindStringSubmatch(job)
	if osLine == nil {
		t.Fatalf("%s: no `os: [...]` matrix line", where)
	}
	var legs []checkLeg
	for _, name := range strings.Split(osLine[1], ",") {
		legs = append(legs, checkLeg{os: strings.TrimSpace(name)})
	}
	minutes := func(s string) time.Duration {
		n, err := strconv.Atoi(s)
		if err != nil {
			t.Fatalf("%s: timeout-minutes %q: %v", where, s, err)
		}
		return time.Duration(n) * time.Minute
	}
	if m := regexp.MustCompile(`(?m)^    timeout-minutes: (\d+)\s*$`).FindStringSubmatch(job); m != nil {
		for i := range legs {
			legs[i].minutes = minutes(m[1])
		}
		return legs
	}
	m := regexp.MustCompile(`(?m)^    timeout-minutes: \$\{\{ matrix\.os == '([a-z0-9.-]+)' && (\d+) \|\| (\d+) \}\}\s*$`).FindStringSubmatch(job)
	if m == nil {
		t.Fatalf("%s: timeout-minutes is neither a number nor "+
			"`${{ matrix.os == '<os>' && <n> || <m> }}`", where)
	}
	named := false
	for i := range legs {
		if legs[i].os == m[1] {
			legs[i].minutes, named = minutes(m[2]), true
		} else {
			legs[i].minutes = minutes(m[3])
		}
	}
	if !named {
		t.Fatalf("%s: timeout-minutes names %q, which is not a leg of the matrix", where, m[1])
	}
	return legs
}

// rulesetMirror is the tree's record of the live branch ruleset on main.
const rulesetMirror = ".abcd/work/rulesets/main-protection.json"

// mergeQueueResponseTimeout reads the merge_queue rule's
// check_response_timeout_minutes from the ruleset mirror, failing the test
// when the mirror declares no merge queue or no positive cap.
func mergeQueueResponseTimeout(t *testing.T, root string) time.Duration {
	t.Helper()
	var ruleset struct {
		Rules []struct {
			Type       string `json:"type"`
			Parameters struct {
				CheckResponseTimeoutMinutes int `json:"check_response_timeout_minutes"`
			} `json:"parameters"`
		} `json:"rules"`
	}
	if err := json.Unmarshal([]byte(readRepoFile(t, root, rulesetMirror)), &ruleset); err != nil {
		t.Fatalf("decoding %s: %v", rulesetMirror, err)
	}
	for _, r := range ruleset.Rules {
		if r.Type != "merge_queue" {
			continue
		}
		if r.Parameters.CheckResponseTimeoutMinutes <= 0 {
			t.Fatalf("%s: the merge_queue rule declares no positive check_response_timeout_minutes", rulesetMirror)
		}
		return time.Duration(r.Parameters.CheckResponseTimeoutMinutes) * time.Minute
	}
	t.Fatalf("%s declares no merge_queue rule; the check job's ceiling is sized against it", rulesetMirror)
	return 0
}

// raceTimeout returns the -timeout the one `go test -race` command in text
// carries, failing the test when there is none.
func raceTimeout(t *testing.T, where, text string) time.Duration {
	t.Helper()
	var cmds []string
	for _, l := range strings.Split(text, "\n") {
		l = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(l), "run:"))
		if strings.HasPrefix(l, "go test -race") {
			cmds = append(cmds, l)
		}
	}
	if len(cmds) != 1 {
		t.Fatalf("%s: want one `go test -race` command, found %d", where, len(cmds))
	}
	m := regexp.MustCompile(`\s-timeout[ =](\S+)`).FindStringSubmatch(cmds[0])
	if m == nil {
		t.Fatalf("%s: `%s` sets no -timeout, so the lane inherits go test's 10m default "+
			"per package, which internal/surface/cli under -race has already crossed", where, cmds[0])
	}
	d, err := time.ParseDuration(m[1])
	if err != nil {
		t.Fatalf("%s: -timeout %q: %v", where, m[1], err)
	}
	return d
}

// jobTimeoutMinutes reads the job-level `timeout-minutes:` (four-space indent,
// directly under the job key), failing the test when the job declares none.
func jobTimeoutMinutes(t *testing.T, where, job string) time.Duration {
	t.Helper()
	m := regexp.MustCompile(`(?m)^    timeout-minutes: (\d+)\s*$`).FindStringSubmatch(job)
	if m == nil {
		t.Fatalf("%s declares no job-level timeout-minutes", where)
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		t.Fatalf("%s: timeout-minutes %q: %v", where, m[1], err)
	}
	return time.Duration(n) * time.Minute
}
