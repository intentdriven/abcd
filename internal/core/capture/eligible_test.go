package capture

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The drain's field-only eligibility (itd-82 decisions 4 and 7,
// spc-2609212015054359 scope 2, 6 and 9). Every test here reads the ledger and
// writes nothing: the dry-run is the whole of what this slice runs.

// drainFixture files one open record per call through Capture, so every record
// the tests judge is one the writer produced.
type drainFixture struct {
	t        *testing.T
	repo, ir string
}

func newDrainFixture(t *testing.T) drainFixture {
	t.Helper()
	repo, ir := ledger(t)
	return drainFixture{t: t, repo: repo, ir: ir}
}

func (f drainFixture) file(id string, sev Severity, cat Category, remedy string, blockedBy ...string) {
	f.t.Helper()
	if _, err := Capture(CaptureRequest{
		RepoRoot: f.repo, IssuesRoot: f.ir, Text: "text of " + id, Severity: sev,
		Category: cat, Source: "manual-test", Slug: "s", FoundDuring: "t", ForceID: id,
		Remedy: remedy, BlockedBy: blockedBy,
	}); err != nil {
		f.t.Fatalf("capture %s: %v", id, err)
	}
}

func (f drainFixture) plan() DrainPlan {
	f.t.Helper()
	p, err := PlanDrain(DrainPlanRequest{RepoRoot: f.repo, IssuesRoot: f.ir})
	if err != nil {
		f.t.Fatalf("PlanDrain: %v", err)
	}
	return p
}

func verdictOf(t *testing.T, p DrainPlan, id string) DrainVerdict {
	t.Helper()
	var found []DrainVerdict
	for _, v := range p.Dispositions {
		if v.ID == id {
			found = append(found, v)
		}
	}
	if len(found) != 1 {
		t.Fatalf("%s has %d dispositions, want exactly one: %+v", id, len(found), p.Dispositions)
	}
	return found[0]
}

// TestDrainRoutesAMixedLedgerByField is acceptance criterion 1 at the field
// layer: a minor bug with a remedy goes to a lane, one without is ineligible
// naming the field, the major, the security and the process issues are handed
// back naming the rule, the blocked one is skipped naming its blocker, and every
// open issue receives exactly one disposition.
func TestDrainRoutesAMixedLedgerByField(t *testing.T) {
	f := newDrainFixture(t)
	f.file("iss-1", SeverityMinor, "bug", "guard the nil map")
	f.file("iss-2", SeverityMinor, "bug", "")
	f.file("iss-3", SeverityMajor, "bug", "rewrite the parser")
	f.file("iss-4", SeverityMinor, "security", "tighten the check")
	f.file("iss-5", SeverityMinor, "process", "change the review rule")
	f.file("iss-6", SeverityNitpick, "documentation", "fix the typo", "iss-2")

	p := f.plan()
	if len(p.Dispositions) != 6 {
		t.Fatalf("want 6 dispositions (one per open issue), got %d: %+v", len(p.Dispositions), p.Dispositions)
	}
	cases := []struct {
		id      string
		outcome DrainOutcome
		rule    DrainRule
		reason  string
	}{
		{"iss-1", DrainEligible, RuleFields, "remedy"},
		{"iss-2", DrainIneligible, RuleRemedy, "remedy"},
		{"iss-3", DrainHandBack, RuleSeverity, "major"},
		{"iss-4", DrainHandBack, RuleSecurity, "security"},
		{"iss-5", DrainHandBack, RuleCategory, "process"},
		{"iss-6", DrainSkipped, RuleBlocked, "iss-2"},
	}
	for _, c := range cases {
		v := verdictOf(t, p, c.id)
		if v.Outcome != c.outcome || v.Rule != c.rule {
			t.Errorf("%s: got %s/%s, want %s/%s (%s)", c.id, v.Outcome, v.Rule, c.outcome, c.rule, v.Reason)
		}
		if !strings.Contains(v.Reason, c.reason) {
			t.Errorf("%s: the reason %q does not name %q", c.id, v.Reason, c.reason)
		}
	}
	if b := verdictOf(t, p, "iss-6").Blockers; len(b) != 1 || b[0] != "iss-2" {
		t.Errorf("the skipped issue names blockers %v, want [iss-2]", b)
	}
	if p.Counts[DrainEligible] != 1 || p.Counts[DrainHandBack] != 3 ||
		p.Counts[DrainIneligible] != 1 || p.Counts[DrainSkipped] != 1 {
		t.Errorf("counts = %v", p.Counts)
	}
}

