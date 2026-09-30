package docfidelity

import (
	"reflect"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/surface"
)

// fixture is a small shipped surface: two verbs, a sub-verb, a hidden verb with
// a child, a moved spelling and one agent.
func fixture() Inputs {
	return Inputs{
		Commands: []surface.Command{
			{Path: "abcd"},
			{Path: "abcd capture"},
			{Path: "abcd capture list"},
			{Path: "abcd hook", Hidden: true},
			{Path: "abcd hook session-end"},
			{Path: "abcd old", MovedTo: "abcd capture"},
		},
		Agents: []string{"scribe"},
		Chapters: map[string]string{
			"06-capture.md": "### `abcd capture`\n\nRun `abcd capture list --json` to list.\n",
			"32-scribe.md":  "The `scribe` agent transcribes.\n",
		},
		Population: []string{"itd-9"},
	}
}

// panicReviewer is the reviewer stub the spec asks for: a refusal from layer 1
// must be composed before the delegated call is constructed.
type panicReviewer struct{}

func (panicReviewer) Review() Review { panic("layer 2 reached after a layer-1 refusal") }

type fixedReviewer Review

func (f fixedReviewer) Review() Review { return Review(f) }

var promote = fixedReviewer{Status: ReviewMatch, Commit: "c1"}

func TestLayerOneCoverageTable(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*Inputs)
		refuses string // "" = allowed
	}{
		{"all covered", func(*Inputs) {}, ""},
		{"new verb without a chapter", func(in *Inputs) {
			in.Commands = append(in.Commands, surface.Command{Path: "abcd rules"})
		}, "verb `abcd rules`"},
		{"new verb with a chapter", func(in *Inputs) {
			in.Commands = append(in.Commands, surface.Command{Path: "abcd rules"})
			in.Chapters["40-rules.md"] = "### `abcd rules`\n"
		}, ""},
		{"new sub-verb without a chapter", func(in *Inputs) {
			in.Commands = append(in.Commands, surface.Command{Path: "abcd capture drop"})
		}, "sub-verb `abcd capture drop`"},
		{"new sub-verb with a chapter", func(in *Inputs) {
			in.Commands = append(in.Commands, surface.Command{Path: "abcd capture drop"})
			in.Chapters["06-capture.md"] += "### `abcd capture drop`\n"
		}, ""},
		{"new agent without a chapter", func(in *Inputs) {
			in.Agents = append(in.Agents, "sota-researcher")
		}, "agent `sota-researcher`"},
		{"new agent with a chapter", func(in *Inputs) {
			in.Agents = append(in.Agents, "sota-researcher")
			in.Chapters["32-scribe.md"] += "See agents/sota-researcher.md.\n"
		}, ""},
		{"a bare word is not a naming", func(in *Inputs) {
			in.Agents = append(in.Agents, "lens")
			in.Chapters["32-scribe.md"] += "The lens is wide.\n"
		}, "agent `lens`"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := fixture()
			c.mutate(&in)
			var r Reviewer = promote
			if c.refuses != "" {
				r = panicReviewer{}
			}
			v := Judge(in, r, false)
			if c.refuses == "" {
				if v.Refuse {
					t.Fatalf("refused a covered surface: %v", v.Reasons)
				}
				return
			}
			if !v.Refuse {
				t.Fatalf("allowed an undocumented surface (%s)", c.refuses)
			}
			if !strings.Contains(strings.Join(v.Reasons, "\n"), c.refuses) {
				t.Fatalf("refusal does not name %q: %v", c.refuses, v.Reasons)
			}
			if v.Review != nil {
				t.Fatalf("layer 2 ran after a layer-1 refusal")
			}
		})
	}
}

func TestHiddenAndMovedSurfacesAreNotShippedSurfaces(t *testing.T) {
	v := Judge(fixture(), promote, false)
	for _, row := range v.Coverage {
		if strings.HasPrefix(row.Name, "abcd hook") || row.Name == "abcd old" || row.Name == "abcd" {
			t.Errorf("coverage row for a surface that does not ship as its own: %q", row.Name)
		}
	}
	if len(v.Coverage) != 3 {
		t.Fatalf("coverage rows = %d, want 3 (capture, capture list, scribe): %+v", len(v.Coverage), v.Coverage)
	}
}

