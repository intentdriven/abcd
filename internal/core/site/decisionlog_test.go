package site

// The decision log page's tests (spc-2610110931381316, step 5): every decision
// record on one page, newest first and grouped by month, published only where
// the repository opts in, and kept off the pages built for the stores that move.

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"
)

// fixtureDecision is one decision record the fixture holds.
type fixtureDecision struct {
	ID, Date, Text string
}

// decisionCount is more than one development deck holds, so a page that capped
// its list the way the development page caps a bucket would lose entries.
const decisionCount = developmentDeckCap + 5

// addDecisions gives the fixture decisionCount decision records over three
// months, declares their store to record-lint, opts the site into publishing
// them when optIn is set, and commits. The records are returned oldest first.
// The first one's body runs onto a second line, so a page that showed only a
// record's first line would be caught.
func (f *fixture) addDecisions(optIn bool) []fixtureDecision {
	f.t.Helper()
	lintCfg := outFile(f.t, f.root, ".abcd/record-lint.json")
	f.write(".abcd/record-lint.json", strings.Replace(lintCfg,
		`"prn": ".abcd/development/principles"`,
		`"prn": ".abcd/development/principles",
      "dcn": ".abcd/work/decisions"`, 1))
	if optIn {
		manifest := outFile(f.t, f.root, ".abcd/site.json")
		f.write(".abcd/site.json", strings.Replace(manifest,
			`"record": {"issue_ledger": true}`, `"record": {"issue_ledger": true, "decisions": true}`, 1))
	}
	var out []fixtureDecision
	for i := 0; i < decisionCount; i++ {
		month, day := 8+i/20, 1+i%20
		date := fmt.Sprintf("2026-%02d-%02d", month, day)
		id := fmt.Sprintf("dcn-26%02d%02d0000000001", month, day)
		slug := fmt.Sprintf("decision-%02d", i)
		text := fmt.Sprintf("%s — Decision %02d holds its whole sentence, tail-%02d.", date, i, i)
		if i == 0 {
			text += "\nIts second line runs on, continued-00."
		}
		f.write(".abcd/work/decisions/"+id+"-"+slug+".md",
			"---\nid: "+id+"\nslug: "+slug+"\ndate: "+date+"\n---\n"+text+"\n")
		out = append(out, fixtureDecision{ID: id, Date: date, Text: text})
	}
	f.commitAt("2026-03-07T09:00:00+00:00", "docs: record the decisions", "None")
	return out
}

// TestDecisionLogPageListsEveryEntry is A2 at the site: one page lists every
// decision, uncapped, newest first, grouped by month, each row its id linking
// the record's own page, its date and its full text — and the build that writes
// it is the same build twice.
func TestDecisionLogPageListsEveryEntry(t *testing.T) {
	f := newFixture(t)
	decs := f.addDecisions(true)
	out := t.TempDir()
	res := buildFixture(t, f, out)

	if !containsString(res.Files, "record/decision-log/index.html") {
		t.Fatalf("the build wrote no decision log page: %v", res.Files)
	}
	page := outFile(t, out, "record/decision-log/index.html")
	if !strings.Contains(page, `<h1 class="pagetitle">Decision log</h1>`) {
		t.Error(`the page is not headed "Decision log"`)
	}

	// decs is oldest first; the page runs newest first, so each row sits
	// above the one for the decision recorded before it.
	last := len(page)
	for _, d := range decs {
		link := `<a href="/record/decision/` + d.ID + `/">` + d.ID + `</a>`
		at := strings.Index(page, link)
		if at < 0 {
			t.Errorf("the page has no row linking %s to its own page", d.ID)
			continue
		}
		if at > last {
			t.Errorf("%s is listed below an older decision; the page must run newest first", d.ID)
		}
		last = at
		if !containsString(res.Files, "record/decision/"+d.ID+"/index.html") {
			t.Errorf("%s has no page of its own", d.ID)
		}
		row := page[at:]
		if end := strings.Index(row, "</li>"); end >= 0 {
			row = row[:end]
		}
		if !strings.Contains(row, `<span class="d">`+d.Date+`</span>`) {
			t.Errorf("the row for %s does not carry its date", d.ID)
		}
		for _, line := range strings.Split(d.Text, "\n") {
			if !strings.Contains(row, line) {
				t.Errorf("the row for %s does not carry its full text: missing %q", d.ID, line)
			}
		}
	}

	// One disclosure per month, every entry in the HTML whether its month is
	// open or not, and the months in order.
	months := regexp.MustCompile(`<details class="panel fold[^"]*"[^>]*><summary><h3>(\d{4}-\d{2})<span>(\d+)</span>`).
		FindAllStringSubmatch(page, -1)
	got := []string{}
	for _, m := range months {
		got = append(got, m[1]+"="+m[2])
	}
	if want := []string{"2026-10=13", "2026-09=20", "2026-08=20"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("the months are %v, want %v", got, want)
	}

	// The navigation reaches the page, and the dashboard has one tile with the
	// count that leads to it.
	dash := outFile(t, out, "record/index.html")
	if !strings.Contains(dash, `<a href="/record/decision-log/">Decision log</a>`) {
		t.Error("the navigation does not reach the decision log")
	}
	tile := fmt.Sprintf(`<a class="tile lead" href="/record/decision-log/"><span class="n">%d</span><span class="l">Decision log</span>`, decisionCount)
	if strings.Count(dash, tile) != 1 {
		t.Errorf("the dashboard has no single tile counting %d decisions and linking the page", decisionCount)
	}
	// The ADR tile keeps its own label.
	if !strings.Contains(dash, `<span class="l">decisions</span>`) {
		t.Error(`the ADR tile lost its "decisions" label`)
	}

	// The export built twice is identical.
	again := t.TempDir()
	res2 := buildFixture(t, f, again)
	if strings.Join(res.Files, "\n") != strings.Join(res2.Files, "\n") {
		t.Fatal("two builds of one tree wrote different files")
	}
	for _, name := range res.Files {
		if outFile(t, out, name) != outFile(t, again, name) {
			t.Errorf("%s differs between two builds of the same tree", name)
		}
	}
}

