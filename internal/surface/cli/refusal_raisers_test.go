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

// raiserProbe is one refused call and the code it must carry. The code is
// written out as the string a reader matches on, not as the registry's
// constant, so a path that switches to another code fails here, and so does
// a code whose name changes: the release guard sees only a code that goes
// missing from the ledger (adr-2610301638558260).
type raiserProbe struct {
	args []string
	want string
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
		{[]string{"capture", "the parser drops a line"}, "capture/missing-remedy"},
		{[]string{"capture", "the parser drops a line", "--remedy", "none (filed automatically)"}, "capture/remedy-machine-specific"},

		{[]string{"capture", "defer", unknownIssueID, "--after", "v0.1.0", "--reason", "the fix needs the parser rewrite"}, "capture/unknown-issue"},
		{[]string{"capture", "link", unknownIssueID, "--blocked-by", open}, "capture/unknown-issue"},
		{[]string{"capture", "promote", unknownIssueID}, "capture/unknown-issue"},
		{[]string{"capture", "remedy", unknownIssueID, "make the map before the first write"}, "capture/unknown-issue"},
		{[]string{"capture", "resolve", unknownIssueID, "fixed by the parser change", "--impact", "fix"}, "capture/unknown-issue"},
		{[]string{"capture", "wontfix", unknownIssueID, "the behaviour is the documented one"}, "capture/unknown-issue"},

		{[]string{"capture", "admit", unknownReadingItem, "--grounds", "the widened configuration is one the next release serves"}, "capture/unknown-reading-record"},
		{[]string{"capture", "disposition", unknownReadingItem, "--state", "accepted", "--grounds", probeGrounds}, "capture/unknown-reading-record"},
		{[]string{"capture", "promote", unknownReadingItem}, "capture/unknown-reading-record"},
		{[]string{"capture", "reframe", "--complete", unknownReframeID}, "capture/unknown-reading-record"},
		{[]string{"capture", "reframe", "--occasioned-by", unknownReadingItem, "--grounds", probeGrounds, "--open"}, "capture/unknown-reading-record"},
		{[]string{"capture", "surprise", "--occasioned-by", unknownReadingItem, probeText}, "capture/unknown-reading-record"},

		{[]string{"capture", "resolve", closed, "fixed by the writer change", "--impact", "fix"}, "capture/not-open"},
		{[]string{"capture", "wontfix", closed, "the behaviour is the documented one"}, "capture/not-open"},
		{[]string{"capture", "defer", closed, "--after", "v0.1.0", "--reason", "the fix needs the parser rewrite"}, "capture/not-open"},
		{[]string{"capture", "remedy", closed, "make the map before the first write"}, "capture/not-open"},

		{[]string{"capture", "resolve", open, "fixed by the parser change", "--impact", "fix", "--grounds", "bogus"}, "capture/insufficient-grounds"},
		{[]string{"capture", "wontfix", open, "the behaviour is the documented one", "--grounds", "bogus"}, "capture/insufficient-grounds"},
		{[]string{"capture", "promote", open, "--grounds", "bogus"}, "capture/insufficient-grounds"},
		{[]string{"capture", "admit", unknownReadingItem, "--grounds", "thin"}, "capture/insufficient-grounds"},
		{[]string{"capture", "surprise", "--occasioned-by", unknownReadingItem, "thin"}, "capture/insufficient-grounds"},
		{[]string{"capture", "reframe", "--occasioned-by", unknownReadingItem, "--grounds", "thin", "--open"}, "capture/insufficient-grounds"},

		{[]string{"capture", "the third thing breaks on empty input", "--remedy", cliTestRemedy, "--found-at", "../elsewhere"}, "capture/invalid-input"},
		{[]string{"capture", "mentions", "--ref", "zzz-no-such-ref"}, "capture/invalid-input"},
		{[]string{"capture", "resolve", open, "fixed by the parser change", "--impact", "fix", "--shipped-in", "not-a-tag"}, "capture/invalid-input"},
		{[]string{"capture", "link", open, "--blocked-by", open}, "capture/invalid-input"},
		{[]string{"capture", "defer", minor, "--after", "v0.1.0", "--reason", "the fix needs the parser rewrite"}, "capture/invalid-input"},
		{[]string{"capture", "remedy", open, ""}, "capture/invalid-input"},
		{[]string{"capture", "surprise", "--occasioned-by", open, probeText}, "capture/invalid-input"},
		{[]string{"capture", "reframe", "--occasioned-by", unknownReframeID, "--grounds", probeGrounds, "--open"}, "capture/invalid-input"},
		{[]string{"capture", "disposition", detectionItem, "--state", "bogus", "--grounds", probeGrounds}, "capture/invalid-input"},
		{[]string{"capture", "admit", detectionItem, "--grounds", "the widened configuration is one the next release serves"}, "capture/invalid-input"},

		{[]string{"capture", "disposition", admitItem, "--state", "accepted", "--grounds", probeGrounds}, "capture/not-characterized"},
		{[]string{"capture", "admit", admitItem, "--grounds", "the widened configuration is one the next release serves"}, "capture/not-characterized"},

		{[]string{"explain", "zzz-no-such-verb"}, "explain/unknown-path"},
		{[]string{"explain", "capture", "--input", "zzz"}, "explain/unrecognized-input"},
		{[]string{"explain", "zzz/no-such-code"}, "explain/unknown-code"},

		{[]string{"start"}, "start/outside-a-terminal"},
		{[]string{"start"}, "start/profile-unusable"},
		{[]string{"start"}, "start/repository-unready"},
		{[]string{"start"}, "start/host-not-runnable"},
		{[]string{"start"}, "start/not-logged-in"},
		{[]string{"start"}, "start/no-verdict"},
		{[]string{"start"}, "start/wall-not-held"},
	}

	seen := map[refusal.Code]map[string]bool{}
	for _, p := range probes {
		code, env, out := 0, errorEnvelope{}, ""
		if setup := startRefusalWorlds[p.want]; setup != nil {
			t.Run(p.want, func(t *testing.T) {
				setup(t)
				code, env, out = codedRefusal(t, p.args...)
			})
		} else {
			code, env, out = codedRefusal(t, p.args...)
		}
		if code != 2 || env.Code != p.want {
			t.Errorf("abcd %s: exit %d code %q, want exit 2 and %s:\n%s", strings.Join(p.args, " "), code, env.Code, p.want, out)
			continue
		}
		want := refusal.Code(p.want)
		if seen[want] == nil {
			seen[want] = map[string]bool{}
		}
		seen[want][raiserPath(p.args)] = true
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
