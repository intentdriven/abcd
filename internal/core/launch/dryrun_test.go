package launch

import (
	"errors"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
)

const fakeSecret = "ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789ab"

// TestDryRunSecretRefusesButExitsZero is brief AC 1: a planted secret inside an
// included file produces a hard-fail in the scan and in WouldRefuseOn, sets
// WouldPublish=false, writes no artefact, and the orchestrator returns nil error
// (exit-0 semantics — a preview never blocks).
func TestDryRunSecretRefusesButExitsZero(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".abcd/config/launch-payload.json", `{"includes": ["commands"]}`)
	writeFile(t, root, "commands/x.md", "# doc\ntoken = "+fakeSecret+"\n")

	report, err := DryRun(DryRunRequest{RepoRoot: root, Version: "1.0.0"})
	if err != nil {
		t.Fatalf("dry-run must return nil error on a finding, got %v", err)
	}
	if report.WouldPublish {
		t.Error("WouldPublish must be false in dry-run")
	}
	if report.Scan.HardFails == 0 {
		t.Fatalf("planted secret not caught by the bundle scan: %+v", report.Scan)
	}
	if len(report.WouldRefuseOn) == 0 {
		t.Error("WouldRefuseOn must list the secret hard-fail")
	}
	var sawScanReason bool
	for _, r := range report.WouldRefuseOn {
		if strings.Contains(r, "hard-fail") {
			sawScanReason = true
		}
	}
	if !sawScanReason {
		t.Errorf("expected a scan hard-fail reason, got %v", report.WouldRefuseOn)
	}
	// The scan reports under the bundle logical path.
	if !hasFindingKind(report.Scan, "token:github_pat") {
		t.Errorf("expected github_pat finding, got %+v", report.Scan.Findings)
	}
}

// TestShipBlocksOnSecret is the ship-side of brief AC 1: the same tree blocks.
func TestShipBlocksOnSecret(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".abcd/config/launch-payload.json", `{"includes": ["commands"]}`)
	writeFile(t, root, "commands/x.md", "token = "+fakeSecret+"\n")

	report, err := Ship(ShipRequest{RepoRoot: root, Version: "1.0.0"})
	if !errors.Is(err, ErrShipBlocked) {
		t.Fatalf("ship must return ErrShipBlocked, got %v", err)
	}
	if !report.Blocked || report.WouldPublish {
		t.Errorf("ship must be blocked and not would-publish: %+v", report)
	}
}

// TestDryRunPreflightError proves a bad include config is the one case DryRun
// returns an error (report impossible).
func TestDryRunPreflightError(t *testing.T) {
	root := t.TempDir()
	// No launch-payload.json → LoadIncludes preflight error.
	if _, err := DryRun(DryRunRequest{RepoRoot: root}); err == nil {
		t.Error("expected a preflight error when the include config is missing")
	}
}

// TestShipCleanWouldPublish proves a clean tree with agreeing manifests stops at
// WouldPublish=true with no error and no network call.
func TestShipCleanWouldPublish(t *testing.T) {
	root := t.TempDir()
	// The payload must carry .claude-plugin: a bundle without the manifests is
	// not an installable plugin, which the installability gate now says out loud.
	writeFile(t, root, ".abcd/config/launch-payload.json", `{"includes": [".claude-plugin", "commands", "README.md"]}`)
	writeFile(t, root, "commands/a.md", "clean content\n")
	writeFile(t, root, "README.md", "clean readme\n")
	// adr-19: a clean SOURCE tree is version-ABSENT; the version keys appear
	// only in the rendered payload.
	writeLockstepTree(t, root, "", "", "")

	// The tag set is injected: the fixture is not a repository, and a tag
	// listing that fails is a retention refusal (iss-194).
	report, err := Ship(ShipRequest{RepoRoot: root, Version: "1.2.3", ExistingTags: []Semver{}})
	if err != nil {
		t.Fatalf("clean tree must not error: %v (reasons %v)", err, report.BlockReasons)
	}
	if !report.WouldPublish || report.Blocked {
		t.Errorf("clean tree must stop at would-publish: %+v", report)
	}
	if report.Lockstep.ExitCode != 0 {
		t.Errorf("clean lockstep expected OK, got %+v", report.Lockstep)
	}
}

// TestZeroCoverageRefuses (Finding 1b, launch side) proves a scanner that
// covered zero of the bundle's files — here a valid config that skips every
// bundle file by extension — surfaces as Unavailable, appears in
// WouldRefuseOn, and blocks Ship instead of letting it would-publish.
func TestZeroCoverageRefuses(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".abcd/config/launch-payload.json", `{"includes": ["commands"]}`)
	writeFile(t, root, "commands/a.md", "clean content\n")
	writeFile(t, root, ".abcd/config/pii.json", `{"skip_extensions": [".md"]}`)
	// adr-19: a clean SOURCE tree is version-ABSENT; the version keys appear
	// only in the rendered payload.
	writeLockstepTree(t, root, "", "", "")

	report, err := DryRun(DryRunRequest{RepoRoot: root, Version: "1.2.3"})
	if err != nil {
		t.Fatalf("dry-run preflight must succeed: %v", err)
	}
	if !report.Scan.Unavailable {
		t.Fatalf("zero-coverage scan must be unavailable: %+v", report.Scan)
	}
	if report.WouldPublish {
		t.Error("WouldPublish must be false")
	}
	var sawUnavail bool
	for _, r := range report.WouldRefuseOn {
		if strings.Contains(r, "unavailable") {
			sawUnavail = true
		}
	}
	if !sawUnavail {
		t.Errorf("WouldRefuseOn must cite scanner unavailable, got %v", report.WouldRefuseOn)
	}

	ship, err := Ship(ShipRequest{RepoRoot: root, Version: "1.2.3"})
	if !errors.Is(err, ErrShipBlocked) {
		t.Fatalf("ship must be blocked on zero coverage, got %v", err)
	}
	if ship.WouldPublish || !ship.Blocked {
		t.Errorf("ship must not would-publish on zero coverage: %+v", ship)
	}
}

func hasFindingKind(res scanner.ScanResult, kind string) bool {
	for _, f := range res.Findings {
		if f.Kind == kind {
			return true
		}
	}
	return false
}
