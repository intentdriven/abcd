package launch

import (
	"errors"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
)

// TestSvgPayloadSecretRefuses is the GHSA-5mmm-3whv-3rqp SVG axis: a secret in
// an included, extension-"skipped" text asset (docs/assets/img/x.svg) must be
// scanned, hard-fail, drive DryRun.WouldRefuseOn non-empty, and block Ship. The
// old skip-by-extension shipped the raw bytes unscanned.
func TestSvgPayloadSecretRefuses(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".abcd/config/launch-payload.json", `{"includes": ["docs"]}`)
	// FAKE token shape only: ghp_ + 36 chars, matching \bghp_[A-Za-z0-9]{36,}.
	token := "ghp_" + strings.Repeat("a", 36)
	writeFile(t, root, "docs/assets/img/x.svg",
		`<svg xmlns="http://www.w3.org/2000/svg"><!-- `+token+` --></svg>`+"\n")

	report, err := DryRun(DryRunRequest{RepoRoot: root, Version: "1.0.0"})
	if err != nil {
		t.Fatalf("dry-run must return nil error on a finding, got %v", err)
	}
	if report.Scan.HardFails == 0 {
		t.Fatalf("secret in the SVG asset was not caught by the scan: %+v", report.Scan)
	}
	if len(report.WouldRefuseOn) == 0 {
		t.Errorf("WouldRefuseOn must be non-empty for a secret in a payload SVG")
	}
	if report.WouldPublish {
		t.Errorf("WouldPublish must be false with a payload secret")
	}

	ship, err := Ship(ShipRequest{RepoRoot: root, Version: "1.0.0"})
	if !errors.Is(err, ErrShipBlocked) {
		t.Fatalf("ship must return ErrShipBlocked for a secret in a payload SVG, got %v", err)
	}
	if !ship.Blocked || ship.WouldPublish {
		t.Errorf("ship must be blocked and not would-publish: %+v", ship)
	}
}

// TestUnscannedPayloadRefuses is the GHSA-5mmm-3whv-3rqp binary-smuggle axis: a
// NUL-prefixed file whose extension is NOT on the reviewed skip list is
// classified unscannable and surfaced in Unscanned; the launch gate must fail
// closed on it rather than let "some other file scanned" count as coverage.
func TestUnscannedPayloadRefuses(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".abcd/config/launch-payload.json", `{"includes": ["commands"]}`)
	// A leading NUL makes an otherwise-.md file read as binary → Unscanned.
	writeFile(t, root, "commands/clean.md", "wholly clean documentation\n")
	writeFile(t, root, "commands/smuggle.md", "\x00ghp_"+strings.Repeat("a", 40))

	report, err := DryRun(DryRunRequest{RepoRoot: root, Version: "1.0.0"})
	if err != nil {
		t.Fatalf("dry-run must return nil error, got %v", err)
	}
	var sawUnscanned bool
	for _, p := range report.Scan.Unscanned {
		if p == "commands/smuggle.md" {
			sawUnscanned = true
		}
	}
	if !sawUnscanned {
		t.Fatalf("unscannable payload file must be surfaced in Unscanned: %+v", report.Scan)
	}
	var sawRefusal bool
	for _, r := range report.WouldRefuseOn {
		if strings.Contains(r, "unscanned") {
			sawRefusal = true
		}
	}
	if !sawRefusal {
		t.Errorf("WouldRefuseOn must fail closed on an unscanned payload file, got %v", report.WouldRefuseOn)
	}

	ship, err := Ship(ShipRequest{RepoRoot: root, Version: "1.0.0"})
	if !errors.Is(err, ErrShipBlocked) {
		t.Fatalf("ship must be blocked on an unscanned payload file, got %v", err)
	}
	if ship.WouldPublish {
		t.Errorf("ship must not would-publish with an unscanned payload file: %+v", ship)
	}
}

