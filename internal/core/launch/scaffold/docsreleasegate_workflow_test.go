package scaffold

import (
	"strings"
	"testing"
)

// TestReleaseVerifyArmsTheDocsCitationGate holds the release path to the
// stricter docs lint (iss-2609091801085579). `abcd lint docs --release-gate`
// promotes a citation past its staleness threshold from a warning to a
// blocker, and nothing ran it: the release workflow invoked the bare lint, so
// an overdue citation reached a release with a warning nobody had to answer
// while the flag read as a release control. verify is the path every release
// takes, so the flag is armed there — and only there: commits are never
// calendar-blocked (spc-17), so ci.yml keeps the bare lint.
func TestReleaseVerifyArmsTheDocsCitationGate(t *testing.T) {
	rendered, err := Render(AbcdSubstitutions())
	if err != nil {
		t.Fatal(err)
	}
	verify := jobSection(t, string(rendered.ReleaseYML), "verify")
	if !strings.Contains(verify, "run: go run ./cmd/abcd lint docs --release-gate\n") {
		t.Error("release.yml's verify job must run the docs lint in release-gate mode")
	}
	if strings.Contains(verify, "run: go run ./cmd/abcd lint docs\n") {
		t.Error("release.yml's verify job still runs the bare docs lint, where an overdue citation only warns")
	}
}
