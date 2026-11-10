package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/implement/loop"
)

// TestADriveWhoseReviewersCannotLaunchIsRefusedAtItsStart
// (iss-2611080057001367): a review set with a reviewer model no route on this
// machine can launch is refused before the drive takes its checkout or opens a
// lane, naming the model, the role and the configuration to set, rather than
// waiting at review for ever.
func TestADriveWhoseReviewersCannotLaunchIsRefusedAtItsStart(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	w.machineConfig(t, `{"oracle":{"denylist":["anthropic/*"]},"runner":{"claude":{}}}`)
	text := refusedDrive(t, "drain", "--drive", "--yes")
	for _, want := range []string{loop.BundledClasses[loop.ClassMiddle], loop.RoleRuthless, "oracle.api.anthropic", "nothing started"} {
		if !strings.Contains(text, want) {
			t.Errorf("the refusal %q does not name %q", text, want)
		}
	}
	if strings.Contains(text, w.home) || strings.Contains(text, w.repo) {
		t.Errorf("the refusal names an absolute path: %s", text)
	}
	noDrainBegun(t, w.repo)
	if _, err := os.Stat(filepath.Join(w.repo, filepath.FromSlash(driveLockRel))); err == nil {
		t.Error("a refused drive took the checkout's lock")
	}
	if w.terminalCalls != 0 {
		t.Fatal("nothing runs")
	}
}

// TestASubscriptionOnlyDriveIsNotRefusedForItsReviewers: the bundled review set
// on a machine that enables the claude harness and configures no provider
// passes the start's check and reaches the loop.
func TestASubscriptionOnlyDriveIsNotRefusedForItsReviewers(t *testing.T) {
	w := newDriveWorld(t)
	w.passWalls(t)
	var out, errb strings.Builder
	if code := Run([]string{"drain", "--drive", "--yes"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d:\n%s%s", code, out.String(), errb.String())
	}
	if w.terminalCalls != 1 {
		t.Fatalf("the loop ran %d times, want once", w.terminalCalls)
	}
}
