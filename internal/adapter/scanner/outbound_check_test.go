package scanner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// CheckOutbound is the check-direction twin of ScrubOutbound, and the tests
// below pin the three properties that make it usable as a GATE rather than as a
// sanitiser: it refuses, it reports what it refused, and it judges exactly the
// outbound policy's own class.

// The session-URL half is the half no deterministic gate covered: the shell
// attribution gate catches the footer and nothing catches the link. A commit
// message carrying one must be refused, and the refusal must arrive as an error
// so a caller that ignores the findings slice still fails closed.
func TestCheckOutboundRefusesASessionURL(t *testing.T) {
	id := synthSessionID(t, 62)
	msg := "fix: the walk skips a record family\n\nSee https://agent-host.dev/code/session_" + id + "\n"

	findings, err := CheckOutbound(t.TempDir(), msg, "commit-message")
	if err == nil {
		t.Fatalf("expected a refusal for a commit message carrying a session URL; got none")
	}
	if !hasKind(findings, kindHarnessSessionURL) {
		t.Errorf("refusal did not report the session URL; findings: %+v", findings)
	}
	if !strings.Contains(err.Error(), "commit-message") {
		t.Errorf("the refusal must name the artefact it judged; got %q", err)
	}
}

// The footer half too. The shell gate already catches this shape on its own
// anchor, and that is fine: one definition of the class, two gates reading it,
// is the position AGENTS.md states. What is not fine is a Go check that knows
// only half the policy it claims to enforce.
func TestCheckOutboundRefusesAnAttributionFooter(t *testing.T) {
	body := "Closes the gate.\n\n🤖 Generated with [Some Tool](https://sometool.dev)\n"

	findings, err := CheckOutbound(t.TempDir(), body, "pr-body")
	if err == nil {
		t.Fatalf("expected a refusal for a body carrying a tool attribution footer; got none")
	}
	if !hasKind(findings, kindHarnessFooter) {
		t.Errorf("refusal did not report the footer; findings: %+v", findings)
	}
}

// Clean text passes with nothing to say. The gate runs on every commit of every
// pull request, so a false red here is the failure that gets the gate disabled.
func TestCheckOutboundPassesCleanText(t *testing.T) {
	msg := "fix: the regime operator-surface walk skips the readings record family\n\n" +
		"Assisted-by: Claude:claude-opus-5\n"

	findings, err := CheckOutbound(t.TempDir(), msg, "commit-message")
	if err != nil {
		t.Fatalf("clean text must pass; got %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("clean text must report nothing; got %+v", findings)
	}
}

// The asymmetry with ScrubOutbound, pinned so it cannot be "fixed" into
// symmetry by someone reading the two side by side.
//
// ScrubOutbound masks EVERYTHING the scanner finds, because masking more than
// the policy names is never wrong — the artefact still reads, and the extra
// mask costs nothing. A CHECK that refuses more than the policy names is a
// different thing entirely: it turns a required pull-request check red on an
// ordinary commit message that happens to quote a private address, which is not
// the outbound policy's business at all. Committed text is judged for that
// class by `abcd lint`'s privacy rule and the record/docs lint; this gate judges
// the two shapes a harness stamps onto public text.
func TestCheckOutboundJudgesOnlyTheOutboundPolicyClass(t *testing.T) {
	msg := "fix: point the collector at the lab box\n\nThe host is 192.168.1.14 now.\n"

	// Guard the fixture: if this stopped being a finding at all the test would
	// pass for the wrong reason.
	sc, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(sc.ScanText(msg, "fixture")) == 0 {
		t.Fatalf("fixture no longer trips any detector, so it cannot prove the narrowing")
	}

	findings, err := CheckOutbound(t.TempDir(), msg, "commit-message")
	if err != nil {
		t.Fatalf("a non-policy finding must not refuse an outbound artefact; got %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("a non-policy finding must not be reported by the outbound gate; got %+v", findings)
	}
}

// The degraded-config refusal every write-time redactor in this repository
// makes. New() hands back a usable scanner on every degradation path, so an
// unreadable .abcd/config/pii.json silently drops the repo's OWN detectors and
// leaves the built-in set. Reporting "clean" from a weakened set is the one
// answer a gate must never give.
func TestCheckOutboundRefusesADegradedScanner(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".abcd", "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".abcd", "config", "pii.json"),
		[]byte("{ not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := CheckOutbound(root, "Ordinary text.\n", "pr-body"); err == nil {
		t.Fatal("expected a refusal on a degraded config")
	}
}
