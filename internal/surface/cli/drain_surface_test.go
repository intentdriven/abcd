package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The front doors of the drain's field-only slice (itd-82,
// spc-2609212015054359): `capture --remedy` writes the field eligibility reads,
// `drain --dry-run` renders what a drain would do and writes nothing, and a
// bare `drain` refuses to start.

// captureWithRemedy files one issue through the CLI and returns its id.
func captureWithRemedy(t *testing.T, text string, flags ...string) string {
	t.Helper()
	out := runCLI(t, append([]string{"capture", text, "--json"}, flags...)...)
	var r struct {
		ID   string `json:"id"`
		Path string `json:"path"`
	}
	if err := json.Unmarshal(out, &r); err != nil {
		t.Fatalf("capture output not JSON: %v\n%s", err, out)
	}
	return r.ID
}

// TestCaptureRemedyFlagWritesTheField: `--remedy` reaches the record as the
// `remedy:` key the drain reads.
func TestCaptureRemedyFlagWritesTheField(t *testing.T) {
	repo := captureLedgerRepo(t)
	id := captureWithRemedy(t, "the help text spells colour two ways",
		"--category", "documentation", "--remedy", "use the British spelling throughout")
	matches, _ := filepath.Glob(filepath.Join(repo, ".abcd", "work", "issues", "open", id+"-*.md"))
	if len(matches) != 1 {
		t.Fatalf("no record for %s", id)
	}
	raw, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "\nremedy: use the British spelling throughout\n") &&
		!strings.Contains(string(raw), "\nremedy: \"use the British spelling throughout\"\n") {
		t.Fatalf("the record does not carry the remedy:\n%s", raw)
	}
}

// TestDrainDryRunRendersEveryDispositionAndWritesNothing: the dry run names
// each open issue once with its disposition and rule, states the order and the
// rule's record, in text and in --json, and leaves the ledger as it was.
func TestDrainDryRunRendersEveryDispositionAndWritesNothing(t *testing.T) {
	repo := captureLedgerRepo(t)
	eligible := captureWithRemedy(t, "a nil map is written before it is made", "--category", "bug", "--remedy", "make the map first")
	// A legacy record: filed before the remedy was required, so it carries
	// none. capture refuses such a filing now (ruling BX3), so the key is taken
	// out of a filed record.
	bare := captureWithRemedy(t, "a flaky timeout in the parser test", "--category", "bug", "--remedy", "raise the timeout")
	stripRemedyLine(t, repo, bare)
	major := captureWithRemedy(t, "the parser drops a whole record", "--category", "bug", "--severity", "major", "--remedy", "rewrite it")

	before := ledgerIssueCount(t, repo)
	text := string(runCLI(t, "drain", "--dry-run"))
	for _, want := range []string{eligible, bare, major, "eligible", "ineligible", "handback",
		"tech-debt, documentation, inconsistency, drift, bug, ux", "adr-", "writes nothing"} {
		if !strings.Contains(text, want) {
			t.Errorf("the dry-run text does not carry %q:\n%s", want, text)
		}
	}

	out := runCLI(t, "drain", "--dry-run", "--json")
	var plan struct {
		DryRun       bool   `json:"dry_run"`
		Record       string `json:"record"`
		Order        string `json:"order"`
		Dispositions []struct {
			ID      string `json:"id"`
			Outcome string `json:"outcome"`
			Rule    string `json:"rule"`
			Reason  string `json:"reason"`
		} `json:"dispositions"`
	}
	if err := json.Unmarshal(out, &plan); err != nil {
		t.Fatalf("--json is not JSON: %v\n%s", err, out)
	}
	if !plan.DryRun || plan.Record == "" || plan.Order == "" || len(plan.Dispositions) != 3 {
		t.Fatalf("--json plan = %+v", plan)
	}
	got := map[string]string{}
	for _, d := range plan.Dispositions {
		got[d.ID] = d.Outcome + "/" + d.Rule
	}
	want := map[string]string{eligible: "eligible/fields", bare: "ineligible/remedy", major: "handback/severity"}
	for id, w := range want {
		if got[id] != w {
			t.Errorf("%s: %s, want %s", id, got[id], w)
		}
	}
	if after := ledgerIssueCount(t, repo); after != before {
		t.Fatalf("the dry run changed the ledger: %d records before, %d after", before, after)
	}
}

// TestDrainWithoutDryRunRefusesToStart: the run itself is not built, so a bare
// `drain` refuses, says why, points at the dry run, and writes nothing.
func TestDrainWithoutDryRunRefusesToStart(t *testing.T) {
	repo := captureLedgerRepo(t)
	captureWithRemedy(t, "a nil map is written before it is made", "--category", "bug", "--remedy", "make the map first")
	before := ledgerIssueCount(t, repo)

	var stdout, stderr bytes.Buffer
	code := Run([]string{"drain"}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("a bare drain exited 0:\n%s%s", stdout.String(), stderr.String())
	}
	msg := stdout.String() + stderr.String()
	for _, want := range []string{"not built", "--dry-run"} {
		if !strings.Contains(msg, want) {
			t.Errorf("the refusal does not say %q:\n%s", want, msg)
		}
	}
	if after := ledgerIssueCount(t, repo); after != before {
		t.Fatalf("a refused drain changed the ledger")
	}
}

// stripRemedyLine takes the remedy: line out of one open record, leaving the
// legacy shape a record filed before the field was required has.
func stripRemedyLine(t *testing.T, repo, id string) {
	t.Helper()
	matches, _ := filepath.Glob(filepath.Join(repo, ".abcd", "work", "issues", "open", id+"-*.md"))
	if len(matches) != 1 {
		t.Fatalf("no open record for %s", id)
	}
	raw, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	var kept []string
	for _, ln := range strings.SplitAfter(string(raw), "\n") {
		if !strings.HasPrefix(ln, "remedy: ") {
			kept = append(kept, ln)
		}
	}
	if err := os.WriteFile(matches[0], []byte(strings.Join(kept, "")), 0o644); err != nil {
		t.Fatal(err)
	}
}
