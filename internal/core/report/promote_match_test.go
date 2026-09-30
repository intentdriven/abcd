package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/record/match"
)

// promote_match_test.go covers the filing-time match on the promoted inbox
// report (itd-2609212137116617, iss-2609281911024185): a promoted report is
// matched against the ledger exactly as a capture is, on the report's own
// title and prose, never on the provenance lines every promoted report
// carries.

const (
	pmFinding = "The capture ledger reader silently skips a record whose frontmatter carries " +
		"a duplicated key, so the finding disappears from every listing without a warning."
	pmDoubleTitle = "Ledger reader skips records with a duplicated frontmatter key"
	pmDouble      = "Capture ledger reader silently skips any record whose frontmatter carries a " +
		"duplicated key: the finding disappears from every listing, and no warning is printed."
	pmFiller1 = "The site builder renders a stale anchor for a heading renamed since the last build."
	pmFiller2 = "The history store drops a transcript that exceeds its byte budget without saying so."
)

// reportWith is the filled template with its title and prose replaced.
func reportWith(t *testing.T, title, prose string) string {
	t.Helper()
	s := filled(t)
	s = strings.Replace(s, `title: "capture refuses a slug with a digit first"`, `title: "`+title+`"`, 1)
	s = strings.Replace(s, "Running capture with a slug of 9lives was refused.", prose, 1)
	return s
}

func plantOpen(t *testing.T, root, text string) string {
	t.Helper()
	res, err := capture.Capture(capture.CaptureRequest{
		RepoRoot: root, Text: text, Severity: capture.SeverityMinor, Category: "bug",
		Source: "user-observation", FoundDuring: "fixture", Remedy: issueschema.MachineRemedy,
	})
	if err != nil {
		t.Fatalf("plant: %v", err)
	}
	return res.ID
}

func bundledMatch() *match.Config { c := match.Bundled(); return &c }

// A promoted report that doubles an open record is written with capture's
// typed link naming it, and the promotion carries the match.
func TestPromoteLinksANearDuplicateOfAnOpenRecord(t *testing.T) {
	sandbox(t, time.Date(2026, 9, 30, 9, 0, 0, 0, time.UTC))
	ledger := abcdCheckout(t)
	held := plantOpen(t, ledger.Root(), pmFinding)
	plantOpen(t, ledger.Root(), pmFiller1)
	plantOpen(t, ledger.Root(), pmFiller2)

	f, err := File(mustParse(t, reportWith(t, pmDoubleTitle, pmDouble)), Sender{Key: strings.Repeat("d", 40), Name: "delta"})
	if err != nil {
		t.Fatalf("File: %v", err)
	}
	p, err := Promote(ledger.Root(), f.ID, bundledMatch())
	if err != nil {
		t.Fatalf("Promote: %v", err)
	}
	if p.Match == nil || len(p.Match.Matches) == 0 {
		t.Fatalf("no match reported: %+v", p.Match)
	}
	m := p.Match.Matches[0]
	if m.ID != held || !m.Linked || (m.Relation != match.Duplicates && m.Relation != match.Refines) {
		t.Fatalf("match = %+v, want %s linked", m, held)
	}
	body, err := os.ReadFile(filepath.Join(ledger.Root(), filepath.FromSlash(p.Path)))
	if err != nil {
		t.Fatal(err)
	}
	if want := "\n" + string(m.Relation) + ": [" + held + "]\n"; !strings.Contains(string(body), want) {
		t.Fatalf("the promoted capture carries no %q link:\n%s", strings.TrimSpace(want), body)
	}
}

// Two unrelated reports share only the provenance every promoted report
// carries (the sender's key, the inbox, the kind, the version, the surface,
// the evidence line). The second must not be matched to the first: the match
// reads the report's own title and prose.
func TestPromoteDoesNotMatchOnTheInboxBoilerplate(t *testing.T) {
	sandbox(t, time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC))
	ledger := abcdCheckout(t)
	sender := Sender{Key: strings.Repeat("e", 40), Name: "echo"}

	a, err := File(mustParse(t, reportWith(t, "Site anchors go stale", "Renamed headings keep their old anchor.")), sender)
	if err != nil {
		t.Fatalf("File a: %v", err)
	}
	pa, err := Promote(ledger.Root(), a.ID, bundledMatch())
	if err != nil {
		t.Fatalf("Promote a: %v", err)
	}
	setClock(t, time.Date(2026, 9, 30, 10, 1, 0, 0, time.UTC))
	b, err := File(mustParse(t, reportWith(t, "Transcripts over budget vanish", "History drops oversized transcripts quietly.")), sender)
	if err != nil {
		t.Fatalf("File b: %v", err)
	}
	pb, err := Promote(ledger.Root(), b.ID, bundledMatch())
	if err != nil {
		t.Fatalf("Promote b: %v", err)
	}
	if pb.Match == nil {
		t.Fatal("no match outcome: the promotion was not matched at all")
	}
	for _, m := range pb.Match.Matches {
		if m.ID == pa.Capture {
			t.Fatalf("%s matched %s on the inbox boilerplate alone: %+v", pb.Capture, pa.Capture, m)
		}
	}
	body, err := os.ReadFile(filepath.Join(ledger.Root(), filepath.FromSlash(pb.Path)))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), pa.Capture) {
		t.Fatalf("the second capture names the first:\n%s", body)
	}
}

// A promotion without a match configuration files unmatched, as before.
func TestPromoteWithoutAMatchReportsNone(t *testing.T) {
	sandbox(t, time.Date(2026, 9, 30, 11, 0, 0, 0, time.UTC))
	ledger := abcdCheckout(t)
	plantOpen(t, ledger.Root(), pmFinding)
	f, err := File(mustParse(t, reportWith(t, pmDoubleTitle, pmDouble)), Sender{Key: strings.Repeat("f", 40), Name: "foxtrot"})
	if err != nil {
		t.Fatalf("File: %v", err)
	}
	p, err := Promote(ledger.Root(), f.ID, nil)
	if err != nil {
		t.Fatalf("Promote: %v", err)
	}
	if p.Match != nil {
		t.Fatalf("match = %+v, want none", p.Match)
	}
}
