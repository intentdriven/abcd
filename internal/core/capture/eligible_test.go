package capture

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/drainrule"
	"github.com/intentdriven/abcd/internal/gittest"
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
	writeRuleRecord(t, repo, drainrule.ProposalFrontmatter())
	return drainFixture{t: t, repo: repo, ir: ir}
}

// strictRuleRecord is the id the fixtures' strict record carries.
const strictRuleRecord = "adr-2609300000000001"

// writeRuleRecord writes the repository's own drain eligibility record (ruling
// BX2): an accepted decision record carrying the drain fields given.
func writeRuleRecord(t *testing.T, repo, fields string) {
	t.Helper()
	dir := filepath.Join(repo, filepath.FromSlash(drainrule.ADRsRelDir))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nid: " + strictRuleRecord + "\nslug: drain-rule\nstatus: accepted\ndate: 2026-09-30\n" + fields + "---\n\n# ADR\n"
	if err := os.WriteFile(filepath.Join(dir, "2609300000000001-drain-rule.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// loosenedFields is a project rule that loosens both floors H11 names.
const loosenedFields = "drain_categories: [tech-debt, documentation, inconsistency, drift, bug, ux]\n" +
	"drain_severities: [nitpick, minor, major]\ndrain_security: take\ndrain_remedy: required\n"

// file captures one open record. An empty remedy files a LEGACY record:
// capture refuses a new issue without a remedy (ruling BX3), so the record is
// filed with one and the key is then taken out, which is the shape every
// record filed before the rule has.
func (f drainFixture) file(id string, sev Severity, cat Category, remedy string, blockedBy ...string) {
	f.t.Helper()
	legacy := remedy == ""
	if legacy {
		remedy = "a remedy the legacy shape then loses"
	}
	res, err := Capture(CaptureRequest{
		RepoRoot: f.repo, IssuesRoot: f.ir, Text: "text of " + id, Severity: sev,
		Category: cat, Source: "manual-test", Slug: "s", FoundDuring: "t", ForceID: id,
		Remedy: remedy, BlockedBy: blockedBy,
	})
	if err != nil {
		f.t.Fatalf("capture %s: %v", id, err)
	}
	if legacy {
		stripRemedy(f.t, filepath.Join(f.repo, res.Path))
	}
}

// stripRemedy takes the remedy: line out of a record, leaving the legacy shape
// a record filed before the field was required has.
func stripRemedy(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var kept []string
	for _, ln := range strings.SplitAfter(string(raw), "\n") {
		if !strings.HasPrefix(ln, "remedy: ") {
			kept = append(kept, ln)
		}
	}
	if err := os.WriteFile(path, []byte(strings.Join(kept, "")), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeLegacyRecord writes an open record in the shape the ledger held before
// the remedy was required: every required key and no remedy.
func writeLegacyRecord(t *testing.T, issuesRoot, id string) string {
	t.Helper()
	dir := filepath.Join(issuesRoot, "open")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, id+"-s.md")
	body := "---\nschema_version: 1\nid: " + id + "\nslug: s\nseverity: minor\ncategory: bug\nsource: manual-test\nfound_during: t\n---\n\nbody\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
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
			v := eligibility(Issue{ID: "iss-1", Category: cat, Severity: sev, Remedy: "r", Status: StateOpen}, drainrule.Baseline(), deferralAnchor{})
			if v.Outcome != DrainEligible {
				t.Errorf("%s/%s with a remedy: %s (%s), want eligible", cat, sev, v.Outcome, v.Reason)
			}
		}
	}
	for _, cat := range []Category{"process", "observation", "architectural-insight", "future-work-seed", "lapse"} {
		v := eligibility(Issue{ID: "iss-1", Category: cat, Severity: SeverityMinor, Remedy: "r", Status: StateOpen}, drainrule.Baseline(), deferralAnchor{})
		if v.Outcome != DrainHandBack || v.Rule != RuleCategory {
			t.Errorf("%s: %s/%s, want handback/category", cat, v.Outcome, v.Rule)
		}
	}
	for _, sev := range []Severity{SeverityMajor, SeverityCritical} {
		v := eligibility(Issue{ID: "iss-1", Category: "bug", Severity: sev, Remedy: "r", Status: StateOpen}, drainrule.Baseline(), deferralAnchor{})
		if v.Outcome != DrainHandBack || v.Rule != RuleSeverity {
			t.Errorf("%s: %s/%s, want handback/severity", sev, v.Outcome, v.Rule)
		}
	}
	// A remedy of blanks is no remedy.
	if v := eligibility(Issue{ID: "iss-1", Category: "bug", Severity: SeverityMinor, Remedy: "  \t", Status: StateOpen}, drainrule.Baseline(), deferralAnchor{}); v.Outcome != DrainIneligible {
		t.Errorf("a blank remedy: %s, want ineligible", v.Outcome)
	}
	// A record that is not open is never a drain's to take.
	if v := eligibility(Issue{ID: "iss-1", Category: "bug", Severity: SeverityMinor, Remedy: "r", Status: StateResolved}, drainrule.Baseline(), deferralAnchor{}); v.Outcome == DrainEligible {
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
	if p.Record != strictRuleRecord {
		t.Errorf("the plan names the rule's record %q, want the repository's own %q", p.Record, strictRuleRecord)
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

// TestADrainRefusesARepositoryWithoutItsOwnRule is criterion 11 under ruling
// BX2: the repository must hold the eligibility decision in its own record, and
// until it does both the dry run and the start refuse, naming how to add it.
// With the record, the start still refuses, because the issue-keyed lane it
// would hand each issue to is not built, and it says so rather than pretending
// to run.
func TestADrainRefusesARepositoryWithoutItsOwnRule(t *testing.T) {
	repo, ir := ledger(t)
	f := drainFixture{t: t, repo: repo, ir: ir}
	f.file("iss-1", SeverityMinor, "bug", "r")
	before := snapshotTree(t, repo)
	if p, err := PlanDrain(DrainPlanRequest{RepoRoot: repo, IssuesRoot: ir}); !errors.Is(err, ErrDrainRuleUnrecorded) {
		t.Fatalf("a dry run without the repository's rule: plan %+v, err %v; want ErrDrainRuleUnrecorded", p, err)
	} else if !strings.Contains(err.Error(), "ahoy install") || !strings.Contains(err.Error(), drainrule.FieldCategories) {
		t.Errorf("the refusal does not name how to add the record: %v", err)
	}
	err := DrainStart(repo)
	if !errors.Is(err, ErrDrainRuleUnrecorded) || !strings.Contains(err.Error(), "ahoy install") {
		t.Fatalf("a start without the repository's rule: got %v, want ErrDrainRuleUnrecorded naming ahoy install", err)
	}
	if after := snapshotTree(t, repo); after != before {
		t.Fatalf("a refused drain wrote:\nbefore %s\nafter  %s", before, after)
	}

	writeRuleRecord(t, repo, drainrule.ProposalFrontmatter())
	err = DrainStart(repo)
	if !errors.Is(err, ErrDrainRunUnbuilt) {
		t.Fatalf("with the record: got %v, want ErrDrainRunUnbuilt", err)
	}
	if !strings.Contains(err.Error(), "--dry-run") || !strings.Contains(err.Error(), strictRuleRecord) {
		t.Errorf("the refusal does not name the dry run and the rule's record: %v", err)
	}
	if strings.Contains(err.Error(), "loosen") {
		t.Errorf("the strict rule's start names a loosened floor: %v", err)
	}
}

// TestADrainAppliesTheRepositorysOwnRule: the rule is the record's, not the
// binary's. A narrowed record hands back what the baseline would take.
func TestADrainAppliesTheRepositorysOwnRule(t *testing.T) {
	repo, ir := ledger(t)
	writeRuleRecord(t, repo, "drain_categories: [documentation]\ndrain_severities: [nitpick]\n"+
		"drain_security: handback\ndrain_remedy: required\n")
	f := drainFixture{t: t, repo: repo, ir: ir}
	f.file("iss-1", SeverityNitpick, "documentation", "fix the typo")
	f.file("iss-2", SeverityMinor, "documentation", "fix the page")
	f.file("iss-3", SeverityNitpick, "bug", "guard the nil map")
	p := f.plan()
	for id, want := range map[string]string{
		"iss-1": "eligible/fields", "iss-2": "handback/severity", "iss-3": "handback/category",
	} {
		v := verdictOf(t, p, id)
		if got := string(v.Outcome) + "/" + string(v.Rule); got != want {
			t.Errorf("%s: %s (%s), want %s", id, got, v.Reason, want)
		}
	}
	if !strings.Contains(verdictOf(t, p, "iss-2").Reason, "(nitpick)") {
		t.Errorf("the severity hand-back does not name the repository's severities: %s", verdictOf(t, p, "iss-2").Reason)
	}
	if len(p.Loosened) != 0 || p.Loosened == nil {
		t.Errorf("a narrowed rule: loosened = %#v, want an empty list", p.Loosened)
	}
}

// TestALoosenedRuleIsLoud is ruling H11: a project's record may let a drain
// take major issues and security issues, and every floor it loosens is named
// in the dry run's plan and in the start's refusal.
func TestALoosenedRuleIsLoud(t *testing.T) {
	repo, ir := ledger(t)
	writeRuleRecord(t, repo, loosenedFields)
	f := drainFixture{t: t, repo: repo, ir: ir}
	f.file("iss-1", SeverityMajor, "bug", "rewrite the parser")
	f.file("iss-2", SeverityMinor, "security", "tighten the check")
	f.file("iss-3", SeverityCritical, "bug", "stop the data loss")
	p := f.plan()
	if strings.Join(p.Loosened, ", ") != "severity major, security" {
		t.Errorf("the plan names loosened floors %v, want [severity major security]", p.Loosened)
	}
	for id, want := range map[string]string{
		"iss-1": "eligible/fields", "iss-2": "eligible/fields", "iss-3": "handback/severity",
	} {
		v := verdictOf(t, p, id)
		if got := string(v.Outcome) + "/" + string(v.Rule); got != want {
			t.Errorf("%s: %s (%s), want %s", id, got, v.Reason, want)
		}
	}
	// Security is taken last, after every fixable category.
	if !strings.Contains(p.Order, "ux, security") || !strings.Contains(p.Order, "nitpick before minor before major") {
		t.Errorf("the order does not state the loosened rule: %s", p.Order)
	}
	err := DrainStart(repo)
	for _, want := range []string{"loosens", "severity major", "security"} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("the start's refusal does not name %q: %v", want, err)
		}
	}
}

// TestAMalformedRuleRefusesTheDrain: a partial record refuses the dry run and
// the start alike, never falling back to either the baseline or a looser rule.
func TestAMalformedRuleRefusesTheDrain(t *testing.T) {
	repo, ir := ledger(t)
	writeRuleRecord(t, repo, "drain_categories: [bug]\ndrain_severities: [nitpick, minor, major]\ndrain_security: take\n")
	f := drainFixture{t: t, repo: repo, ir: ir}
	f.file("iss-1", SeverityMinor, "bug", "r")
	if p, err := PlanDrain(DrainPlanRequest{RepoRoot: repo, IssuesRoot: ir}); !errors.Is(err, drainrule.ErrMalformed) {
		t.Fatalf("a partial record: plan %+v, err %v; want ErrMalformed", p, err)
	}
	if err := DrainStart(repo); !errors.Is(err, drainrule.ErrMalformed) || !strings.Contains(err.Error(), drainrule.FieldRemedy) {
		t.Fatalf("a partial record's start: %v; want ErrMalformed naming %s", err, drainrule.FieldRemedy)
	}
}

// TestADrainHandsBackARecordWaitingOnAPerson is the gap the remedy lanes found:
// a record whose remedy opens "Waits on" names a fix that waits on a person's
// ruling, and a record whose deferral past the current anchor tag is live was
// carried past this release by a person. Both are handed back, each naming its
// rule, whatever the project's rule would otherwise let through. A deferral
// past an earlier tag has lapsed and holds nothing back.
func TestADrainHandsBackARecordWaitingOnAPerson(t *testing.T) {
	r := gittest.NewRepo(t)
	r.Commit("root")
	r.Git("tag", "v0.1.0")
	r.Commit("next")
	r.Git("tag", "v0.2.0")
	repo := r.Root()
	ir := filepath.Join(repo, LedgerRelPath)
	writeRuleRecord(t, repo, loosenedFields)
	f := drainFixture{t: t, repo: repo, ir: ir}
	f.file("iss-1", SeverityMinor, "bug", "Waits on ruling G (keep or drop the flag): if dropped, delete it")
	f.file("iss-2", SeverityMajor, "bug", "rewrite the parser")
	f.file("iss-3", SeverityMajor, "bug", "rewrite the lexer")
	f.file("iss-4", SeverityMinor, "bug", "waits on the facilitator's round trip: then rerun it")
	f.file("iss-5", SeverityMinor, "bug", "guard the nil map; it waits on nothing")
	setDeferral(t, ir, "iss-2", "v0.2.0")
	setDeferral(t, ir, "iss-3", "v0.1.0")

	p := f.plan()
	if p.Anchor != "v0.2.0" {
		t.Errorf("the plan's anchor = %q, want v0.2.0", p.Anchor)
	}
	cases := map[string]struct{ want, reason string }{
		"iss-1": {"handback/waits-on-ruling", "Waits on"},
		"iss-2": {"handback/deferred", "v0.2.0"},
		"iss-3": {"eligible/fields", ""},
		"iss-4": {"handback/waits-on-ruling", "Waits on"},
		"iss-5": {"eligible/fields", ""},
	}
	for id, c := range cases {
		v := verdictOf(t, p, id)
		if got := string(v.Outcome) + "/" + string(v.Rule); got != c.want {
			t.Errorf("%s: %s (%s), want %s", id, got, v.Reason, c.want)
		}
		if !strings.Contains(v.Reason, c.reason) {
			t.Errorf("%s: the reason %q does not name %q", id, v.Reason, c.reason)
		}
	}
}

// setDeferral writes the waiver pair onto an open record by hand, the shape
// `capture defer` writes.
func setDeferral(t *testing.T, ir, id, after string) {
	t.Helper()
	matches, _ := filepath.Glob(filepath.Join(ir, "open", id+"-*.md"))
	if len(matches) != 1 {
		t.Fatalf("no open record for %s", id)
	}
	raw, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	s := strings.Replace(string(raw), "\nslug: ", "\ndeferred_after: \""+after+"\"\ndeferral_reason: a person's reason\nslug: ", 1)
	if err := os.WriteFile(matches[0], []byte(s), 0o644); err != nil {
		t.Fatal(err)
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

// abcdsOwnRuleRecord is the record abcd's own repository states its drain rule
// in, cited by the brief's invariants.
const abcdsOwnRuleRecord = "adr-2609291342092738"

// TestAbcdsOwnDrainRuleIsTheStrictBaseline is the tree half of criterion 11,
// itd-82 decision 4 and ruling H11's note: this repository holds its own drain
// rule in an accepted record, that record states abcd's strict baseline and
// loosens nothing, and the brief's invariants cite it. Loosening the record,
// deleting it, leaving it proposed, or dropping the invariant fails here.
func TestAbcdsOwnDrainRuleIsTheStrictBaseline(t *testing.T) {
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
	r, err := drainrule.Load(dir)
	if err != nil {
		t.Fatalf("abcd's own repository holds no readable drain rule: %v", err)
	}
	if r.Record != abcdsOwnRuleRecord {
		t.Errorf("abcd's drain rule is stated in %s, want %s", r.Record, abcdsOwnRuleRecord)
	}
	b := drainrule.Baseline()
	if len(r.Loosened) != 0 || strings.Join(r.Categories, ",") != strings.Join(b.Categories, ",") ||
		strings.Join(r.Severities, ",") != strings.Join(b.Severities, ",") || r.Security != b.Security {
		t.Errorf("abcd's own drain rule is not the strict baseline: %+v", r)
	}
	inv, err := os.ReadFile(filepath.Join(dir, ".abcd", "development", "brief", "02-constraints", "03-invariants.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(inv), "["+abcdsOwnRuleRecord+"]") {
		t.Errorf("the brief's invariants do not cite %s", abcdsOwnRuleRecord)
	}
}

// TestADeferralIsHandedBackWhenTheCheckoutHoldsNoReleaseTag: a shallow clone
// (fetch-depth 1, the unattended drain's likely checkout) fetches no tags, so
// whether a deferral is live cannot be known. Not knowing must not let the
// record through: every record carrying a deferral is handed back, naming the
// missing tags and how to fetch them, and a record without one is untouched.
func TestADeferralIsHandedBackWhenTheCheckoutHoldsNoReleaseTag(t *testing.T) {
	src := gittest.NewRepo(t)
	src.Commit("root")
	src.Git("tag", "v0.1.0")
	repo := src.Root()
	ir := filepath.Join(repo, LedgerRelPath)
	writeRuleRecord(t, repo, loosenedFields)
	f := drainFixture{t: t, repo: repo, ir: ir}
	f.file("iss-2", SeverityMajor, "bug", "rewrite the parser")
	f.file("iss-3", SeverityMajor, "bug", "rewrite the lexer")
	f.file("iss-5", SeverityMinor, "bug", "guard the nil map")
	setDeferral(t, ir, "iss-2", "v0.1.0")
	setDeferral(t, ir, "iss-3", "v0.0.9")
	src.Commit("ledger")
	src.Git("tag", "v0.2.0")

	clone := filepath.Join(t.TempDir(), "clone")
	src.Git("clone", "--quiet", "--depth", "1", "--no-tags", "file://"+repo, clone)
	shallow := drainFixture{t: t, repo: clone, ir: filepath.Join(clone, LedgerRelPath)}
	p := shallow.plan()
	if p.Anchor != "" {
		t.Errorf("a tagless clone reports anchor %q", p.Anchor)
	}
	cases := map[string]string{
		"iss-2": "handback/deferred",
		"iss-3": "handback/deferred",
		"iss-5": "eligible/fields",
	}
	for id, want := range cases {
		v := verdictOf(t, p, id)
		if got := string(v.Outcome) + "/" + string(v.Rule); got != want {
			t.Errorf("%s: %s (%s), want %s", id, got, v.Reason, want)
		}
		if want == "handback/deferred" {
			for _, w := range []string{"anchor unknown", "no release tag", "git fetch --tags"} {
				if !strings.Contains(v.Reason, w) {
					t.Errorf("%s: the reason %q does not name %q", id, v.Reason, w)
				}
			}
		}
	}
	if !p.AnchorUnknown {
		t.Error("the plan does not say its anchor is unknown")
	}
}

// TestWaitsOnIsReadAsAWordNotAPrefix: "Waits on" followed by a colon, or
// ending the remedy, opens a remedy that waits on a ruling as surely as one
// followed by a space; a word that merely starts with it does not.
func TestWaitsOnIsReadAsAWordNotAPrefix(t *testing.T) {
	for remedy, want := range map[string]bool{
		"Waits on: ruling H4.":            true,
		"WAITS ON:H4":                     true,
		"waits on":                        true,
		"  Waits on\tH4: then do it":      true,
		"Waits on ruling G: drop it":      true,
		"Waits onward: nothing":           false,
		"guard the map; it waits on none": false,
	} {
		if got := waitsOnRuling(remedy); got != want {
			t.Errorf("waitsOnRuling(%q) = %v, want %v", remedy, got, want)
		}
	}
}
