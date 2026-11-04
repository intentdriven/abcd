package scaffold

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

var (
	verifyJobLimit = regexp.MustCompile(`(?m)^\s*timeout-minutes: (\d+)\s*$`)
	testStepLimit  = regexp.MustCompile(`go test[^\n]*-timeout (\S+)`)
)

// A hang must fail on go test's own limit, which prints a goroutine dump, not on
// the job's, which cancels with none. The eval and race lane budget tests read
// abcd's own release.yml, so a scaffolded repository's render is held here. Go's
// default limit counts when a step sets none.
func TestEveryRenderedTestStepLimitFitsUnderTheVerifyJob(t *testing.T) {
	const goDefault = 10 * time.Minute
	for name, subs := range map[string]Substitutions{
		"abcd": AbcdSubstitutions(), "bare": BareSubstitutions("main"), "gate": GateSubstitutions("main", ""),
	} {
		rendered, err := Render(subs)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		verify := jobSection(t, string(rendered.ReleaseYML), "verify")
		jm := verifyJobLimit.FindStringSubmatch(verify)
		if jm == nil {
			t.Fatalf("%s: the verify job declares no timeout-minutes", name)
		}
		mins, _ := strconv.Atoi(jm[1])
		job := time.Duration(mins) * time.Minute
		steps := 0
		for _, line := range strings.Split(verify, "\n") {
			if !strings.Contains(line, "run: go test") {
				continue
			}
			steps++
			limit := goDefault
			if m := testStepLimit.FindStringSubmatch(line); m != nil {
				if limit, err = time.ParseDuration(m[1]); err != nil {
					t.Errorf("%s: %q: %v", name, line, err)
					continue
				}
			}
			if limit >= job {
				t.Errorf("%s: %q has a %s limit, not under the job's %s; a hang would be cancelled with no goroutine dump", name, strings.TrimSpace(line), limit, job)
			}
		}
		if steps == 0 {
			t.Errorf("%s: the verify job runs no go test; the render or this test changed shape", name)
		}
	}
}
