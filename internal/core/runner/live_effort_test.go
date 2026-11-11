//go:build live

package runner_test

// The live check of step 8 of spc-2611101423523725 (itd-2611101419006280): one
// short launch per bundled class through the claude runner at the bundled
// effort, each returning an answer and reporting the model that ran. It spends
// the person's key, so it sits behind the `live` build tag and nothing runs it
// by default:
//
//	GOFLAGS=-trimpath go test -tags live -count=1 -run TestEachBundledClassLaunchesAtItsEffort ./internal/core/runner
//
// The runner launches the harness bare, which bills only through a key held in
// the environment, never a login kept in the keychain. Where the harness is not
// on PATH or no such key is held, the check is skipped and says so on stderr
// as well as in the test's own output.

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/runner"
)

// skipLoudly skips the live check, naming why on stderr too, so a run that
// meant to spend and did not cannot read as a pass.
func skipLoudly(t *testing.T, why string) {
	t.Helper()
	msg := "SKIPPED the live model-set check: " + why
	fmt.Fprintln(os.Stderr, msg)
	t.Skip(msg)
}

func TestEachBundledClassLaunchesAtItsEffort(t *testing.T) {
	if _, err := exec.LookPath(runner.Claude); err != nil {
		skipLoudly(t, "no claude harness is on PATH")
	}
	if len(runner.HeldCredentials()) == 0 {
		skipLoudly(t, "no harness is logged in: the claude runner launches bare, which bills only through a key "+
			"held in the environment (ANTHROPIC_API_KEY or another the harness reads), and none is set")
	}
	for _, class := range loop.Classes {
		route := loop.BundledClasses[class]
		t.Run(class, func(t *testing.T) {
			dir := t.TempDir()
			lane := filepath.Join(dir, "lane")
			if err := os.Mkdir(lane, 0o700); err != nil {
				t.Fatal(err)
			}
			brief := filepath.Join(lane, "brief.md")
			if err := os.WriteFile(brief, []byte("# brief\n\nReply with the single word ok and do nothing else. "+
				"No receipt is wanted.\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			req := runner.Request{
				Role:      "live-check",
				Brief:     brief,
				Receipt:   filepath.Join(lane, "receipt.json"),
				Dir:       dir,
				Tools:     []string{"Read"},
				Model:     route,
				Class:     class,
				Effort:    loop.BundledEffort,
				SessionID: "live-check-" + class,
				Timeout:   runner.MinTimeout,
			}
			ans, _, err := runner.NewClaudeForLive("").Run(context.Background(), req)
			if err != nil {
				t.Fatalf("the %s class (%s, at effort %s) did not launch: %v", class, route, loop.BundledEffort, err)
			}
			if strings.TrimSpace(ans.Text) == "" {
				t.Errorf("the %s class (%s) returned no answer", class, route)
			}
			if ans.Model == "" {
				t.Errorf("the %s class (%s) reported no model", class, route)
			}
			if ans.EffortAsked != loop.BundledEffort {
				t.Errorf("the %s class's launch records effort %q asked, want %q", class, ans.EffortAsked, loop.BundledEffort)
			}
			t.Logf("%s: asked %s at effort %s, the harness reported %s", class, route, ans.EffortAsked, ans.Model)
		})
	}
}