// TestEligibilityIsTheFieldRuleAndNothingElse is the rule as a table over one
// record at a time: the fixable categories and the two small severities pass
// with a remedy, and every other category and severity is handed back whatever
// its remedy says.
func TestEligibilityIsTheFieldRuleAndNothingElse(t *testing.T) {
	for _, cat := range []Category{"tech-debt", "documentation", "inconsistency", "drift", "bug", "ux"} {
		for _, sev := range []Severity{SeverityNitpick, SeverityMinor} {
			v := Eligibility(Issue{ID: "iss-1", Category: cat, Severity: sev, Remedy: "r", Status: StateOpen})
			if v.Outcome != DrainEligible {
				t.Errorf("%s/%s with a remedy: %s (%s), want eligible", cat, sev, v.Outcome, v.Reason)
			}
		}
	}
	for _, cat := range []Category{"process", "observation", "architectural-insight", "future-work-seed", "lapse"} {
		v := Eligibility(Issue{ID: "iss-1", Category: cat, Severity: SeverityMinor, Remedy: "r", Status: StateOpen})
		if v.Outcome != DrainHandBack || v.Rule != RuleCategory {
			t.Errorf("%s: %s/%s, want handback/category", cat, v.Outcome, v.Rule)
		}
	}
	for _, sev := range []Severity{SeverityMajor, SeverityCritical} {
		v := Eligibility(Issue{ID: "iss-1", Category: "bug", Severity: sev, Remedy: "r", Status: StateOpen})
		if v.Outcome != DrainHandBack || v.Rule != RuleSeverity {
			t.Errorf("%s: %s/%s, want handback/severity", sev, v.Outcome, v.Rule)
		}
	}
	// A remedy of blanks is no remedy.
	if v := Eligibility(Issue{ID: "iss-1", Category: "bug", Severity: SeverityMinor, Remedy: "  \t", Status: StateOpen}); v.Outcome != DrainIneligible {
		t.Errorf("a blank remedy: %s, want ineligible", v.Outcome)
	}
	// A record that is not open is never a drain's to take.
	if v := Eligibility(Issue{ID: "iss-1", Category: "bug", Severity: SeverityMinor, Remedy: "r", Status: StateResolved}); v.Outcome == DrainEligible {
		t.Errorf("a resolved record was eligible")
	}
}

// TestDrainOrdersTheEligibleByCategoryThenSeverityThenAge is decision 7: the
// category order tech-debt, documentation, inconsistency, drift, bug, ux; then
// nitpick before minor; then oldest first. The summary states the rule.
func TestDrainOrdersTheEligibleByCategoryThenSeverityThenAge(t *testing.T) {
	f := newDrainFixture(t)
	f.file("iss-10", SeverityMinor, "ux", "r")
	f.file("iss-11", SeverityMinor, "bug", "r")
	f.file("iss-12", SeverityNitpick, "bug", "r")
	f.file("iss-13", SeverityMinor, "drift", "r")
	f.file("iss-14", SeverityMinor, "inconsistency", "r")
	f.file("iss-15", SeverityMinor, "documentation", "r")
	f.file("iss-16", SeverityMinor, "tech-debt", "r")
	f.file("iss-9", SeverityMinor, "tech-debt", "r")
	f.file("iss-8", SeverityMajor, "tech-debt", "r") // handed back: never in the order

	p := f.plan()
	var got []string
	for _, v := range p.Dispositions {
		if v.Outcome == DrainEligible {
			got = append(got, v.ID)
		}
	}
	want := []string{"iss-9", "iss-16", "iss-15", "iss-14", "iss-13", "iss-12", "iss-11", "iss-10"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("eligible order = %v\nwant           %v", got, want)
	}
	// The eligible come first, in that order, and the rest after them.
	if p.Dispositions[len(p.Dispositions)-1].ID != "iss-8" {
		t.Errorf("the handed-back record is not after the eligible ones: %+v", p.Dispositions)
	}
	for _, word := range []string{"tech-debt, documentation, inconsistency, drift, bug, ux", "nitpick before minor", "oldest first"} {
		if !strings.Contains(p.Order, word) {
			t.Errorf("the order rule %q does not state %q", p.Order, word)
		}
	}
}

