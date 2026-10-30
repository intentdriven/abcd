package lint_test

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
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
// rather than a budget anyone had sized, so it is written down here, in four
// places that must agree: the merge gate (ci.yml's race lane on ubuntu and its
// one race-enabled pass on macOS), the release gate (release.yml's verify job)
// and the local pre-push gate (make preflight).
//
// The enclosing ceilings are the other half, and they differ by job.
//
// ci.yml's check job is a required context, and on a merge-group run the merge
// queue fails the group when a required check has not concluded within the
// ruleset's check_response_timeout_minutes. That cap is the outer ceiling on
// the event the lane was ejected on, and a job ceiling above it is unreachable
// there: the queue gives up first, with no job failure to read. So no leg of
// the check job may carry a timeout-minutes above the cap the ruleset mirror
// records. Ruling Z (2026-09-28, the technical facilitator) raised the macOS
// leg, and every other leg carries 45 since 2026-10-28 (dcn-2610280635092222),
// after a first spell at 45 from 2026-10-17 (dcn-2610171803056931) and 30
// between (iss-2610171803053186); the job declares that as one matrix
// expression, which checkLegTimeouts reads leg by leg. The queue's cap is 90
// minutes since 2026-10-11 and the macOS leg's is 85 since 2026-10-28, below
// it, so on that leg the job's own ceiling is the binding one. The step's -timeout still earns its place under that cap: it lifts the
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
//
// The commands also name one package list, in one order. go test starts
// packages in the order its arguments list them, a few at a time on a runner
// with few cores, so the slowest packages are named ahead of the pattern and
// start first instead of last (iss-2609281715083637). The order may change; the
// set may not: go list must resolve the command's packages to exactly the ones
// ./internal/... matches. The macOS pass is the one exception, by design
// (iss-2610110938293273): it is that leg's only pass over the module, so the same
// slowest-first list ends in ./... and resolves to the whole module.
func TestRaceLaneBudgetIsDeclaredAndFitsItsJob(t *testing.T) {
	root := filepath.Join("..", "..", "..")

	recipe, ok := makeRecipe(readRepoFile(t, root, "Makefile"), preflightStages)
	if !ok {
		t.Fatalf("Makefile declares no `%s:` recipe", preflightStages)
	}
	local := raceTimeout(t, "Makefile preflight", recipe)
	localPkgs := racePackages(t, "Makefile preflight", recipe)
	sameRacePackageSet(t, root, "Makefile preflight", localPkgs, raceLanePattern)

	// raceStep finds a job's race step by name and holds its -timeout to the
	// local gate's and its package list to want, in order.
	raceStep := func(where, block, name string, want []string) (time.Duration, bool) {
		step, ok := workflowStepBlock(block, name)
		if !ok {
			t.Errorf("%s: no `%s` step", where, name)
			return 0, false
		}
		pkg := raceTimeout(t, where, step)
		if pkg != local {
			t.Errorf("%s step %q runs under -timeout %s but make preflight uses %s; "+
				"local and CI must judge the lane against one budget", where, name, pkg, local)
		}
		if pkgs := racePackages(t, where, step); !slices.Equal(pkgs, want) {
			t.Errorf("%s step %q runs over %q; want %q: local and CI start the same "+
				"packages in the same order", where, name, pkgs, want)
		}
		return pkg, true
	}
	// raceJob returns a workflow job's block and the -timeout its race step
	// carries, holding that timeout to the local gate's.
	raceJob := func(file, job string) (string, time.Duration, bool) {
		where := file + " job " + job
		block, ok := workflowJobBlock(readRepoFile(t, root, file), job)
		if !ok {
			t.Errorf("%s: no such job; the parser or the workflow changed shape", where)
			return "", 0, false
		}
		pkg, ok := raceStep(where, block, "Test (race, internal)", localPkgs)
		return block, pkg, ok
	}

	if check, _, ok := raceJob(".github/workflows/ci.yml", "check"); ok {
		const where = ".github/workflows/ci.yml job check"
		// The macOS leg's one pass: the same slowest-first list, ending in the
		// whole module rather than ./internal/....
		whole := append(slices.Clone(localPkgs[:len(localPkgs)-1]), modulePattern)
		if _, ok := raceStep(where, check, macOSRaceStep, whole); ok {
			step, _ := workflowStepBlock(check, macOSRaceStep)
			sameRacePackageSet(t, root, where+" step "+macOSRaceStep, racePackages(t, where, step), modulePattern)
			if !regexp.MustCompile(`(?m)^\s+ABCD_TEST_SKIP_FLUSH: "1"\s*$`).MatchString(step) {
				t.Errorf("%s step %q does not set ABCD_TEST_SKIP_FLUSH; every synced write on the "+
					"macOS runner is then an F_FULLFSYNC no test asserts anything about", where, macOSRaceStep)
			}
		}
		queue := mergeQueueResponseTimeout(t, root)
		for _, leg := range checkLegTimeouts(t, where, check) {
			if leg.minutes > queue {
				t.Errorf("%s: the %s leg's timeout-minutes is %s, above the merge queue's %s check "+
					"response timeout in %s; on a merge-group run the queue fails the group first, "+
					"so the job ceiling is unreachable there and any budget sized against it is "+
					"false. Raise the live ruleset's cap (an admin act) before the job's",
					where, leg.os, leg.minutes, queue, rulesetMirror)
			}
			if leg.os == "macos-latest" && leg.minutes != checkMacOSCeilingMinutes*time.Minute {
				t.Errorf("%s: the macos-latest leg's timeout-minutes is %s; want %dm, the ceiling "+
					"set on 2026-10-28 (dcn-2610282036219503)", where, leg.minutes, checkMacOSCeilingMinutes)
			}
			// Ruling Z raised the macOS leg; every other leg carries 45 since
			// 2026-10-28 (dcn-2610280635092222).
			if leg.os != "macos-latest" && leg.minutes != checkUbuntuCeilingMinutes*time.Minute {
				t.Errorf("%s: the %s leg's timeout-minutes is %s; want %dm, the ceiling "+
					"of every leg but macOS", where, leg.os, leg.minutes, checkUbuntuCeilingMinutes)
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

// checkMacOSCeilingMinutes is the macOS leg's timeout-minutes in ci.yml's check
// job: 45 under ruling Z (2026-09-28); 60 from 2026-10-15, when merge-queue
// macOS runs had grown from 31-35 to 34-40 minutes; 70 from 2026-10-18, when
// ten merge-group runs took 35.1 to 49.3 minutes on a leg whose race pass keeps
// its three test slots full (dcn-2610181756328249); 85 since 2026-10-28, when
// passing merge-group runs took 46 to 68 minutes and run 40752008734 was
// cancelled at the cap after 71 (dcn-2610282036219503). It stays below the merge
// queue's check response timeout, which the budget test reads from the mirror.
const checkMacOSCeilingMinutes = 85

// checkUbuntuCeilingMinutes is the check job's ceiling on every leg but
// macOS: 30 under ruling Z; 45 from 2026-10-17, when the drain bundle
// (spc-2610161044461448) grew the loop package's race run past 30
// (dcn-2610171803056931); 30 when the plain pass stopped rerunning the race
// lane's tests (iss-2610171803053186); 45 again since 2026-10-28, when
// merge-group runs took 29 minutes and one was cancelled at 30 a second after
// its race tests passed (dcn-2610280635092222).
const checkUbuntuCeilingMinutes = 45

// checkWarnLeadMinutes is how far ahead of each leg's ceiling the warning step
// fires: ruling Z's cancel policy (2026-09-28) set the line ten minutes under
// the cap, so a speed lane opens before anything is cancelled.
const checkWarnLeadMinutes = 10

// checkWarnStep is the name of that warning step in the check job.
const checkWarnStep = "Warn when the check nears the queue's cap"

// checkClockStep is the step that records the job's start for the warning.
const checkClockStep = "Record the job's start"

// Each leg warns before its cap cancels it (ruling Z's cancel policy,
// 2026-09-28). A cancellation at the cap carries nothing but the
// cancellation, and the rule for it is rerun once, then stop that pull request
// and open a speed lane; the warning moves that signal ahead of the first
// cancellation. It is a warning and nothing else: it may not fail the job, it
// runs however the steps before it ended, and each leg's threshold sits below
// that leg's ceiling, or it could never fire before the cancellation it
// announces.
func TestCheckJobWarnsBeforeTheQueueCap(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	const where = ".github/workflows/ci.yml job check"
	job, ok := workflowJobBlock(readRepoFile(t, root, ".github/workflows/ci.yml"), "check")
	if !ok {
		t.Fatalf("%s: no such job; the parser or the workflow changed shape", where)
	}
	step, ok := workflowStepBlock(job, checkWarnStep)
	if !ok {
		t.Fatalf("%s: no %q step; a leg would reach its cap with no warning ahead of the "+
			"cancellation", where, checkWarnStep)
	}

	legs := checkLegTimeouts(t, where, job)
	var oses []string
	for _, leg := range legs {
		oses = append(oses, leg.os)
	}
	// stepEnv reads one of the step's env values per leg: a number every leg
	// shares, or the matrix expression the job's timeout-minutes uses.
	stepEnv := func(key string) (map[string]int, bool) {
		m := regexp.MustCompile(`(?m)^\s+` + key + `: (.+?)\s*$`).FindStringSubmatch(step)
		if m == nil {
			return nil, false
		}
		return legValues(t, where+" step "+checkWarnStep, key, m[1], oses), true
	}
	warns, ok := stepEnv("WARN_AFTER_MINUTES")
	if !ok {
		t.Fatalf("%s step %q: no WARN_AFTER_MINUTES in its env", where, checkWarnStep)
	}
	// The cap the message names comes from the step's own env, held to each
	// leg's ceiling here, so the message cannot name a cap the job no longer
	// has (dcn-2610181756328249).
	ceilings, ok := stepEnv("CEILING_MINUTES")
	if !ok {
		t.Errorf("%s step %q: no CEILING_MINUTES in its env; the message would name the cap from a "+
			"number no test holds to the leg's ceiling", where, checkWarnStep)
	}
	queue := mergeQueueResponseTimeout(t, root)
	lead := time.Duration(checkWarnLeadMinutes) * time.Minute
	for _, leg := range legs {
		warnAfter := time.Duration(warns[leg.os]) * time.Minute
		if leg.minutes-warnAfter != lead {
			t.Errorf("%s step %q warns the %s leg after %s, %s before its %s ceiling; "+
				"the cancel policy of ruling Z puts the line %s before it", where, checkWarnStep,
				leg.os, warnAfter, leg.minutes-warnAfter, leg.minutes, lead)
		}
		if warnAfter >= queue {
			t.Errorf("%s step %q warns the %s leg after %s, not below the merge queue's %s cap in %s",
				where, checkWarnStep, leg.os, warnAfter, queue, rulesetMirror)
		}
		if ceilings != nil && time.Duration(ceilings[leg.os])*time.Minute != leg.minutes {
			t.Errorf("%s step %q: CEILING_MINUTES is %dm on the %s leg; want %s, that leg's ceiling",
				where, checkWarnStep, ceilings[leg.os], leg.os, leg.minutes)
		}
	}

	// Every leg warns; the macOS leg only off a pull request, where it runs
	// nothing but its stand-in.
	const wantIf = "always() && (matrix.os != 'macos-latest' || github.event_name != 'pull_request')"
	ifLine := regexp.MustCompile(`(?m)^\s+if: (.*)$`).FindStringSubmatch(step)
	if ifLine == nil || strings.TrimSpace(ifLine[1]) != wantIf {
		t.Errorf("%s step %q: want `if: %s`, so it runs on every leg however the steps before "+
			"it ended", where, checkWarnStep, wantIf)
	}
	// The warning reads its clock from the step that records the job's start,
	// so that step runs on every leg the warning does; on a leg it skips, the
	// warning has no start time and stays silent.
	const wantClockIf = "matrix.os != 'macos-latest' || github.event_name != 'pull_request'"
	if clock, ok := workflowStepBlock(job, checkClockStep); !ok {
		t.Errorf("%s: no %q step; the warning has no clock", where, checkClockStep)
	} else if m := regexp.MustCompile(`(?m)^\s+if: (.*)$`).FindStringSubmatch(clock); m == nil || strings.TrimSpace(m[1]) != wantClockIf {
		t.Errorf("%s step %q: want `if: %s`, the legs the warning runs on; on any other leg the "+
			"warning finds no start time and never fires", where, checkClockStep, wantClockIf)
	}
	if !regexp.MustCompile(`(?m)^\s+LEG: \$\{\{ matrix\.os \}\}\s*$`).MatchString(step) {
		t.Errorf("%s step %q: want `LEG: ${{ matrix.os }}` in its env, so the warning names the leg "+
			"it ran on", where, checkWarnStep)
	}
	if !regexp.MustCompile(`(?m)^\s+continue-on-error: true\s*$`).MatchString(step) {
		t.Errorf("%s step %q: want `continue-on-error: true`; a warning may never fail the job", where, checkWarnStep)
	}
	at := strings.Index(step, "run:")
	if at < 0 {
		t.Fatalf("%s step %q has no run: script", where, checkWarnStep)
	}
	run := step[at:]
	if !strings.Contains(run, "${CEILING_MINUTES}-minute cap") {
		t.Errorf("%s step %q: its script does not name the cap as ${CEILING_MINUTES}-minute cap; a "+
			"cap written any other way can drift from the ceiling", where, checkWarnStep)
	}
	if strings.Contains(run, "macos-latest") || strings.Contains(run, "ubuntu-latest") {
		t.Errorf("%s step %q: its script names a leg; it names ${LEG}, the leg it ran on", where, checkWarnStep)
	}
	for _, want := range []string{"::warning", "GITHUB_STEP_SUMMARY", "rerun", "speed lane", "${LEG}"} {
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
	var oses []string
	for _, name := range strings.Split(osLine[1], ",") {
		oses = append(oses, strings.TrimSpace(name))
	}
	m := regexp.MustCompile(`(?m)^    timeout-minutes: (.+?)\s*$`).FindStringSubmatch(job)
	if m == nil {
		t.Fatalf("%s: no job-level timeout-minutes", where)
	}
	vals := legValues(t, where, "timeout-minutes", m[1], oses)
	var legs []checkLeg
	for _, os := range oses {
		legs = append(legs, checkLeg{os: os, minutes: time.Duration(vals[os]) * time.Minute})
	}
	return legs
}

// legValues reads raw, a workflow value given per matrix leg, for each of
// oses: a plain number every leg shares, or the one expression
// `${{ matrix.os == '<os>' && <n> || <m> }}`, which gives <os> its own value
// and every other leg <m>.
func legValues(t *testing.T, where, what, raw string, oses []string) map[string]int {
	t.Helper()
	raw = strings.Trim(raw, `"`)
	vals := make(map[string]int, len(oses))
	if n, err := strconv.Atoi(raw); err == nil {
		for _, os := range oses {
			vals[os] = n
		}
		return vals
	}
	m := regexp.MustCompile(`^\$\{\{ matrix\.os == '([a-z0-9.-]+)' && (\d+) \|\| (\d+) \}\}$`).FindStringSubmatch(raw)
	if m == nil {
		t.Fatalf("%s: %s is %q, neither a number nor `${{ matrix.os == '<os>' && <n> || <m> }}`", where, what, raw)
	}
	if !slices.Contains(oses, m[1]) {
		t.Fatalf("%s: %s names %q, which is not a leg of the matrix", where, what, m[1])
	}
	named, _ := strconv.Atoi(m[2])
	other, _ := strconv.Atoi(m[3])
	for _, os := range oses {
		if os == m[1] {
			vals[os] = named
		} else {
			vals[os] = other
		}
	}
	return vals
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

// raceCommand returns the one `go test -race` command in text, failing the
// test when there is not exactly one.
func raceCommand(t *testing.T, where, text string) string {
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
	return cmds[0]
}

// racePackages returns the package arguments of the one `go test -race`
// command in text, in the order written: every word after `go test` that is
// neither a flag nor a flag's value. make preflight's -p is the derived
// parallelism (`-p "$$p"`, cmd/test-parallelism), whose value is a word too.
func racePackages(t *testing.T, where, text string) []string {
	t.Helper()
	var pkgs []string
	words := strings.Fields(raceCommand(t, where, text))[2:]
	for i := 0; i < len(words); i++ {
		switch w := words[i]; {
		case w == "-timeout" || w == "-p":
			i++ // its value is the next word
		case strings.HasPrefix(w, "-"):
		default:
			pkgs = append(pkgs, w)
		}
	}
	if len(pkgs) == 0 {
		t.Fatalf("%s: the `go test -race` command names no packages", where)
	}
	return pkgs
}

// raceLanePattern is the package set the race lane covers. The command may
// name packages ahead of it, so go test starts the slowest first on a runner
// with few cores, but it may neither add nor drop one.
const raceLanePattern = "./internal/..."

// modulePattern is the package set of the macOS leg's one race-enabled pass:
// the whole module, since it is that leg's only pass over the module.
const modulePattern = "./..."

// macOSRaceStep names that pass in ci.yml's check job.
const macOSRaceStep = "Test (race, whole module)"

// sameRacePackageSet holds a race pass's packages to exactly the ones pattern
// matches, as go list resolves both: the slowest-first order
// (iss-2609281715083637) names packages twice, once explicitly and once in the
// pattern, and go test runs each once, so the order is free while the set is
// not.
func sameRacePackageSet(t *testing.T, root, where string, pkgs []string, pattern string) {
	t.Helper()
	if pkgs[len(pkgs)-1] != pattern {
		t.Errorf("%s: the race lane's packages are %q; want %s last, so every package it matches runs",
			where, pkgs, pattern)
	}
	seen := map[string]bool{}
	for _, p := range pkgs {
		if seen[p] {
			t.Errorf("%s: the race lane names %s twice", where, p)
		}
		seen[p] = true
	}
	list := func(args ...string) []string {
		cmd := exec.Command("go", append([]string{"list"}, args...)...)
		cmd.Dir = root
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("%s: go list %s: %v", where, strings.Join(args, " "), err)
		}
		return strings.Fields(string(out))
	}
	got, want := list(pkgs...), list(pattern)
	if len(got) != len(want) {
		t.Errorf("%s: go list resolves the race lane to %d packages, %s to %d", where, len(got), pattern, len(want))
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("%s: the race lane's packages differ from %s's:\n  lane: %q\n  want: %q", where, pattern, got, want)
	}
}

// raceTimeout returns the -timeout the one `go test -race` command in text
// carries, failing the test when there is none.
func raceTimeout(t *testing.T, where, text string) time.Duration {
	t.Helper()
	cmd := raceCommand(t, where, text)
	m := regexp.MustCompile(`\s-timeout[ =](\S+)`).FindStringSubmatch(cmd)
	if m != nil && m[1] == `"$$t"` {
		// make preflight's stage scales its base with the load as it starts
		// (timeoutCall); the base is the budget CI's steps declare.
		m = nil
		before := text
		if i := strings.Index(text, cmd); i >= 0 {
			before = text[:i]
		}
		for _, l := range strings.Split(before, "\n") {
			if b := timeoutCall.FindStringSubmatch(strings.TrimSpace(l)); b != nil {
				m = b
			}
		}
	}
	if m == nil {
		t.Fatalf("%s: `%s` sets no -timeout, so the lane inherits go test's 10m default "+
			"per package, which internal/surface/cli under -race has already crossed", where, cmd)
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

// outsideInternalPattern is the plain pass the ubuntu leg runs beside its race
// lane: every package ./... matches that ./internal/... does not.
const outsideInternalPattern = "./cmd/..."

// TestTheOutsideInternalStepRunsWhatTheRaceLaneDoesNot: where the race lane
// runs, the ubuntu leg's plain pass covers only the packages it does not
// (iss-2610171803053186), and exactly those: go list resolves the step's
// pattern to the module's packages less the race lane's.
func TestTheOutsideInternalStepRunsWhatTheRaceLaneDoesNot(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	const where = ".github/workflows/ci.yml job check"
	check, ok := workflowJobBlock(readRepoFile(t, root, ".github/workflows/ci.yml"), "check")
	if !ok {
		t.Fatalf("%s: no such job; the parser or the workflow changed shape", where)
	}
	const name = "Test (outside internal)"
	step, ok := workflowStepBlock(check, name)
	if !ok {
		t.Fatalf("%s: no %q step", where, name)
	}
	if cmd, want := oneGoTest(t, where+" step "+name, step), "go test "+outsideInternalPattern; cmd != want {
		t.Fatalf("%s step %q runs `%s`; want `%s`", where, name, cmd, want)
	}
	list := func(pattern string) []string {
		cmd := exec.Command("go", "list", pattern)
		cmd.Dir = root
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("go list %s: %v", pattern, err)
		}
		return strings.Fields(string(out))
	}
	internal := list(raceLanePattern)
	var want []string
	for _, p := range list(modulePattern) {
		if !slices.Contains(internal, p) {
			want = append(want, p)
		}
	}
	if got := list(outsideInternalPattern); !slices.Equal(got, want) {
		t.Errorf("%s step %q: %s resolves to %q; the packages outside the race lane are %q",
			where, name, outsideInternalPattern, got, want)
	}
}

// TestPlainTestStepRunsTheWholeModule: the ubuntu leg's plain pass on an inert
// change, where the race lane stands down and it is the leg's only test pass,
// names its slowest packages ahead of ./..., so they start with the step
// instead of behind every faster package (a speed lane under the cancel policy
// of 2026-09-28), and it still runs exactly the module: ./... last, no package
// named twice, and the same set go list resolves ./... to. Its -timeout is
// make preflight's unit-stage base, declared rather than go test's ten-minute
// default: internal/core/implement/loop ran 466s to 593s here and crossed 600s
// on 2026-10-30 with no test hung (iss-2610302058222905).
func TestPlainTestStepRunsTheWholeModule(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	const where = ".github/workflows/ci.yml job check"
	check, ok := workflowJobBlock(readRepoFile(t, root, ".github/workflows/ci.yml"), "check")
	if !ok {
		t.Fatalf("%s: no such job; the parser or the workflow changed shape", where)
	}
	step, ok := workflowStepBlock(check, "Test")
	if !ok {
		t.Fatalf("%s: no `Test` step", where)
	}
	var cmds []string
	for _, l := range strings.Split(step, "\n") {
		l = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(l), "run:"))
		if strings.HasPrefix(l, "go test ") {
			cmds = append(cmds, l)
		}
	}
	if len(cmds) != 1 {
		t.Fatalf("%s step Test: want one `go test` command, found %d", where, len(cmds))
	}
	var pkgs []string
	timeout := ""
	words := strings.Fields(cmds[0])[2:]
	for i := 0; i < len(words); i++ {
		switch w := words[i]; {
		case w == "-timeout" && i+1 < len(words):
			i++
			timeout = words[i]
		case strings.HasPrefix(w, "-timeout="):
			timeout = strings.TrimPrefix(w, "-timeout=")
		case strings.HasPrefix(w, "-"):
		default:
			pkgs = append(pkgs, w)
		}
	}
	if len(pkgs) == 0 {
		t.Fatalf("%s step Test: `%s` names no packages", where, cmds[0])
	}
	sameRacePackageSet(t, root, where+" step Test", pkgs, modulePattern)

	recipe, ok := makeRecipe(readRepoFile(t, root, "Makefile"), preflightStages)
	if !ok {
		t.Fatalf("Makefile declares no `%s:` recipe", preflightStages)
	}
	local := unitStageTimeout(t, "Makefile preflight", recipe)
	if timeout == "" {
		t.Fatalf("%s step Test: `%s` sets no -timeout, so each package inherits go test's 10m "+
			"default, which internal/core/implement/loop has already crossed; make preflight's "+
			"unit stage uses %s", where, cmds[0], local)
	}
	d, err := time.ParseDuration(timeout)
	if err != nil {
		t.Fatalf("%s step Test: -timeout %q: %v", where, timeout, err)
	}
	if d != local {
		t.Errorf("%s step Test runs under -timeout %s but make preflight's unit stage uses %s; "+
			"local and CI must judge the module against one budget", where, d, local)
	}
}

// unitStageTimeout is the base -timeout make preflight's unit stage scales
// from: the `t=$$(go run ./cmd/test-parallelism -timeout <base>)` line last
// before the recipe's plain `go test` command, the one without -race.
func unitStageTimeout(t *testing.T, where, recipe string) time.Duration {
	t.Helper()
	base := ""
	for _, l := range strings.Split(recipe, "\n") {
		l = strings.TrimSpace(l)
		if m := timeoutCall.FindStringSubmatch(l); m != nil {
			base = m[1]
			continue
		}
		if strings.HasPrefix(l, "go test ") && !strings.Contains(l, "-race") {
			if base == "" {
				t.Fatalf("%s: `%s` takes no base -timeout from cmd/test-parallelism", where, l)
			}
			d, err := time.ParseDuration(base)
			if err != nil {
				t.Fatalf("%s: unit stage -timeout %q: %v", where, base, err)
			}
			return d
		}
	}
	t.Fatalf("%s: no plain `go test` command", where)
	return 0
}
