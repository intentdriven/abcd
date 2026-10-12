package embargo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// The fixture advisories use repeated-letter ids of GitHub's documented shape
// (the GHSA-xxxx-xxxx-xxxx placeholder form), never a real advisory, and
// invented titles (examples-use-reserved-identifiers).

const listPath = "repos/example/example/security-advisories"

// advisoryJSON renders one advisory the way GitHub's list and get endpoints
// carry it; severity "" is null.
func advisoryJSON(ghsa, state, severity, summary, description string) map[string]any {
	a := map[string]any{
		"ghsa_id":     ghsa,
		"cve_id":      nil,
		"summary":     summary,
		"description": description,
		"severity":    nil,
		"state":       state,
		"html_url":    "https://github.com/example/example/security/advisories/" + ghsa,
	}
	if severity != "" {
		a["severity"] = severity
	}
	return a
}

func page(t *testing.T, advisories ...map[string]any) string {
	t.Helper()
	if advisories == nil {
		advisories = []map[string]any{}
	}
	b, err := json.Marshal(advisories)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// intakeFixture is a checkout with a github.com origin, a home apart from it,
// and a fake gh answering both list requests with no advisories.
func intakeFixture(t *testing.T) (fixture, *ghFake) {
	t.Helper()
	f := newFixture(t)
	f.repo.Git("remote", "add", "origin", "https://github.com/example/example.git")
	g := newGHFake(t)
	g.answer("GET", listPath, "triage", 0, "[]")
	g.answer("GET", listPath, "draft", 0, "[]")
	return f, g
}

var fixedNow = time.Date(2026, 10, 12, 9, 30, 0, 0, time.UTC)

func runIntake(t *testing.T, f fixture) IntakeResult {
	t.Helper()
	res, err := Intake(IntakeRequest{Home: f.home, RepoRoot: f.repo.Root(), Now: func() time.Time { return fixedNow }})
	if err != nil {
		t.Fatalf("Intake: %v", err)
	}
	return res
}

func storeRecord(t *testing.T, store, status, issID string) string {
	t.Helper()
	matches, _ := filepath.Glob(filepath.Join(store, ".abcd", "work", "issues", status, issID+"-*.md"))
	if len(matches) != 1 {
		t.Fatalf("%d records of %s in the store's %s/, want 1", len(matches), issID, status)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// TestSeverityMapping: GitHub's severities map onto the ledger's, and a null or
// unknown one is not mapped at all.
func TestSeverityMapping(t *testing.T) {
	for in, want := range map[string]string{
		"critical": "critical",
		"high":     "major",
		"medium":   "minor",
		"low":      "minor",
		"":         "",
		"moderate": "",
		"HIGH":     "",
	} {
		var p *string
		if in != "" {
			p = &in
		}
		got, ok := MapSeverity(p)
		if got != want || ok != (want != "") {
			t.Errorf("MapSeverity(%q) = %q, %v; want %q", in, got, ok, want)
		}
	}
}

// TestIntakeMintsPrivateRecordInStore: an advisory waiting on GitHub becomes a
// record in the store's own ledger, with the security-advisory source, the
// mapped severity and a neutral slug, and the state links the two.
func TestIntakeMintsPrivateRecordInStore(t *testing.T) {
	f, g := intakeFixture(t)
	g.answer("GET", listPath, "triage", 0, page(t,
		advisoryJSON("GHSA-xxxx-xxxx-xxxx", "triage", "high", "a sample report", "the sample body")))
	res := runIntake(t, f)
	if len(res.Minted) != 1 {
		t.Fatalf("minted %d, want 1: %+v", len(res.Minted), res)
	}
	m := res.Minted[0]
	if m.GHSAID != "GHSA-xxxx-xxxx-xxxx" || m.Severity != "major" {
		t.Fatalf("minted %+v", m)
	}
	rec := storeRecord(t, res.Store, "open", m.IssueID)
	for _, want := range []string{`source: "security-advisory"`, `severity: "major"`, `category: "security"`, "a sample report", "the sample body"} {
		if !strings.Contains(rec, want) {
			t.Errorf("the record does not carry %q:\n%s", want, rec)
		}
	}
	if strings.Contains(rec, "GHSA-") {
		t.Errorf("the record carries the advisory id; the link lives in the state:\n%s", rec)
	}
	matches, _ := filepath.Glob(filepath.Join(res.Store, ".abcd", "work", "issues", "open", m.IssueID+"-*.md"))
	if filepath.Base(matches[0]) != m.IssueID+"-private-report.md" {
		t.Errorf("record named %s, want the neutral slug", filepath.Base(matches[0]))
	}
	s := Store{Root: res.Store, Checkout: f.repo.Root()}
	st, err := s.Verify()
	if err != nil {
		t.Fatal(err)
	}
	if st.Repo != "example/example" || st.IntakeAt == nil || !st.IntakeAt.Equal(fixedNow) {
		t.Fatalf("state repo %q, intake_at %v", st.Repo, st.IntakeAt)
	}
	if len(st.Advisories) != 1 {
		t.Fatalf("state holds %d advisories", len(st.Advisories))
	}
	a := st.Advisories[0]
	if a.GHSAID != m.GHSAID || a.IssueID != m.IssueID || a.Title != "a sample report" || a.ForgeState != "triage" || a.Severity != "major" || a.Verdict != nil {
		t.Fatalf("state entry %+v", a)
	}
	// The list requests name the host, page through, and ask for both states.
	calls := g.calls()
	if len(calls) != 2 {
		t.Fatalf("%d gh calls, want the two list requests", len(calls))
	}
	for i, state := range []string{"triage", "draft"} {
		c := calls[i]
		for _, want := range []string{"--hostname", "github.com", "--paginate", listPath, "state=" + state, "per_page=100"} {
			if !c.has(want) {
				t.Errorf("call %d %q lacks %q", i+1, c.argv, want)
			}
		}
		if c.method() != "GET" {
			t.Errorf("call %d is a %s", i+1, c.method())
		}
	}
}

// TestIntakeLeavesPublicLedgerUntouched: intake writes nothing in the checkout
// itself: its status, ignored files included, and its index are the same
// after as before.
func TestIntakeLeavesPublicLedgerUntouched(t *testing.T) {
	f, g := intakeFixture(t)
	g.answer("GET", listPath, "draft", 0, page(t,
		advisoryJSON("GHSA-vvvv-vvvv-vvvv", "draft", "critical", "a sample report", "the sample body")))
	snapshot := func() string {
		return f.repo.Git("status", "--porcelain=v1", "--untracked-files=all", "--ignored") + "\n--\n" +
			f.repo.Git("ls-files", "--stage") + "\n--\n" + f.repo.Git("rev-parse", "HEAD")
	}
	before := snapshot()
	res := runIntake(t, f)
	if len(res.Minted) != 1 {
		t.Fatalf("minted %d", len(res.Minted))
	}
	if after := snapshot(); after != before {
		t.Fatalf("intake changed the checkout:\nbefore\n%s\nafter\n%s", before, after)
	}
	open, _ := os.ReadDir(filepath.Join(f.repo.Root(), ".abcd", "work", "issues", "open"))
	for _, e := range open {
		if e.Name() != ".gitkeep" {
			t.Fatalf("the public ledger holds %s", e.Name())
		}
	}
}

// TestIntakeIsIdempotent: a second run records nothing twice; it refreshes
// what GitHub now says of a held advisory and mints only the new one.
func TestIntakeIsIdempotent(t *testing.T) {
	f, g := intakeFixture(t)
	g.answer("GET", listPath, "triage", 1, page(t,
		advisoryJSON("GHSA-xxxx-xxxx-xxxx", "triage", "medium", "a sample report", "the sample body")))
	g.answer("GET", listPath, "triage", 2, "[]")
	g.answer("GET", listPath, "draft", 1, "[]")
	g.answer("GET", listPath, "draft", 2, page(t,
		advisoryJSON("GHSA-xxxx-xxxx-xxxx", "draft", "medium", "a sample report", "the sample body"),
		advisoryJSON("GHSA-wwww-wwww-wwww", "draft", "low", "another sample report", "another body")))
	first := runIntake(t, f)
	if len(first.Minted) != 1 {
		t.Fatalf("first run minted %d", len(first.Minted))
	}
	second := runIntake(t, f)
	if len(second.Minted) != 1 || second.Minted[0].GHSAID != "GHSA-wwww-wwww-wwww" {
		t.Fatalf("second run minted %+v, want only the new advisory", second.Minted)
	}
	if !slices.Equal(second.Held, []string{"GHSA-xxxx-xxxx-xxxx"}) {
		t.Fatalf("second run held %v", second.Held)
	}
	st, err := Store{Root: second.Store, Checkout: f.repo.Root()}.Verify()
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Advisories) != 2 {
		t.Fatalf("state holds %d advisories, want 2", len(st.Advisories))
	}
	if st.Advisories[0].ForgeState != "draft" {
		t.Fatalf("the held advisory's forge state is %q, want what GitHub now says", st.Advisories[0].ForgeState)
	}
	open, _ := filepath.Glob(filepath.Join(second.Store, ".abcd", "work", "issues", "open", "iss-*-private-report.md"))
	if len(open) != 2 {
		t.Fatalf("the store holds %d private records, want 2", len(open))
	}
}

// TestIntakeListsNullSeverityAndMintsNothing: an advisory GitHub gives no
// severity is listed for the person to set on GitHub; no record is minted for
// it with a guessed one, and the state does not hold it.
func TestIntakeListsNullSeverityAndMintsNothing(t *testing.T) {
	f, g := intakeFixture(t)
	g.answer("GET", listPath, "triage", 0, page(t,
		advisoryJSON("GHSA-rrrr-rrrr-rrrr", "triage", "", "a sample report", "the sample body"),
		advisoryJSON("GHSA-qqqq-qqqq-qqqq", "triage", "moderate", "another sample report", "another body")))
	res := runIntake(t, f)
	if len(res.Minted) != 0 {
		t.Fatalf("minted %+v for advisories with no usable severity", res.Minted)
	}
	if !slices.Equal(res.NeedsSeverity, []string{"GHSA-qqqq-qqqq-qqqq", "GHSA-rrrr-rrrr-rrrr"}) {
		t.Fatalf("needs_severity = %v", res.NeedsSeverity)
	}
	open, _ := filepath.Glob(filepath.Join(res.Store, ".abcd", "work", "issues", "open", "iss-*-private-report.md"))
	if len(open) != 0 {
		t.Fatalf("the store holds %d records", len(open))
	}
	st, err := Store{Root: res.Store, Checkout: f.repo.Root()}.Verify()
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Advisories) != 0 {
		t.Fatalf("state holds %d advisories", len(st.Advisories))
	}
}

// TestIntakePaginates: `gh api --paginate` writes one JSON array per page,
// back to back; every page's advisories are read.
func TestIntakePaginates(t *testing.T) {
	f, g := intakeFixture(t)
	g.answer("GET", listPath, "triage", 0,
		page(t, advisoryJSON("GHSA-mmmm-mmmm-mmmm", "triage", "low", "page one report", "body one"))+"\n"+
			page(t, advisoryJSON("GHSA-jjjj-jjjj-jjjj", "triage", "high", "page two report", "body two")))
	res := runIntake(t, f)
	var got []string
	for _, m := range res.Minted {
		got = append(got, m.GHSAID)
	}
	slices.Sort(got)
	if !slices.Equal(got, []string{"GHSA-jjjj-jjjj-jjjj", "GHSA-mmmm-mmmm-mmmm"}) {
		t.Fatalf("minted %v, want both pages", got)
	}
}

// TestIntakeRefusesMalformedListingWritingNothing: an advisory GitHub lists
// with an id or state this abcd does not recognise refuses the whole intake
// before any record is minted (unrecognized-input-never-writes).
func TestIntakeRefusesMalformedListingWritingNothing(t *testing.T) {
	f, g := intakeFixture(t)
	g.answer("GET", listPath, "triage", 0, page(t,
		advisoryJSON("GHSA-hhhh-hhhh-hhhh", "triage", "high", "a sample report", "the sample body"),
		advisoryJSON("GHSA-1111-1111-1111", "triage", "high", "another sample report", "another body")))
	_, err := Intake(IntakeRequest{Home: f.home, RepoRoot: f.repo.Root()})
	mustRefuse(t, err, "advisory id")
	p, _ := StorePath(f.home, f.repo.Root())
	if open, _ := filepath.Glob(filepath.Join(p, ".abcd", "work", "issues", "open", "iss-*")); len(open) != 0 {
		t.Fatalf("a refused intake minted %d records", len(open))
	}
}

// TestIntakeRefusesAnotherRepositorysStore: a store whose state names another
// repository (a fork sharing the root commit) is not mixed with this one's.
func TestIntakeRefusesAnotherRepositorysStore(t *testing.T) {
	f, _ := intakeFixture(t)
	s, _, err := Open(f.home, f.repo.Root())
	if err != nil {
		t.Fatal(err)
	}
	st := emptyState()
	st.Repo = "example/other"
	if err := s.SaveState(st); err != nil {
		t.Fatal(err)
	}
	_, err = Intake(IntakeRequest{Home: f.home, RepoRoot: f.repo.Root()})
	mustRefuse(t, err, "another repository")
}

// TestIntakeReportsMissingScope: a 403 from GitHub is reported with the scope
// the request needs and the command that grants it; abcd never runs gh auth.
func TestIntakeReportsMissingScope(t *testing.T) {
	f, g := intakeFixture(t)
	g.fail("GET", listPath, "triage", "HTTP 403: Resource not accessible by integration")
	_, err := Intake(IntakeRequest{Home: f.home, RepoRoot: f.repo.Root()})
	if err == nil || !strings.Contains(err.Error(), "gh auth refresh -s repository_advisories:") {
		t.Fatalf("err = %v, want the scope remedy", err)
	}
	for _, c := range g.calls() {
		if c.has("auth") {
			t.Fatalf("abcd ran gh auth: %q", c.argv)
		}
	}
}
