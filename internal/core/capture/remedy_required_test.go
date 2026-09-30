package capture

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/issueschema"
)

// Every new issue carries a remedy (ruling BX3 of 2026-09-29, itd-82 decision
// 6), and an automatic filer with no fix yet writes the one machine value
// (ruling H12), which a drain skips until a person writes a real remedy with
// SetRemedy. Records filed before the rule stay readable and valid.

// TestCaptureRefusesANewIssueWithoutARemedy: a blank or absent remedy is
// refused before anything is written, naming the field and the machine value.
func TestCaptureRefusesANewIssueWithoutARemedy(t *testing.T) {
	for _, remedy := range []string{"", "   \t"} {
		repo, ir := ledger(t)
		_, err := Capture(CaptureRequest{
			RepoRoot: repo, IssuesRoot: ir, Text: "the parser drops a line", Severity: SeverityMinor,
			Category: "bug", Source: "manual-test", Slug: "s", FoundDuring: "t", Remedy: remedy,
		})
		if !errors.Is(err, ErrRemedyRequired) || !errors.Is(err, ErrRequestRefused) {
			t.Fatalf("remedy %q: err = %v, want ErrRemedyRequired as a request refusal", remedy, err)
		}
		if !strings.Contains(err.Error(), "remedy") || !strings.Contains(err.Error(), issueschema.MachineRemedy) {
			t.Errorf("the refusal does not name the field and the machine value: %v", err)
		}
		if _, statErr := os.Stat(ir); !os.IsNotExist(statErr) {
			t.Errorf("a refused capture wrote the ledger directories (stat: %v)", statErr)
		}
	}
}

// TestTheMachineRemedyIsOneValue pins the spelling H12 rules and the one
// predicate every reader asks.
func TestTheMachineRemedyIsOneValue(t *testing.T) {
	if issueschema.MachineRemedy != "none (filed automatically)" {
		t.Fatalf("MachineRemedy = %q", issueschema.MachineRemedy)
	}
	for _, s := range []string{"none (filed automatically)", "  none (filed automatically)\t", "None (Filed Automatically)"} {
		if !issueschema.IsMachineRemedy(s) {
			t.Errorf("IsMachineRemedy(%q) = false", s)
		}
	}
	for _, s := range []string{"", "none", "none (filed automatically) then fix it", "guard the nil map"} {
		if issueschema.IsMachineRemedy(s) {
			t.Errorf("IsMachineRemedy(%q) = true", s)
		}
	}
}

// TestCaptureAcceptsTheMachineRemedy: the core files the machine value, as the
// automatic filers write it; the drain then lists the record as ineligible.
func TestCaptureAcceptsTheMachineRemedy(t *testing.T) {
	f := newDrainFixture(t)
	f.file("iss-1", SeverityMinor, "bug", issueschema.MachineRemedy)
	v := verdictOf(t, f.plan(), "iss-1")
	if v.Outcome != DrainIneligible || v.Rule != RuleRemedy {
		t.Fatalf("machine remedy: %s/%s (%s), want ineligible/remedy", v.Outcome, v.Rule, v.Reason)
	}
	for _, want := range []string{issueschema.MachineRemedy, "a person", "capture remedy"} {
		if !strings.Contains(v.Reason, want) {
			t.Errorf("the reason %q does not name %q", v.Reason, want)
		}
	}
}

// TestConsistencyFilesTheMachineRemedy: the consistency pass is an automatic
// filer, so its record carries the machine value.
func TestConsistencyFilesTheMachineRemedy(t *testing.T) {
	root := consistencyLedgerRepo(t)
	if _, err := IngestConsistency(root, consistencyPayload(t, root), "2026-09-26"); err != nil {
		t.Fatal(err)
	}
	list, err := List(ListRequest{RepoRoot: root, State: StateOpen})
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Issues) != 1 || list.Issues[0].Remedy != issueschema.MachineRemedy {
		t.Fatalf("filed = %+v; want one record carrying %q", list.Issues, issueschema.MachineRemedy)
	}
}

// TestSetRemedyWritesAndReplacesTheRemedy: the verb a person runs writes a
// remedy onto an open record, replaces the machine value, reports what it
// replaced, and leaves the drain able to take the record.
func TestSetRemedyWritesAndReplacesTheRemedy(t *testing.T) {
	f := newDrainFixture(t)
	f.file("iss-1", SeverityMinor, "bug", issueschema.MachineRemedy)
	res, err := SetRemedy(RemedyRequest{RepoRoot: f.repo, IssuesRoot: f.ir, ID: "iss-1", Remedy: "  make the map\n before writing  "})
	if err != nil {
		t.Fatalf("SetRemedy: %v", err)
	}
	if res.Remedy != "make the map before writing" || res.Previous != issueschema.MachineRemedy || res.Status != StateOpen {
		t.Fatalf("result = %+v", res)
	}
	if v := verdictOf(t, f.plan(), "iss-1"); v.Outcome != DrainEligible {
		t.Fatalf("after a person's remedy: %s (%s), want eligible", v.Outcome, v.Reason)
	}
	raw, err := os.ReadFile(filepath.Join(f.repo, res.Path))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(raw), "\nremedy: ") != 1 {
		t.Fatalf("want exactly one remedy: line:\n%s", raw)
	}
	// A legacy record with no remedy gains the key.
	legacy := writeLegacyRecord(t, f.ir, "iss-2")
	if _, err := SetRemedy(RemedyRequest{RepoRoot: f.repo, IssuesRoot: f.ir, ID: "iss-2", Remedy: "fix the typo"}); err != nil {
		t.Fatalf("SetRemedy on a legacy record: %v", err)
	}
	if raw, _ := os.ReadFile(legacy); !strings.Contains(string(raw), "\nremedy: ") {
		t.Fatalf("the legacy record gained no remedy:\n%s", raw)
	}
}

// TestSetRemedyRefusesWhatItCannotWrite: an empty text, the machine value, a
// malformed id, an unknown id and a record that is not open are refused with
// nothing written.
func TestSetRemedyRefusesWhatItCannotWrite(t *testing.T) {
	f := newDrainFixture(t)
	f.file("iss-1", SeverityMinor, "bug", "guard the nil map")
	f.file("iss-2", SeverityMinor, "bug", "guard the nil map")
	if _, err := Resolve(ResolveRequest{RepoRoot: f.repo, IssuesRoot: f.ir, ID: "iss-2", Resolution: "fixed",
		Impact: "fix", Grounds: "pursued: the map is made first; shown wrong if it panics again"}); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	before, err := os.ReadFile(filepath.Join(f.ir, "open", "iss-1-s.md"))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, id, remedy string
		is               error
	}{
		{"empty", "iss-1", " \n\t", ErrRequestRefused},
		{"machine value", "iss-1", " None (filed automatically) ", ErrRequestRefused},
		{"malformed id", "../iss-1", "fix it", ErrRequestRefused},
		{"unknown id", "iss-9", "fix it", ErrUnknownIssueID},
		{"resolved", "iss-2", "fix it", ErrTransitionConflict},
	}
	for _, c := range cases {
		_, err := SetRemedy(RemedyRequest{RepoRoot: f.repo, IssuesRoot: f.ir, ID: c.id, Remedy: c.remedy})
		if !errors.Is(err, c.is) {
			t.Errorf("%s: err = %v, want %v", c.name, err, c.is)
		}
	}
	after, err := os.ReadFile(filepath.Join(f.ir, "open", "iss-1-s.md"))
	if err != nil || string(after) != string(before) {
		t.Errorf("a refused SetRemedy changed the record (err %v)", err)
	}
}
