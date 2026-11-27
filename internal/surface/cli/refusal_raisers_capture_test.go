package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/findings"
)

// refusal_raisers_capture_test.go — the probes for capture's and explain's
// own codes (TestRaisedByIsWhatTheVerbsRaise).

var (
	_ = registerRaiserProbes("capture", captureRaiserProbes)
	_ = registerRaiserProbes("capture-owed", captureOwedRaiserProbes)
)

// captureOwedRaiserProbes builds a checkout with one commit and one entry on
// its findings list, under a home of the probes' own, and returns the probes
// of `capture owed`.
func captureOwedRaiserProbes(t *testing.T) []raiserProbe {
	repo := findingsRepo(t)
	h := plantEntry(t, repo, findings.KindUntested, "a finding the probes leave on the list")
	return []raiserProbe{
		probe("capture/invalid-input", "capture", "owed", "f-00000000", "--refuted", "the parser keeps the line"),
		probe("capture/invalid-input", "capture", "owed", h, "--own", "the base fails the same check"),
		probe("capture/unknown-issue", "capture", "owed", h, "--captured", unknownIssueID),
	}
}

// Ids no fixture writes, one per record kind a capture verb looks up.
const (
	unknownIssueID     = "iss-2601010101010101"
	unknownReadingItem = "rdi-2601010101010101"
	unknownReframeID   = "rfm-2601010101010101"
	probeGrounds       = "pursued: the tension is real and the next reading shows it"
	probeText          = "the proposal nobody expected ranked first"
)

// captureIssueForProbe files one issue at severity and returns its id.
func captureIssueForProbe(t *testing.T, text, severity string) string {
	t.Helper()
	var stdout, stderr strings.Builder
	if code := Run([]string{"capture", text, "--remedy", cliTestRemedy, "--severity", severity, "--json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("capture: exit %d\n%s%s", code, stdout.String(), stderr.String())
	}
	var got struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(stdout.String()), &got); err != nil || got.ID == "" {
		t.Fatalf("capture printed no id: %v\n%s", err, stdout.String())
	}
	return got.ID
}

// captureRaiserProbes builds a fixture ledger with an open, a minor and a
// resolved issue and the reading fixtures, and returns capture's and
// explain's probes.
func captureRaiserProbes(t *testing.T) []raiserProbe {
	repo := captureLedgerRepo(t)
	open := captureIssueForProbe(t, "the parser drops the last line of a file", "major")
	minor := captureIssueForProbe(t, "the reader breaks on an empty input file", "minor")
	closed := captureIssueForProbe(t, "the writer loses the final newline", "major")
	if code, _, out := codedRefusal(t, "capture", "resolve", closed, "fixed by the writer change", "--impact", "fix"); code != 0 {
		t.Fatalf("resolving the fixture issue: exit %d\n%s", code, out)
	}
	const detectionRun, detectionItem = "rdg-2608300000000011", "rdi-2608300000000012"
	writeReadingFixture(t, repo, detectionRun, detectionItem)
	writeWideningFixture(t, repo, false)

	return []raiserProbe{
		probe("capture/missing-remedy", "capture", "the parser drops a line"),
		probe("capture/remedy-machine-specific", "capture", "the parser drops a line", "--remedy", "none (filed automatically)"),

		probe("capture/unknown-issue", "capture", "defer", unknownIssueID, "--after", "v0.1.0", "--reason", "the fix needs the parser rewrite"),
		probe("capture/unknown-issue", "capture", "link", unknownIssueID, "--blocked-by", open),
		probe("capture/unknown-issue", "capture", "promote", unknownIssueID),
		probe("capture/unknown-issue", "capture", "remedy", unknownIssueID, "make the map before the first write"),
		probe("capture/unknown-issue", "capture", "resolve", unknownIssueID, "fixed by the parser change", "--impact", "fix"),
		probe("capture/unknown-issue", "capture", "wontfix", unknownIssueID, "the behaviour is the documented one"),

		probe("capture/unknown-reading-record", "capture", "admit", unknownReadingItem, "--grounds", "the widened configuration is one the next release serves"),
		probe("capture/unknown-reading-record", "capture", "disposition", unknownReadingItem, "--state", "accepted", "--grounds", probeGrounds),
		probe("capture/unknown-reading-record", "capture", "promote", unknownReadingItem),
		probe("capture/unknown-reading-record", "capture", "reframe", "--complete", unknownReframeID),
		probe("capture/unknown-reading-record", "capture", "reframe", "--occasioned-by", unknownReadingItem, "--grounds", probeGrounds, "--open"),
		probe("capture/unknown-reading-record", "capture", "surprise", "--occasioned-by", unknownReadingItem, probeText),

		probe("capture/not-open", "capture", "resolve", closed, "fixed by the writer change", "--impact", "fix"),
		probe("capture/not-open", "capture", "wontfix", closed, "the behaviour is the documented one"),
		probe("capture/not-open", "capture", "defer", closed, "--after", "v0.1.0", "--reason", "the fix needs the parser rewrite"),
		probe("capture/not-open", "capture", "remedy", closed, "make the map before the first write"),

		probe("capture/insufficient-grounds", "capture", "resolve", open, "fixed by the parser change", "--impact", "fix", "--grounds", "bogus"),
		probe("capture/insufficient-grounds", "capture", "wontfix", open, "the behaviour is the documented one", "--grounds", "bogus"),
		probe("capture/insufficient-grounds", "capture", "promote", open, "--grounds", "bogus"),
		probe("capture/insufficient-grounds", "capture", "admit", unknownReadingItem, "--grounds", "thin"),
		probe("capture/insufficient-grounds", "capture", "surprise", "--occasioned-by", unknownReadingItem, "thin"),
		probe("capture/insufficient-grounds", "capture", "reframe", "--occasioned-by", unknownReadingItem, "--grounds", "thin", "--open"),

		probe("capture/invalid-input", "capture", "the third thing breaks on empty input", "--remedy", cliTestRemedy, "--found-at", "../elsewhere"),
		probe("capture/invalid-input", "capture", "mentions", "--ref", "zzz-no-such-ref"),
		probe("capture/invalid-input", "capture", "resolve", open, "fixed by the parser change", "--impact", "fix", "--shipped-in", "not-a-tag"),
		probe("capture/invalid-input", "capture", "link", open, "--blocked-by", open),
		probe("capture/invalid-input", "capture", "defer", minor, "--after", "v0.1.0", "--reason", "the fix needs the parser rewrite"),
		probe("capture/invalid-input", "capture", "remedy", open, ""),
		probe("capture/invalid-input", "capture", "surprise", "--occasioned-by", open, probeText),
		probe("capture/invalid-input", "capture", "reframe", "--occasioned-by", unknownReframeID, "--grounds", probeGrounds, "--open"),
		probe("capture/invalid-input", "capture", "disposition", detectionItem, "--state", "bogus", "--grounds", probeGrounds),
		probe("capture/invalid-input", "capture", "admit", detectionItem, "--grounds", "the widened configuration is one the next release serves"),

		probe("capture/not-characterized", "capture", "disposition", admitItem, "--state", "accepted", "--grounds", probeGrounds),
		probe("capture/not-characterized", "capture", "admit", admitItem, "--grounds", "the widened configuration is one the next release serves"),

		probe("explain/unknown-path", "explain", "zzz-no-such-verb"),
		probe("explain/unrecognized-input", "explain", "capture", "--input", "zzz"),
		probe("explain/unknown-code", "explain", "zzz/no-such-code"),
	}
}
