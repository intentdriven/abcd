package lint_test

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// CI's race passes run the longest packages as several test processes, through
// the same switch make preflight sets and with the same count
// (iss-2610290720087336). As one process internal/core/implement/loop took
// 2244s of the macOS pass's 40-minute -timeout on a passing run, and a queue
// entry timed out on it with no test hung, dropping the change from the merge
// queue. internal/testshard splits a package only when the variable asks and go
// test runs the whole package, so the -timeout stays the same budget, now per
// shard, and a step that names its tests with -run is left as one process: the
// plain-only guard steps carry no count, since it would do nothing there. The
// same holds for -json, -cover, -shuffle and every other flag testshard does
// not repeat, so the race steps' go test may carry -race, -timeout and the
// -count=1 every hosted test step passes, which testshard does repeat: another
// flag would run each package as one process with the count still set.
// A macOS shard's race pass carries the count like the rest: splitting the suite
// across runners does not make a package shorter on the runner it lands on. The
// check job's plain pass on an inert change, its only test pass there, shards the
// same way: as one process internal/core/implement/loop crossed go
// test's ten-minute default with no test hung (iss-2610302058222905).
func TestCIRacePassesShardLikePreflight(t *testing.T) {
	root := filepath.Join("..", "..", "..")

	m := regexp.MustCompile(`(?m)^preflight: export ABCD_TEST_SHARDS := (\d+)$`).
		FindStringSubmatch(readRepoFile(t, root, "Makefile"))
	if m == nil {
		t.Fatal("the Makefile's preflight recipe exports no ABCD_TEST_SHARDS count")
	}
	want := `ABCD_TEST_SHARDS: "` + m[1] + `"`

	const ci = ".github/workflows/ci.yml"
	for _, pass := range []struct{ file, name string }{
		{ci, "Test (race, internal)"},
		{macOSSuiteWorkflow, macOSRaceStep},
		{ci, "Test"},
	} {
		file, name := pass.file, pass.name
		step, ok := workflowStepBlock(readRepoFile(t, root, file), name)
		if !ok {
			t.Errorf("%s has no `%s` step", file, name)
			continue
		}
		if !strings.Contains(step, "\n          "+want+"\n") {
			t.Errorf("%s's `%s` step does not set %s in its env, the count preflight runs with:\n%s", file, name, want, step)
		}
		run := regexp.MustCompile(`(?m)^\s+(?:run: )?go test (.*)$`).FindStringSubmatch(step)
		if run == nil {
			t.Errorf("%s's `%s` step runs no `go test` line", file, name)
			continue
		}
		for _, f := range strings.Fields(run[1]) {
			if strings.HasPrefix(f, "-") && f != "-race" && f != "-timeout" && f != "-count=1" {
				t.Errorf("%s's `%s` step passes %s, which runs each package as one process whatever the shard count", file, name, f)
			}
		}
	}
}
