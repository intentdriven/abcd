package lint_test

import (
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
// The enclosing job's timeout-minutes is the other half. A package timeout that
// reaches past the job's own ceiling never fires: the runner cancels the job
// first, and a cancellation carries no goroutine dump, so a real hang would read
// as a slow runner. Each job therefore holds the package timeout plus the time
// its slowest package waits before it starts: the steps ahead of the race step,
// and the packages `go test` runs ahead of it inside the step.
func TestRaceLaneBudgetIsDeclaredAndFitsItsJob(t *testing.T) {
	root := filepath.Join("..", "..", "..")

	recipe, ok := makeRecipe(readRepoFile(t, root, "Makefile"), "preflight")
	if !ok {
		t.Fatal("Makefile declares no `preflight:` recipe")
	}
	local := raceTimeout(t, "Makefile preflight", recipe)

	for _, c := range []struct {
		file, job string
		// headroom is what the job spends outside the slowest package's own
		// run, measured on its slowest leg and rounded up.
		headroom time.Duration
	}{
		// macOS leg of run 36212607621: 8.3 minutes before the race step began,
		// then 9.8 minutes inside it before internal/surface/cli started.
		{".github/workflows/ci.yml", "check", 20 * time.Minute},
		// The last release (run 35963282477, ubuntu, uncached toolchain): 2.8
		// minutes before the race step, 4.8 inside it before
		// internal/surface/cli started, 0.5 after it; the ubuntu race step
		// has grown by about two minutes since.
		{".github/workflows/release.yml", "verify", 10 * time.Minute},
	} {
		where := c.file + " job " + c.job
		job, ok := workflowJobBlock(readRepoFile(t, root, c.file), c.job)
		if !ok {
			t.Errorf("%s: no such job; the parser or the workflow changed shape", where)
			continue
		}
		step, ok := workflowStepBlock(job, "Test (race, internal)")
		if !ok {
			t.Errorf("%s: no `Test (race, internal)` step", where)
			continue
		}
		pkg := raceTimeout(t, where, step)
		if pkg != local {
			t.Errorf("%s runs the race lane under -timeout %s but make preflight uses %s; "+
				"local and CI must judge the lane against one budget", where, pkg, local)
		}
		capMin := jobTimeoutMinutes(t, where, job)
		if need := pkg + c.headroom; capMin < need {
			t.Errorf("%s: timeout-minutes is %s, below the package timeout %s plus %s of headroom (%s); "+
				"the runner would cancel the job before go test could report a hang",
				where, capMin, pkg, c.headroom, need)
		}
	}
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
