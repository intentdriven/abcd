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
// there: the queue gives up first, with no job failure to read. So the check
// job's timeout-minutes may not exceed the cap the ruleset mirror records. The
// step's -timeout still earns its place under that cap: it lifts the ten-minute
// per-package default, so a slow package fails on the job's clock rather than
// at ten minutes. Raising the queue's cap is an admin act on the live ruleset,
// left to the technical facilitator; this test reads the mirror, so the day the
// cap is raised the check job may follow it.
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
		capMin := jobTimeoutMinutes(t, where, check)
		queue := mergeQueueResponseTimeout(t, root)
		if capMin > queue {
			t.Errorf("%s: timeout-minutes is %s, above the merge queue's %s check response "+
				"timeout in %s; on a merge-group run the queue fails the group first, so the "+
				"job ceiling is unreachable there and any budget sized against it is false. "+
				"Raise the live ruleset's cap (an admin act) before the job's",
				where, capMin, queue, rulesetMirror)
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
