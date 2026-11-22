package cli

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/refusal"
)

// refusal_raisers_test.go — a verb's own code names the commands that raise
// it (`abcd explain <code>` prints them as "Raised by"), and the list is held
// by driving each command into the refusal rather than kept by hand
// (spc-2610301431108175 step 1).

// raiserProbe is one refused call and the code it must carry.
type raiserProbe struct {
	args []string
	want refusal.Code
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

// raiserPath is the command path a probe's arguments run: the leading words
// that name commands of the tree.
func raiserPath(args []string) string {
	root := NewRootCommand()
	path := []string{}
	for _, w := range args {
		if findByPath(root, append(slices.Clone(path), w)) == nil {
			break
		}
		path = append(path, w)
	}
	return strings.Join(append([]string{"abcd"}, path...), " ")
}

// TestRaisedByIsWhatTheVerbsRaise drives every command a verb's own code
// names into that refusal, and every refusal it can reach in a fixture
// ledger into its code, and holds the registry's RaisedBy to exactly the
// pairs seen: a listed command that never raises the code fails, and so does
// a command seen raising a code it is not listed for. Usage codes are raised
// by every command and list none.
func TestRaisedByIsWhatTheVerbsRaise(t *testing.T) {
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

	probes := []raiserProbe{
		{[]string{"capture", "the parser drops a line"}, refusal.CaptureRemedyMissing},
		{[]string{"capture", "the parser drops a line", "--remedy", "none (filed automatically)"}, refusal.CaptureRemedyMachineValue},

		{[]string{"capture", "defer", unknownIssueID, "--after", "v0.1.0", "--reason", "the fix needs the parser rewrite"}, refusal.CaptureUnknownIssue},
		{[]string{"capture", "link", unknownIssueID, "--blocked-by", open}, refusal.CaptureUnknownIssue},
		{[]string{"capture", "promote", unknownIssueID}, refusal.CaptureUnknownIssue},
		{[]string{"capture", "remedy", unknownIssueID, "make the map before the first write"}, refusal.CaptureUnknownIssue},
		{[]string{"capture", "resolve", unknownIssueID, "fixed by the parser change", "--impact", "fix"}, refusal.CaptureUnknownIssue},
		{[]string{"capture", "wontfix", unknownIssueID, "the behaviour is the documented one"}, refusal.CaptureUnknownIssue},

		{[]string{"capture", "admit", unknownReadingItem, "--grounds", "the widened configuration is one the next release serves"}, refusal.CaptureUnknownReadingRecord},
		{[]string{"capture", "disposition", unknownReadingItem, "--state", "accepted", "--grounds", probeGrounds}, refusal.CaptureUnknownReadingRecord},
		{[]string{"capture", "promote", unknownReadingItem}, refusal.CaptureUnknownReadingRecord},
		{[]string{"capture", "reframe", "--complete", unknownReframeID}, refusal.CaptureUnknownReadingRecord},
		{[]string{"capture", "reframe", "--occasioned-by", unknownReadingItem, "--grounds", probeGrounds, "--open"}, refusal.CaptureUnknownReadingRecord},
		{[]string{"capture", "surprise", "--occasioned-by", unknownReadingItem, probeText}, refusal.CaptureUnknownReadingRecord},

		{[]string{"capture", "resolve", closed, "fixed by the writer change", "--impact", "fix"}, refusal.CaptureNotOpen},
		{[]string{"capture", "wontfix", closed, "the behaviour is the documented one"}, refusal.CaptureNotOpen},
		{[]string{"capture", "defer", closed, "--after", "v0.1.0", "--reason", "the fix needs the parser rewrite"}, refusal.CaptureNotOpen},
		{[]string{"capture", "remedy", closed, "make the map before the first write"}, refusal.CaptureNotOpen},

		{[]string{"capture", "resolve", open, "fixed by the parser change", "--impact", "fix", "--grounds", "bogus"}, refusal.CaptureGroundsRefused},
		{[]string{"capture", "wontfix", open, "the behaviour is the documented one", "--grounds", "bogus"}, refusal.CaptureGroundsRefused},
		{[]string{"capture", "promote", open, "--grounds", "bogus"}, refusal.CaptureGroundsRefused},
		{[]string{"capture", "admit", unknownReadingItem, "--grounds", "thin"}, refusal.CaptureGroundsRefused},
		{[]string{"capture", "surprise", "--occasioned-by", unknownReadingItem, "thin"}, refusal.CaptureGroundsRefused},
		{[]string{"capture", "reframe", "--occasioned-by", unknownReadingItem, "--grounds", "thin", "--open"}, refusal.CaptureGroundsRefused},

		{[]string{"capture", "the third thing breaks on empty input", "--remedy", cliTestRemedy, "--found-at", "../elsewhere"}, refusal.CaptureInputInvalid},
		{[]string{"capture", "mentions", "--ref", "zzz-no-such-ref"}, refusal.CaptureInputInvalid},
		{[]string{"capture", "resolve", open, "fixed by the parser change", "--impact", "fix", "--shipped-in", "not-a-tag"}, refusal.CaptureInputInvalid},
		{[]string{"capture", "link", open, "--blocked-by", open}, refusal.CaptureInputInvalid},
		{[]string{"capture", "defer", minor, "--after", "v0.1.0", "--reason", "the fix needs the parser rewrite"}, refusal.CaptureInputInvalid},
		{[]string{"capture", "remedy", open, ""}, refusal.CaptureInputInvalid},
		{[]string{"capture", "surprise", "--occasioned-by", open, probeText}, refusal.CaptureInputInvalid},
		{[]string{"capture", "reframe", "--occasioned-by", unknownReframeID, "--grounds", probeGrounds, "--open"}, refusal.CaptureInputInvalid},
		{[]string{"capture", "disposition", detectionItem, "--state", "bogus", "--grounds", probeGrounds}, refusal.CaptureInputInvalid},
		{[]string{"capture", "admit", detectionItem, "--grounds", "the widened configuration is one the next release serves"}, refusal.CaptureInputInvalid},

		{[]string{"capture", "disposition", admitItem, "--state", "accepted", "--grounds", probeGrounds}, refusal.CaptureNotCharacterized},
		{[]string{"capture", "admit", admitItem, "--grounds", "the widened configuration is one the next release serves"}, refusal.CaptureNotCharacterized},

		{[]string{"explain", "zzz-no-such-verb"}, refusal.ExplainUnknownPath},
		{[]string{"explain", "capture", "--input", "zzz"}, refusal.ExplainUnknownInput},
		{[]string{"explain", "zzz/no-such-code"}, refusal.ExplainUnknownCode},
	}

	seen := map[refusal.Code]map[string]bool{}
	for _, p := range probes {
		code, env, out := codedRefusal(t, p.args...)
		if code != 2 || refusal.Code(env.Code) != p.want {
			t.Errorf("abcd %s: exit %d code %q, want exit 2 and %s:\n%s", strings.Join(p.args, " "), code, env.Code, p.want, out)
			continue
		}
		if seen[p.want] == nil {
			seen[p.want] = map[string]bool{}
		}
		seen[p.want][raiserPath(p.args)] = true
	}
	if t.Failed() {
		return
	}
	for _, e := range refusal.Entries() {
		if strings.HasPrefix(string(e.Code), "usage/") {
			if len(e.RaisedBy) != 0 {
				t.Errorf("%s: a usage code is raised by every command and lists none, got %v", e.Code, e.RaisedBy)
			}
			continue
		}
		var got []string
		for path := range seen[e.Code] {
			got = append(got, path)
		}
		slices.Sort(got)
		listed := slices.Sorted(slices.Values(e.RaisedBy))
		if !slices.Equal(got, listed) {
			t.Errorf("%s: RaisedBy lists %v, but the commands seen raising it are %v", e.Code, listed, got)
		}
	}
}
