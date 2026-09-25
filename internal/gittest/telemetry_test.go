package gittest

import (
	"os/exec"
	"strings"
	"testing"
)

// A test that runs the go command under the HOME Env hands it must not start
// Go's telemetry: with the default mode the go command spawns a detached child
// that keeps writing counters under the temp HOME after the test ends, and the
// test's TempDir cleanup then fails with "directory not empty" (a CI flake on
// both platforms in the commit-msg hook tests).
func TestEnvTurnsGoTelemetryOff(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go is not on PATH")
	}
	cmd := exec.Command("go", "env", "GOTELEMETRY")
	cmd.Env = Env(t)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go env GOTELEMETRY: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != "off" {
		t.Fatalf("go telemetry mode under the test HOME is %q, want off", got)
	}
}