func TestLayerTwoOutcomes(t *testing.T) {
	cases := []struct {
		name    string
		review  Review
		refuses string
	}{
		{"matching PROMOTE receipt", Review{Status: ReviewMatch, Commit: "c1"}, ""},
		{"confirmed false sentence", Review{Status: ReviewHold, Commit: "c1", Findings: []Sentence{
			{Doc: DocBrief, Chapter: "06-capture.md", Sentence: "capture list prints YAML.", Evidence: "cli.go:10 prints JSON"},
		}}, `"capture list prints YAML."`},
		{"no receipt", Review{Status: ReviewNone, Commit: "c1"}, RunReviewFirst},
		{"stale receipt", Review{Status: ReviewStale, Commit: "c1", Named: "c0"}, RunReviewFirst},
		{"unreadable receipt", Review{Status: ReviewInvalid, Commit: "c1", Problems: []string{"malformed JSON"}}, RunReviewFirst},
		{"inconclusive verdict", Review{Status: ReviewInconclusive, Commit: "c1", Verdict: "INCONCLUSIVE"}, RunReviewFirst},
		{"HOLD naming no sentence", Review{Status: ReviewHold, Commit: "c1"}, "names no sentence"},
		{"PROMOTE carrying a false brief sentence", Review{Status: ReviewMatch, Commit: "c1", Verdict: "PROMOTE", Findings: []Sentence{
			{Doc: DocBrief, Chapter: "06-capture.md", Sentence: "capture list prints YAML.", Evidence: "cli.go:10 prints JSON"},
		}}, `"capture list prints YAML."`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v := Judge(fixture(), fixedReviewer(c.review), false)
			if c.refuses == "" {
				if v.Refuse {
					t.Fatalf("refused: %v", v.Reasons)
				}
				return
			}
			if !v.Refuse {
				t.Fatalf("allowed: %+v", v)
			}
			joined := strings.Join(v.Reasons, "\n")
			if !strings.Contains(joined, c.refuses) {
				t.Fatalf("refusal lacks %q: %v", c.refuses, v.Reasons)
			}
			if !strings.Contains(joined, "itd-9") {
				t.Fatalf("refusal does not name the population: %v", v.Reasons)
			}
		})
	}
}

// ac-6: the pair — the public-doc finding is present AND the verdict allows.
func TestPublicDocSentenceIsReportedAndNeverRefuses(t *testing.T) {
	for _, status := range []ReviewStatus{ReviewMatch, ReviewHold} {
		r := Review{Status: status, Commit: "c1", Findings: []Sentence{
			{Doc: DocPublic, Chapter: "docs/guide.md", Sentence: "abcd sings.", Evidence: "it does not"},
		}}
		v := Judge(fixture(), fixedReviewer(r), false)
		if v.Refuse {
			t.Fatalf("%s: a public-doc sentence refused: %v", status, v.Reasons)
		}
		if len(v.Public) != 1 || v.Public[0].Sentence != "abcd sings." {
			t.Fatalf("%s: the public-doc finding is not reported: %+v", status, v.Public)
		}
	}
}

// ac-7: report mode yields the same findings and never a refusal.
func TestReportModeReportsTheSameFindingsAndNeverRefuses(t *testing.T) {
	in := fixture()
	in.Commands = append(in.Commands, surface.Command{Path: "abcd rules"})
	hold := fixedReviewer{Status: ReviewHold, Commit: "c1", Findings: []Sentence{{Doc: DocBrief, Chapter: "06-capture.md", Sentence: "x.", Evidence: "y"}}}
	gate := Judge(in, hold, false)
	in2 := fixture()
	in2.Commands = append(in2.Commands, surface.Command{Path: "abcd rules"})
	rep := Judge(in2, hold, true)
	if rep.Refuse {
		t.Fatalf("report mode refused")
	}
	if !reflect.DeepEqual(gate.Uncovered, rep.Uncovered) {
		t.Fatalf("report mode's coverage findings differ: gate %+v report %+v", gate.Uncovered, rep.Uncovered)
	}
	if len(rep.False) != 1 {
		t.Fatalf("report mode dropped the false sentence: %+v", rep.False)
	}
	if len(rep.Reasons) == 0 {
		t.Fatalf("report mode stated no findings")
	}
}

// ac-8: a chapter ahead of the last cut — describing a surface the binary
// ships — is current. The judgement reads the tree, never a tag: a chapter
// naming a surface the tree does not carry is no finding either.
func TestTheLegitimateLeadIsNotDrift(t *testing.T) {
	in := fixture()
	in.Chapters["41-next.md"] = "### `abcd next`\n\nThe `planner` agent is coming.\n"
	v := Judge(in, promote, false)
	if v.Refuse || len(v.Uncovered) != 0 {
		t.Fatalf("a brief ahead of the tree was reported: %+v", v)
	}
}
