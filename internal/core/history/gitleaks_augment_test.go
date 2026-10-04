package history

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/adapter/gitleaks"
	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/adapter/scanner/augmenttest"
	"github.com/intentdriven/abcd/internal/testsecret"
)

// A bare, unanchored high-entropy value with a key name and delimiter — the
// exact residue iss-96 pins: the native prefix set passes it through, and it is
// what an opted-in gitleaks catches (its generic-api-key rule). Built at
// runtime (never a source literal), so the full-history secret scan has nothing
// to find — see the principle secret-shaped-fixtures-at-runtime.
var gitleaksResidueSecret = testsecret.Synthetic(96, 40)

// TestCaptureDefaultOffStoresResidueVerbatim proves the default-off path is
// unchanged: with no gitleaks opt-in config in the repo, the residue value the
// native scanner misses is stored verbatim, exactly as before this adapter
// existed. No augmenter is installed here, so no scanner.New wires one, as for a
// repository that did not opt in.
func TestCaptureDefaultOffStoresResidueVerbatim(t *testing.T) {
	repoRoot, _ := setupStore(t)

	transcript := strings.Join([]string{
		"user: set the key",
		"api_key = " + gitleaksResidueSecret,
		"assistant: done",
	}, "\n")

	res, err := Capture(repoRoot, testRootSHA, []byte(transcript), CaptureMeta{SessionID: "sess-defoff", Kind: "native"})
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if !res.Wrote {
		t.Fatal("expected Wrote=true")
	}
	onDisk, err := os.ReadFile(res.Record.Path)
	if err != nil {
		t.Fatalf("read record: %v", err)
	}
	// Default-off: the native scanner does not catch it, and no adapter ran, so
	// the value is present verbatim — the behaviour the iss-96 pin records.
	if !bytes.Contains(onDisk, []byte(gitleaksResidueSecret)) {
		t.Error("default-off path changed behaviour: the residue value was redacted with no opt-in")
	}
}

// TestCaptureFoldsGitleaksFindings proves the augmentation path end to end: with
// the seam injected to stand in for an opted-in gitleaks, the residue value is
// redacted out of the stored record and counted in the audit buckets.
func TestCaptureFoldsGitleaksFindings(t *testing.T) {
	repoRoot, _ := setupStore(t)

	setAugmenter(t, func(text, logical string) ([]scanner.Finding, error) {
		// Locate the residue value as the real adapter would and emit a finding.
		lines := strings.Split(text, "\n")
		var out []scanner.Finding
		for i, ln := range lines {
			if col := strings.Index(ln, gitleaksResidueSecret); col >= 0 {
				out = append(out, scanner.Finding{
					File:     logical,
					Line:     i + 1,
					Column:   col + 1,
					Kind:     "gitleaks:generic-api-key",
					Severity: scanner.SeverityHardFail,
					Matched:  gitleaksResidueSecret,
					Snippet:  ln,
				})
			}
		}
		return out, nil
	})

	transcript := strings.Join([]string{
		"user: set the key",
		"api_key = " + gitleaksResidueSecret,
		"assistant: done",
	}, "\n")

	res, err := Capture(repoRoot, testRootSHA, []byte(transcript), CaptureMeta{SessionID: "sess-fold", Kind: "native"})
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	onDisk, err := os.ReadFile(res.Record.Path)
	if err != nil {
		t.Fatalf("read record: %v", err)
	}
	if bytes.Contains(onDisk, []byte(gitleaksResidueSecret)) {
		t.Errorf("the gitleaks-augmented secret leaked into the stored record:\n%s", onDisk)
	}
	if res.Record.Secrets < 1 {
		t.Errorf("expected the gitleaks finding counted in Secrets, got %d", res.Record.Secrets)
	}
}

// TestCaptureRecordsTheGitleaksGap is the 2026-09-25 ruling at the store: a
// repository that armed gitleaks with no binary installed still has its
// transcript stored, scanned by the native scanner, and the receipt names the
// gap and the opt-in (iss-2608291814575788). The augmenter is never asked to
// scan.
func TestCaptureRecordsTheGitleaksGap(t *testing.T) {
	repoRoot, _ := setupStore(t)
	aug := &augmenttest.Func{
		Err: fmt.Errorf("%w: not on PATH and no path configured", gitleaks.ErrConfiguredNotFound),
		F: func(_, _ string) ([]scanner.Finding, error) {
			t.Error("a not-found augmenter was asked to scan")
			return nil, nil
		},
	}
	augmenttest.Install(t, aug)

	res, err := Capture(repoRoot, testRootSHA, []byte("user: hi\n"), CaptureMeta{SessionID: "sess-gap", Kind: "native"})
	if err != nil {
		t.Fatalf("Capture refused on the gap: %v", err)
	}
	if !res.Wrote {
		t.Fatal("the transcript was not stored")
	}
	if !strings.Contains(res.ScanGap, "gitleaks configured but not found") {
		t.Fatalf("the receipt does not name the gap: %q", res.ScanGap)
	}
}

