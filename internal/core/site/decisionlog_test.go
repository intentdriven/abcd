package site

// The decision records' tests at the site (spc-2610110931381316, step 5): every decision
// record on one page, oldest first and grouped by month, published only where
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