// TestBinaryPayloadSecretRefuses is the GHSA-9wv7-88w3-f77m axis
// (iss-2608291807454357): a secret inside an included file whose EXTENSION is
// on the binary skip list (docs/assets/notes.png, no image data needed) must
// have its bytes scanned by the secret rules, hard-fail, be reported as
// ScannedBinary rather than a skip, drive WouldRefuseOn, and block Ship. The
// filename-keyed skip shipped the raw bytes with zero findings.
func TestBinaryPayloadSecretRefuses(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".abcd/config/launch-payload.json", `{"includes": ["docs"]}`)
	writeFile(t, root, "docs/README.md", "clean documentation\n")
	// FAKE token shape only, built at runtime: ghp_ + 36 chars.
	token := "ghp_" + strings.Repeat("b", 36)
	writeFile(t, root, "docs/assets/notes.png", "not an image; token="+token+"\n")

	report, err := DryRun(DryRunRequest{RepoRoot: root, Version: "1.0.0"})
	if err != nil {
		t.Fatalf("dry-run must return nil error on a finding, got %v", err)
	}
	if report.Scan.HardFails == 0 {
		t.Fatalf("secret in the skip-listed .png was not caught by the scan: %+v", report.Scan)
	}
	var sawBinary bool
	for _, p := range report.Scan.ContentUnverified {
		if p == "docs/assets/notes.png" {
			sawBinary = true
		}
	}
	if !sawBinary {
		t.Errorf("the .png must be reported as ContentUnverified, got %+v", report.Scan)
	}
	if len(report.WouldRefuseOn) == 0 || report.WouldPublish {
		t.Errorf("a secret in a payload binary must refuse: refuse=%v publish=%v", report.WouldRefuseOn, report.WouldPublish)
	}

	ship, err := Ship(ShipRequest{RepoRoot: root, Version: "1.0.0"})
	if !errors.Is(err, ErrShipBlocked) {
		t.Fatalf("ship must return ErrShipBlocked for a secret in a payload binary, got %v", err)
	}
	if !ship.Blocked || ship.WouldPublish {
		t.Errorf("ship must be blocked and not would-publish: %+v", ship)
	}
}

// TestRepoPayloadBinariesScanClean is the false-positive guard for the same
// fix: this repository's own payload carries genuine binary assets, and the
// secret rules run over their bytes must find nothing. Text-file findings are
// deliberately not asserted here — an environmental identity match on a text
// file (iss-2608291444328326) is a different rule and a different file class.
func TestRepoPayloadBinariesScanClean(t *testing.T) {
	root := repoRootForTest(t)
	bundle, err := ResolveBundle(root, nil)
	if err != nil {
		t.Fatalf("resolve the payload bundle: %v", err)
	}
	scan := scanBundle(root, bundle)
	if scan.Unavailable {
		t.Fatalf("scanner unavailable on the repo's own payload: %s", scan.UnavailableReason)
	}
	if len(scan.ContentDecoded)+len(scan.ContentUnverified) == 0 {
		t.Fatalf("the repo payload is expected to carry at least one image asset; none was reported: %+v", scan)
	}
	// The decoder runs on the real payload, not only on fixtures: the repo's
	// own images are decoded — IDAT and text chunks inflated and scanned —
	// and none of that manufactures a finding (iss-2608291832160371).
	if len(scan.ContentDecoded) == 0 {
		t.Errorf("the repo payload's image assets must be decoded, not merely byte-scanned: %+v", scan)
	}
	binary := map[string]bool{}
	for _, p := range append(append(append([]string{}, scan.ScannedBinary...), scan.ContentUnverified...), scan.ContentDecoded...) {
		binary[p] = true
	}
	for _, f := range scan.Findings {
		if binary[f.File] {
			t.Errorf("byte rule %s tripped on a genuine binary asset %s (line %d)", f.Kind, f.File, f.Line)
		}
	}
	for p, why := range scan.ContentUnverifiedWhy {
		if why == "" {
			t.Errorf("an unverified payload file must say why it could not be read: %s", p)
		}
	}
	if len(scan.Unscanned) != 0 {
		t.Errorf("the repo payload must leave no coverage gap: %v (%v)", scan.Unscanned, scan.UnscannedWhy)
	}
}

// TestScanDetailSeparatesCoverageTiers: the gate row must not fold the
// byte-scanned files into the full-rule-set count, and must keep the decoded
// tier ("decoded and clean") apart from the tier the scan could not read at
// all — never one green (iss-2608291832160371).
func TestScanDetailSeparatesCoverageTiers(t *testing.T) {
	detail := scanDetail(scanner.ScanResult{
		FilesScanned:      3,
		ScannedBinary:     []string{"a.ico"},
		ContentDecoded:    []string{"r.zip"},
		ContentUnverified: []string{"p.tgz", "q.png"},
	})
	want := "scanned 3 files with the full rule set, 1 binary (byte rules only), 1 decoded (entries scanned), 2 compressed (not content-verified), 0 hard-fails"
	if detail != want {
		t.Fatalf("scanDetail = %q, want %q", detail, want)
	}
}

// TestUnscannedRefusalCarriesWhy: the refusal line names the reason, so an
// asset over the cap is distinguishable from an I/O error.
func TestUnscannedRefusalCarriesWhy(t *testing.T) {
	reasons := scanRefusals(scanner.ScanResult{
		Unscanned:    []string{"big.pdf"},
		UnscannedWhy: map[string]string{"big.pdf": "over the 4 MiB scan cap"},
	})
	if len(reasons) != 1 || !strings.Contains(reasons[0], "big.pdf (over the 4 MiB scan cap)") {
		t.Fatalf("refusal must carry the why, got %v", reasons)
	}
}