// TestDrainPlanWritesNothing is the dry run's promise: the plan reads the ledger
// and leaves every byte of it as it was, and names the rule's decision record.
func TestDrainPlanWritesNothing(t *testing.T) {
	f := newDrainFixture(t)
	f.file("iss-1", SeverityMinor, "bug", "r")
	before := snapshotTree(t, f.repo)
	p := f.plan()
	if after := snapshotTree(t, f.repo); after != before {
		t.Fatalf("the plan changed the tree:\nbefore %s\nafter  %s", before, after)
	}
	if p.Record != EligibilityRecord || !strings.HasPrefix(p.Record, "adr-") {
		t.Errorf("the plan names the rule's record %q, want %q", p.Record, EligibilityRecord)
	}
}

// TestDrainPlanGivesAnUnreadableOpenRecordItsOwnDisposition keeps "every open
// issue receives exactly one disposition" true of a record the reader refuses:
// it is named, never silently absent.
func TestDrainPlanGivesAnUnreadableOpenRecordItsOwnDisposition(t *testing.T) {
	f := newDrainFixture(t)
	f.file("iss-1", SeverityMinor, "bug", "r")
	bad := filepath.Join(f.ir, "open", "iss-7-broken.md")
	if err := os.WriteFile(bad, []byte("no frontmatter here\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := f.plan()
	v := verdictOf(t, p, "iss-7")
	if v.Outcome != DrainUnreadable || v.Reason == "" {
		t.Fatalf("the unreadable record: %+v", v)
	}
}

// TestADrainRefusesToStartWithoutTheRuleRecord is criterion 11: with no
// decision record for the eligibility rule, the run refuses naming the record it
// needs; with one, it still refuses, because the issue-keyed lane it would hand
// each issue to is not built, and it says so rather than pretending to run.
func TestADrainRefusesToStartWithoutTheRuleRecord(t *testing.T) {
	err := drainStartCheck("")
	if !errors.Is(err, ErrDrainRuleUnrecorded) {
		t.Fatalf("no rule record: got %v, want ErrDrainRuleUnrecorded", err)
	}
	if !strings.Contains(err.Error(), "itd-82") || !strings.Contains(err.Error(), "decision record") {
		t.Errorf("the refusal does not name the record it needs: %v", err)
	}
	err = DrainStart()
	if !errors.Is(err, ErrDrainRunUnbuilt) {
		t.Fatalf("with the record: got %v, want ErrDrainRunUnbuilt", err)
	}
	if !strings.Contains(err.Error(), "--dry-run") {
		t.Errorf("the refusal does not name the dry run it can do: %v", err)
	}
}

// snapshotTree renders every file under root with its bytes, so any write shows.
func snapshotTree(t *testing.T, root string) string {
	t.Helper()
	var b strings.Builder
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		if info.IsDir() {
			b.WriteString("d " + rel + "\n")
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		b.WriteString("f " + rel + " " + string(data) + "\n")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return b.String()
}

// TestTheEligibilityRuleIsRecordedAndAccepted is the tree half of criterion 11
// and itd-82 decision 4: the record the binary names as the rule's is an
// accepted decision record in this repository's store, and the brief's
// invariants cite it. Deleting the record, leaving it proposed, or dropping the
// invariant fails here, before a drain could ship without them.
func TestTheEligibilityRuleIsRecordedAndAccepted(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("no go.mod above the package directory")
		}
		dir = parent
	}
	stamp := strings.TrimPrefix(EligibilityRecord, "adr-")
	matches, err := filepath.Glob(filepath.Join(dir, ".abcd", "development", "decisions", "adrs", stamp+"-*.md"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("the rule's record %s: want one file in the ADR store, got %v (%v)", EligibilityRecord, matches, err)
	}
	adr, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(adr), "\nid: "+EligibilityRecord+"\n") || !strings.Contains(string(adr), "\nstatus: accepted\n") {
		t.Errorf("%s is not an accepted record with that id", filepath.Base(matches[0]))
	}
	inv, err := os.ReadFile(filepath.Join(dir, ".abcd", "development", "brief", "02-constraints", "03-invariants.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(inv), "[adr-"+stamp+"]") {
		t.Errorf("the brief's invariants do not cite %s", EligibilityRecord)
	}
}