// TestDecisionsAreOptedInto is adr-32's tiering at the site: the decisions folder
// is working-tier data, so without the switch no decision reaches the export,
// no page is written for one, and nothing links the log.
func TestDecisionsAreOptedInto(t *testing.T) {
	f := newFixture(t)
	f.addDecisions(false)
	out := t.TempDir()
	res := buildFixture(t, f, out)
	for _, file := range res.Files {
		if strings.Contains(file, "record/decision/") || strings.Contains(file, "record/decision-log/") {
			t.Errorf("a decision was published without the opt-in: %s", file)
		}
	}
	if strings.Contains(outFile(t, out, "record.json"), "dcn-") {
		t.Error("record.json carries a decision the repository did not opt in to publishing")
	}
	if strings.Contains(outFile(t, out, "record/index.html"), "/record/decision-log/") {
		t.Error("the dashboard links a decision log the repository did not opt in to publishing")
	}
}

// TestDevelopmentPageAndChartLeaveDecisionsOut: the development page and the
// relationship chart are built for the stores that move, and hundreds of
// one-line decisions would bury them. With the log published, neither carries a
// decision, the chart's own data holds none among its nodes, and the genealogy
// is drawn exactly as it is without them.
func TestDevelopmentPageAndChartLeaveDecisionsOut(t *testing.T) {
	without := newFixture(t)
	without.addDecisions(false)
	plain := t.TempDir()
	buildFixture(t, without, plain)

	f := newFixture(t)
	decs := f.addDecisions(true)
	out := t.TempDir()
	buildFixture(t, f, out)

	dev := outFile(t, out, "record/development/index.html")
	graph := outFile(t, out, "record/graph/index.html")
	for _, d := range decs {
		if strings.Contains(dev, d.ID) {
			t.Errorf("the development page lists %s", d.ID)
		}
		if strings.Contains(graph, d.ID) {
			t.Errorf("the relationship chart's page carries %s", d.ID)
		}
	}
	if strings.Contains(graph, `data-t="decision"`) {
		t.Error("the relationship chart offers a decision filter")
	}

	var exp struct {
		Nodes     []ExportNode `json:"nodes"`
		Decisions []ExportNode `json:"decisions"`
		Layout    Arrangements `json:"layout"`
	}
	if err := json.Unmarshal([]byte(outFile(t, out, "record.json")), &exp); err != nil {
		t.Fatal(err)
	}
	for _, n := range exp.Nodes {
		if n.Type == "decision" {
			t.Errorf("the chart's nodes carry the decision %s", n.ID)
		}
	}
	if len(exp.Layout.Coil) != len(exp.Nodes) {
		t.Errorf("the chart places %d bubbles for %d nodes", len(exp.Layout.Coil), len(exp.Nodes))
	}
	if len(exp.Decisions) != len(decs) {
		t.Errorf("record.json carries %d decisions, want %d", len(exp.Decisions), len(decs))
	}

	svg := regexp.MustCompile(`(?s)<svg viewBox="[^"]*" class="tlsvg".*?</svg>`)
	a := svg.FindString(outFile(t, plain, "record/index.html"))
	b := svg.FindString(outFile(t, out, "record/index.html"))
	if a == "" || a != b {
		t.Error("publishing the decisions changed the genealogy")
	}
}
