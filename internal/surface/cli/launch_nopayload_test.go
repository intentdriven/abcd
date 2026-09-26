package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLaunchDryRunInANonPayloadRepoNamesTheReleasePath is iss-2608270559313719:
// a repository with no launch payload is told which release path it does have,
// not handed a missing-file error.
func TestLaunchDryRunInANonPayloadRepoNamesTheReleasePath(t *testing.T) {
	r := shipFixture(t)
	out, err := shipIn(t, r, "launch", "--dry-run")
	if code := exitCodeOf(err); code != 1 {
		t.Fatalf("exit = %d, want 1\n%s", code, out)
	}
	for _, want := range []string{"declares no launch payload", "launch scaffold", "CHANGELOG", "auto-release"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not name %q:\n%v", want, err)
		}
	}
	if _, statErr := os.Stat(filepath.Join(r.Root(), ".abcd", ".work.local")); statErr == nil {
		t.Error("a repository with no launch payload must not have a report written into it")
	}
}
