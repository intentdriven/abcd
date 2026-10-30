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
// not repeat, so the race steps' go test may carry -race and -timeout alone:
// another flag would run each package as one process with the count still set.
func TestCIRacePassesShardLikePreflight(t *testing.T) {
	root := filepath.Join("..", "..", "..")

	m := regexp.MustCompile(`(?m)^preflight: export ABCD_TEST_SHARDS := (\d+)$`).
		FindStringSubmatch(readRepoFile(t, root, "Makefile"))
	if m == nil {
		t.Fatal("the Makefile's preflight recipe exports no ABCD_TEST_SHARDS count")
	}
	want := `ABCD_TEST_SHARDS: "` + m[1] + `"`

	ci := readRepoFile(t, root, filepath.Join(".github", "workflows", "ci.yml"))
	for _, name := range []string{"Test (race, internal)", "Test (race, whole module)"} {
		step, ok := workflowStepBlock(ci, name)
		if !ok {
			t.Errorf("ci.yml has no `%s` step", name)
			continue
		}
		if !strings.Contains(step, "\n          "+want+"\n") {
			t.Errorf("ci.yml's `%s` step does not set %s in its env, the count preflight runs with:\n%s", name, want, step)
		}
		run := regexp.MustCompile(`(?m)^\s+run: go test (.*)$`).FindStringSubmatch(step)
		if run == nil {
			t.Errorf("ci.yml's `%s` step runs no `go test` line", name)
			continue
		}
		for _, f := range strings.Fields(run[1]) {
			if strings.HasPrefix(f, "-") && f != "-race" && f != "-timeout" {
				t.Errorf("ci.yml's `%s` step passes %s, which runs each package as one process whatever the shard count", name, f)
			}
		}
	}
}
