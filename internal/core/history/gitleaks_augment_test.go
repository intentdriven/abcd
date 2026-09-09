package history

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/gitleaks"
	"github.com/intentdriven/abcd/internal/adapter/scanner"
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
// existed. scanGitleaks is NOT overridden here — the real Scan runs, finds no
// config, and returns (nil, nil).
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

	restore := scanGitleaks
	t.Cleanup(func() { scanGitleaks = restore })
	scanGitleaks = func(_, text, logical string) ([]scanner.Finding, error) {
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
	}

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

// TestCaptureGitleaksLoudStagePropagates proves the loud-stage reaches the
// caller: an opted-in-but-absent binary makes Capture fail closed and write
// nothing, naming the opt-in.
func TestCaptureGitleaksLoudStagePropagates(t *testing.T) {
	repoRoot, _ := setupStore(t)

	restore := scanGitleaks
	t.Cleanup(func() { scanGitleaks = restore })
	scanGitleaks = func(_, _, _ string) ([]scanner.Finding, error) {
		return nil, gitleaks.ErrConfiguredNotFound
	}

	_, err := Capture(repoRoot, testRootSHA, []byte("user: hi\n"), CaptureMeta{SessionID: "sess-loud", Kind: "native"})
	if err == nil {
		t.Fatal("expected Capture to fail closed on an armed-but-absent gitleaks")
	}
	if !errors.Is(err, gitleaks.ErrConfiguredNotFound) {
		t.Fatalf("error is not ErrConfiguredNotFound: %v", err)
	}
	if !strings.Contains(err.Error(), "gitleaks configured but not found") {
		t.Fatalf("error does not name the opt-in: %q", err.Error())
	}
	// Nothing was written.
	recs, lerr := List(testRootSHA)
	if lerr != nil {
		t.Fatalf("List: %v", lerr)
	}
	for _, r := range recs {
		if r.SessionID == "sess-loud" {
			t.Error("a record was written despite the loud-stage failure")
		}
	}
}

// TestCaptureRefusesWhenAugmentedSpanIsNotMasked pins GHSA-j7v5-q7x6-v3rp's
// asymmetric-verification limb at the store: an augmented finding whose span
// Redact could not apply (here a line number past the end of the text, which
// Redact silently skips) must make Capture refuse the write. Without a span-
// exact verify the record is written with the secret verbatim and its
// frontmatter counts the finding as redacted — a record asserting cleanliness
// over bytes it holds.
func TestCaptureRefusesWhenAugmentedSpanIsNotMasked(t *testing.T) {
	repoRoot, home := setupStore(t)

	restore := scanGitleaks
	t.Cleanup(func() { scanGitleaks = restore })
	scanGitleaks = func(_, _, logical string) ([]scanner.Finding, error) {
		return []scanner.Finding{{
			File:     logical,
			Line:     999, // a span Redact cannot apply
			Column:   1,
			Kind:     "gitleaks:generic-api-key",
			Severity: scanner.SeverityHardFail,
			Matched:  gitleaksResidueSecret,
		}}, nil
	}

	transcript := strings.Join([]string{
		"user: set the key",
		"api_key = " + gitleaksResidueSecret,
		"assistant: done",
	}, "\n")

	res, err := Capture(repoRoot, testRootSHA, []byte(transcript), CaptureMeta{SessionID: "sess-unsealed", Kind: "native"})
	var rerr *RedactionResidualError
	if !errors.As(err, &rerr) {
		t.Fatalf("Capture = (wrote=%v, err=%v); want a *RedactionResidualError for the unmasked augmented span", res.Wrote, err)
	}
	if res.Wrote {
		t.Error("Capture reported Wrote=true alongside a refusal")
	}
	tdir := filepath.Join(home, ".abcd", "history", testRootSHA, "transcripts")
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
// makes Capture refuse and write nothing, exactly as an armed-but-absent binary
// does (TestCaptureGitleaksLoudStagePropagates). Silently capturing with less
// coverage than the repo armed is the fail-open this store forbids.
func TestCaptureFailsClosedOnUnlocatableGitleaksReport(t *testing.T) {
	repoRoot, _ := setupStore(t)

	restore := scanGitleaks
	t.Cleanup(func() { scanGitleaks = restore })
	scanGitleaks = func(_, _, _ string) ([]scanner.Finding, error) {
		return nil, gitleaks.ErrFindingNotLocated
	}

	_, err := Capture(repoRoot, testRootSHA, []byte("user: hi\n"), CaptureMeta{SessionID: "sess-unlocated", Kind: "native"})
	if !errors.Is(err, gitleaks.ErrFindingNotLocated) {
		t.Fatalf("Capture did not fail closed on an unlocatable gitleaks report: %v", err)
	}
	recs, lerr := List(testRootSHA)
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
// report, the way an opted-in repo with a gitleaks binary reaches it. The
// binary is a mode-0755 file outside the repo root, which is all admitBinary
// asks of it; cannedGitleaks never executes it.
func armGitleaks(t *testing.T, report string) {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "gitleaks")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	a := &gitleaks.Adapter{
		LookPath: func(string) (string, error) { return bin, nil },
		Runner:   cannedGitleaks{report: report},
	}
	restore := scanGitleaks
	t.Cleanup(func() { scanGitleaks = restore })
	scanGitleaks = func(repoRoot, text, logical string) ([]scanner.Finding, error) {
		return a.Augment(context.Background(), repoRoot, gitleaks.Config{Enabled: true}, text, logical)
	}
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
			armGitleaks(t, string(reported))

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