// TestCaptureRefusesWhenAugmentedSpanIsNotMasked pins GHSA-j7v5-q7x6-v3rp at
// the store: an augmented finding whose span Redact could not apply (here a
// line number past the end of the text, which Redact silently skips) must make
// Capture refuse the write. The scanner refuses such a report before Redact
// sees it (it keeps only findings located in the text, and degrades on any
// other), so the record is never written with the secret verbatim while its
// frontmatter counts the finding as redacted.
func TestCaptureRefusesWhenAugmentedSpanIsNotMasked(t *testing.T) {
	repoRoot, home := setupStore(t)

	setAugmenter(t, func(_, logical string) ([]scanner.Finding, error) {
		return []scanner.Finding{{
			File:     logical,
			Line:     999, // a span Redact cannot apply
			Column:   1,
			Kind:     "gitleaks:generic-api-key",
			Severity: scanner.SeverityHardFail,
			Matched:  gitleaksResidueSecret,
		}}, nil
	})

	transcript := strings.Join([]string{
		"user: set the key",
		"api_key = " + gitleaksResidueSecret,
		"assistant: done",
	}, "\n")

	res, err := Capture(repoRoot, testRootSHA, []byte(transcript), CaptureMeta{SessionID: "sess-unsealed", Kind: "native"})
	if err == nil || !strings.Contains(err.Error(), "not located") {
		t.Fatalf("Capture = (wrote=%v, err=%v); want a refusal of the unlocated augmented span", res.Wrote, err)
	}
	if res.Wrote {
		t.Error("Capture reported Wrote=true alongside a refusal")
	}
	tdir := abcdhome.Path(home, "transcripts", testRootSHA, "records")
	entries, err := os.ReadDir(tdir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(tdir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(data, []byte(gitleaksResidueSecret)) {
			t.Errorf("the secret is on disk in %s despite the refusal", e.Name())
		}
	}
}

// TestCaptureFailsClosedOnUnlocatableGitleaksReport is the store-level echo of
// the adapter's ErrFindingNotLocated: a report the adapter could not place
// degrades the scanner, and Capture refuses and writes nothing, as it does on
// a degraded pii.json. Silently capturing with less coverage than the repo
// armed, over a run that failed, is the fail-open this store forbids.
func TestCaptureFailsClosedOnUnlocatableGitleaksReport(t *testing.T) {
	repoRoot, _ := setupStore(t)

	setAugmenter(t, func(_, _ string) ([]scanner.Finding, error) {
		return nil, gitleaks.ErrFindingNotLocated
	})

	_, err := Capture(repoRoot, testRootSHA, []byte("user: hi\n"), CaptureMeta{SessionID: "sess-unlocated", Kind: "native"})
	if err == nil || !strings.Contains(err.Error(), gitleaks.ErrFindingNotLocated.Error()) {
		t.Fatalf("Capture did not fail closed on an unlocatable gitleaks report: %v", err)
	}
	recs, lerr := List(repoRoot, testRootSHA)
	if lerr != nil {
		t.Fatalf("List: %v", lerr)
	}
	for _, r := range recs {
		if r.SessionID == "sess-unlocated" {
			t.Error("a record was written despite the refused augmentation")
		}
	}
}

// cannedGitleaks stands in for the gitleaks binary: it returns a fixed JSON
// report, so the REAL adapter conversion runs (toFindings, the code under test)
// with no process spawned.
type cannedGitleaks struct{ report string }

func (c cannedGitleaks) Run(_ context.Context, _, _ string) ([]byte, error) {
	return []byte(c.report), nil
}

// armGitleaks points the store's seam at the real adapter driven by a canned
// report, the way an opted-in repo with a gitleaks binary reaches it: the
// repository's own .abcd/config/gitleaks.json arms it, and the augmenter the
// adapter builds from that config is the one every scanner.New wires. The
// binary is a mode-0755 file outside the repo root, which is all admitBinary
// asks of it; cannedGitleaks never executes it.
func armGitleaks(t *testing.T, repoRoot, report string) {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "gitleaks")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	a := &gitleaks.Adapter{
		LookPath: func(string) (string, error) { return bin, nil },
		Runner:   cannedGitleaks{report: report},
	}
	cfgDir := filepath.Join(repoRoot, ".abcd", "config")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "gitleaks.json"), []byte(`{"schema_version":1,"enabled":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	aug := a.AugmenterFor(repoRoot)
	if err := aug.Available(); err != nil {
		t.Fatalf("the armed augmenter is unavailable: %v", err)
	}
	augmenttest.Install(t, aug)
}

// setAugmenter installs an augmenter built from f as the default every
// scanner.New wires, for the rest of the test: an error f returns is kept and
// reported as the augmenter's state, as a failed gitleaks run is.
func setAugmenter(t *testing.T, f func(text, logical string) ([]scanner.Finding, error)) {
	t.Helper()
	augmenttest.Install(t, &augmenttest.Func{F: f})
}

// TestCaptureSealsEveryRecurrenceOfAnAugmentedFragment pins one scope for
// detection and verification (iss-2609020231145566). A multi-line reported
// value is located as a whole and split into one finding per line it crosses,
// while the store verifies each FRAGMENT by presence anywhere in the redacted
// text — so a line of the value that also occurs benignly elsewhere (a quoted
// end-of-key marker, a repeated header) was never sealed at its second site
// and tripped the residual guard on every capture: the transcript could never
// be stored, and the drain kept its unredacted staged copy forever. Detection
// now locates each fragment across the whole text, so every occurrence of every
// fragment is sealed and the two scopes agree.
//
// The markers are deliberately NOT PEM-shaped, as elsewhere in this file: a
// native pattern that matched them would seal the recurrence for the wrong
// reason and mask the regression.
func TestCaptureSealsEveryRecurrenceOfAnAugmentedFragment(t *testing.T) {
	const begin = "===BEGIN SYNTHETIC KEYBLOCK==="
	const end = "===END SYNTHETIC KEYBLOCK==="
	body1 := testsecret.Synthetic(102, 40)
	body2 := testsecret.Synthetic(103, 40)
	block := strings.Join([]string{begin, body1, body2, end}, "\n")

	reported, err := json.Marshal([]map[string]string{{"RuleID": "private-key", "Secret": block, "Match": block}})
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name    string
		recurs  string // a line the transcript repeats outside the reported value
		session string
	}{
		{"clean", "", "sess-frag-clean"},
		{"header repeated", begin, "sess-frag-header"},
		{"footer repeated", end, "sess-frag-footer"},
		{"body repeated", body1, "sess-frag-body"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repoRoot, _ := setupStore(t)
			armGitleaks(t, repoRoot, string(reported))

			lines := []string{"user: here is the key", block, "assistant: stored"}
			if tc.recurs != "" {
				lines = append(lines, "user: and this line again", tc.recurs, "assistant: noted")
			}
			transcript := strings.Join(lines, "\n") + "\n"

			res, err := Capture(repoRoot, testRootSHA, []byte(transcript), CaptureMeta{SessionID: tc.session, Kind: "native"})
			if err != nil {
				t.Fatalf("Capture refused a transcript it can seal: %v", err)
			}
			if !res.Wrote {
				t.Fatal("expected the record to be written")
			}
			onDisk, err := os.ReadFile(res.Record.Path)
			if err != nil {
				t.Fatal(err)
			}
			for _, secret := range []string{body1, body2} {
				if bytes.Contains(onDisk, []byte(secret)) {
					t.Errorf("a line of the reported value survived redaction:\n%s", onDisk)
				}
			}
			if tc.recurs != "" && bytes.Contains(onDisk, []byte(tc.recurs)) {
				t.Errorf("the recurrence of %q outside the value was never sealed:\n%s", tc.recurs, onDisk)
			}
		})
	}
}

// TestDrainRecordsTheGitleaksGap: the automatic path (SessionStart's drain)
// stores the staged transcript on the native scanner when the armed gitleaks
// is not installed, and its receipt carries the gap, so the notice can say so
// rather than the gap vanishing inside an automatic pass.
func TestDrainRecordsTheGitleaksGap(t *testing.T) {
	repoRoot, _ := setupStore(t)
	augmenttest.Install(t, &augmenttest.Func{
		Err: fmt.Errorf("%w: not on PATH and no path configured", gitleaks.ErrConfiguredNotFound),
	})
	if _, err := Stage(repoRoot, testRootSHA, mainStage("sess-drain-gap"), []byte("user: hi\n")); err != nil {
		t.Fatal(err)
	}
	res, err := Drain(repoRoot, testRootSHA, DrainBudget{})
	if err != nil {
		t.Fatalf("Drain: %v", err)
	}
	if len(res.Captured) != 1 {
		t.Fatalf("captured %d, want 1 (failed: %+v)", len(res.Captured), res.Failed)
	}
	if !strings.Contains(res.ScanGap, "gitleaks configured but not found") {
		t.Fatalf("the drain's receipt does not carry the gap: %q", res.ScanGap)
	}
}
