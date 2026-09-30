package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/drainrule"
	"github.com/intentdriven/abcd/internal/core/issueschema"
)

// Every new issue carries a remedy (ruling BX3 of 2026-09-29): `capture`
// refuses a filing without `--remedy`, the machine value automatic filers
// write (ruling H12) is refused from a person, and `capture remedy` is how a
// person writes the fix onto an open record afterwards.

// TestCaptureWithoutARemedyIsRefused: exit 2, the flag and the machine value
// named, nothing captured.
func TestCaptureWithoutARemedyIsRefused(t *testing.T) {
	repo := captureLedgerRepo(t)
	for _, args := range [][]string{
		{"capture", "the parser drops a line"},
		{"capture", "the parser drops a line", "--remedy", "   "},
	} {
		// runCLISplit, not runCLIErr: the shared harness defaults a remedy.
		out, _, err := runCLISplit(t, args...)
		if code := exitCodeOf(err); code != 2 {
			t.Fatalf("%v: exit %d (err %v), want 2:\n%s", args, code, err, out)
		}
		for _, want := range []string{"--remedy", issueschema.MachineRemedy, "nothing captured"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("%v: the refusal does not name %q: %v", args, want, err)
			}
		}
	}
	if n := ledgerIssueCount(t, repo); n != 0 {
		t.Fatalf("a refused capture wrote %d record(s)", n)
	}
}

// TestCaptureRefusesTheMachineRemedyFromAPerson: the value means an automatic
// filer wrote the record, so a person typing it is refused, whatever its case.
func TestCaptureRefusesTheMachineRemedyFromAPerson(t *testing.T) {
	repo := captureLedgerRepo(t)
	out, _, err := runCLISplit(t, "capture", "the parser drops a line", "--remedy", " None (filed automatically) ")
	if code := exitCodeOf(err); code != 2 {
		t.Fatalf("exit %d (err %v), want 2:\n%s", code, err, out)
	}
	if !strings.Contains(err.Error(), "automatic filer") || !strings.Contains(err.Error(), "nothing captured") {
		t.Errorf("the refusal does not say why: %v", err)
	}
	if n := ledgerIssueCount(t, repo); n != 0 {
		t.Fatalf("a refused capture wrote %d record(s)", n)
	}
}

// TestCaptureRemedyVerbWritesTheRemedyAndTheDrainTakesIt: a record an
// automatic filer wrote is listed by the dry run as waiting on a person; `capture
// remedy` writes the person's fix, says what it replaced, and the dry run then
// lists the record as eligible.
func TestCaptureRemedyVerbWritesTheRemedyAndTheDrainTakesIt(t *testing.T) {
	repo := drainRuleRepo(t, drainrule.ProposalFrontmatter())
	res, err := capture.Capture(capture.CaptureRequest{
		RepoRoot: repo, Text: "a nil map is written before it is made", Severity: "minor",
		Category: "bug", Source: "agent-finding", FoundDuring: "t", Remedy: issueschema.MachineRemedy,
	})
	if err != nil {
		t.Fatal(err)
	}
	dry := string(runCLI(t, "drain", "--dry-run"))
	if !strings.Contains(dry, res.ID) || !strings.Contains(dry, "automatic filer") || !strings.Contains(dry, "capture remedy "+res.ID) {
		t.Fatalf("the dry run does not say the record waits on a person's remedy:\n%s", dry)
	}

	text := string(runCLI(t, "capture", "remedy", res.ID, "make the map", "before writing"))
	for _, want := range []string{res.ID, "make the map before writing", "replaced", issueschema.MachineRemedy} {
		if !strings.Contains(text, want) {
			t.Errorf("the verb's text does not carry %q:\n%s", want, text)
		}
	}
	raw, err := os.ReadFile(filepath.Join(repo, res.Path))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "remedy: make the map before writing\n") && !strings.Contains(string(raw), "remedy: \"make the map before writing\"\n") {
		t.Fatalf("the record does not carry the person's remedy:\n%s", raw)
	}
	var plan struct {
		Dispositions []struct {
			ID      string `json:"id"`
			Outcome string `json:"outcome"`
		} `json:"dispositions"`
	}
	if err := json.Unmarshal(runCLI(t, "drain", "--dry-run", "--json"), &plan); err != nil {
		t.Fatal(err)
	}
	if len(plan.Dispositions) != 1 || plan.Dispositions[0].Outcome != "eligible" {
		t.Fatalf("after the person's remedy the dry run gives %+v, want eligible", plan.Dispositions)
	}

	var got struct {
		ID       string `json:"id"`
		Remedy   string `json:"remedy"`
		Previous string `json:"previous"`
	}
	if err := json.Unmarshal(runCLI(t, "capture", "remedy", res.ID, "make the map first", "--json"), &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != res.ID || got.Remedy != "make the map first" || got.Previous != "make the map before writing" {
		t.Fatalf("--json = %+v", got)
	}
}

// TestCaptureRemedyVerbRefuses: an empty text, the machine value and a
// resolved record are refused with exit 2 and nothing written.
func TestCaptureRemedyVerbRefuses(t *testing.T) {
	repo := captureLedgerRepo(t)
	id := captureWithRemedy(t, "a nil map is written before it is made", "--category", "bug", "--remedy", "make the map first")
	done := captureWithRemedy(t, "the parser drops a whole record", "--category", "bug", "--remedy", "keep the record")
	runCLI(t, "capture", "resolve", done, "kept", "--impact", "fix",
		"--grounds", "pursued: the record is kept; shown wrong if it drops again")
	for _, args := range [][]string{
		{"capture", "remedy", id, "  "},
		{"capture", "remedy", id, issueschema.MachineRemedy},
		{"capture", "remedy", done, "fix it"},
	} {
		out, err := runCLIErr(t, args...)
		if code := exitCodeOf(err); code != 2 {
			t.Errorf("%v: exit %d (err %v), want 2:\n%s", args, code, err, out)
		}
	}
	matches, _ := filepath.Glob(filepath.Join(repo, ".abcd", "work", "issues", "open", id+"-*.md"))
	raw, err := os.ReadFile(matches[0])
	if err != nil || !strings.Contains(string(raw), "make the map first") {
		t.Fatalf("a refused remedy changed the record (err %v):\n%s", err, raw)
	}
}
