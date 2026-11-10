package cli

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/runner"
)

// stageEnvPrefix is the prefix every variable a drive's stage sets shares
// (internal/core/runner/profile.go).
const stageEnvPrefix = "ABCD_STAGE_"

// stageEnvChildEnv marks the re-executed test binary as the child that reports
// whether a stage's variables reached it.
const stageEnvChildEnv = "ABCD_TEST_STAGE_ENV_CHILD"

// clearStageEnv unsets every variable a drive's stage sets, so a test run from
// a stage's shell (a drive's own lane running this package) behaves as it does
// anywhere else: with them kept, the hook and `rules --prompt` read the rules
// from the stage's base commit and recall no notes or pages, so a test of
// either passes or fails on the stage rather than on the code. A test that
// wants a stage sets the variables with t.Setenv. Matching the prefix rather
// than a list keeps a variable added later out of reach too.
func clearStageEnv() {
	for _, kv := range os.Environ() {
		name, _, _ := strings.Cut(kv, "=")
		if strings.HasPrefix(name, stageEnvPrefix) {
			os.Unsetenv(name)
		}
	}
}

// TestNoTestInheritsADrivesStage: the package's tests run as if no stage were
// set, however the test binary was started. The child is this test binary
// re-executed with every stage variable set; it reports each one that reached
// a test, and whether the stage still reads as active.
func TestNoTestInheritsADrivesStage(t *testing.T) {
	names := []string{
		runner.StageWritesEnv, runner.StageHomeEnv, runner.StageProtectEnv,
		runner.StageDeclaredEnv, runner.StageBaseEnv, runner.StageHarnessEnv,
	}
	if os.Getenv(stageEnvChildEnv) != "" {
		var kept []string
		for _, name := range names {
			if _, ok := os.LookupEnv(name); ok {
				kept = append(kept, name)
			}
		}
		if stageActive() {
			kept = append(kept, "stageActive")
		}
		if len(kept) > 0 {
			t.Fatalf("a stage reached the tests: %s", strings.Join(kept, ", "))
		}
		return
	}
	for _, name := range names {
		if !strings.HasPrefix(name, stageEnvPrefix) {
			t.Fatalf("stage variable %s lacks the %s prefix the test set-up clears", name, stageEnvPrefix)
		}
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestNoTestInheritsADrivesStage$", "-test.count=1")
	cmd.Env = append(os.Environ(), stageEnvChildEnv+"=1")
	for _, name := range names {
		cmd.Env = append(cmd.Env, name+"=/stage")
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("the child run with a stage's variables set failed (%v):\n%s", err, out)
	}
}
