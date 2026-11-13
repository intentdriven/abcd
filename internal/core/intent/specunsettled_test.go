package intent

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/spec"
)

// TestSpecUnsettledMasksAndCaps: a spec's open questions and its statements
// reach the spec_body row masked for a terminal, at most three of each named
// and the rest counted, each clipped (iss-2611101318042398).
func TestSpecUnsettledMasksAndCaps(t *testing.T) {
	content := "---\nid: spc-1\n---\n# S\n\n" +
		"Not READY: one.\nNot READY: two.\nNot READY: three.\nNot READY: four.\nNot READY: five.\n\n" +
		"## Open Questions\n\n- \x1b]0;owned\x07 which " + strings.Repeat("c", 400) + "?\n- b?\n- c?\n- d?\n"
	got := specUnsettled(content)
	noRawControl(t, "specUnsettled", got)
	for _, w := range []string{"it asks 4 open question(s)", "| and 1 more", "and 2 more", "line 6", "line 8"} {
		if !strings.Contains(got, w) {
			t.Errorf("specUnsettled = %q, want it to contain %q", got, w)
		}
	}
	if strings.Contains(got, "line 9") || strings.Contains(got, strings.Repeat("c", 200)) {
		t.Fatalf("specUnsettled names a fourth statement or an unclipped question: %q", got)
	}
}

// TestSpecBodyCheckExemptsWhatIsNeverBuilt: a closed spec, and an intent in
// shipped/, superseded/ or disciplines/, is history; what its spec says about
// readiness holds nothing, as record-lint reads open specs alone.
func TestSpecBodyCheckExemptsWhatIsNeverBuilt(t *testing.T) {
	root := t.TempDir()
	const rel = ".abcd/development/specs/closed/spc-1-s.md"
	writeFile(t, root, filepath.FromSlash(rel), "---\nid: spc-1\nintent: itd-10\n---\n# S\n\nNot READY: once.\n")
	cases := []struct {
		name   string
		bucket string
		status string
		ok     bool
	}{
		{"a planned intent, an open spec", BucketPlanned, spec.StatusOpen, false},
		{"a planned intent, a closed spec", BucketPlanned, spec.StatusClosed, true},
		{"a shipped intent", BucketShipped, spec.StatusOpen, true},
		{"a superseded intent", BucketSuperseded, spec.StatusOpen, true},
		{"a discipline", BucketDisciplines, spec.StatusOpen, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c2, err := specBodyCheck(root, Intent{ID: "itd-10", Bucket: c.bucket}, spec.Spec{ID: "spc-1", Path: rel, Status: c.status}, true)
			if err != nil {
				t.Fatal(err)
			}
			if c2.OK != c.ok {
				t.Fatalf("spec_body OK = %v, want %v: %+v", c2.OK, c.ok, c2)
			}
		})
	}
}

// TestSpecBodyCheckNamesAnOpenerBelowARunLeftOpen: the spec_body row the
// build's readiness gate reads takes spec.Unsettled's layout statement, so an
// open spec holding a comment opener quoted in a one-line code span, below a
// backtick run a line leaves open and with no closer after it on its line, is
// refused with that line named.
func TestSpecBodyCheckNamesAnOpenerBelowARunLeftOpen(t *testing.T) {
	root := t.TempDir()
	const rel = ".abcd/development/specs/open/spc-1-s.md"
	writeFile(t, root, filepath.FromSlash(rel), "---\nid: spc-1\nintent: itd-10\n---\n# S\n\n"+
		"Stray ` tick\nsee `<!-- parked` here\n## Parked\n-->\n")
	c, err := specBodyCheck(root, Intent{ID: "itd-10", Bucket: BucketPlanned}, spec.Spec{ID: "spc-1", Path: rel, Status: spec.StatusOpen}, true)
	if err != nil {
		t.Fatal(err)
	}
	if c.OK || !strings.Contains(c.Detail, "line 8: the text from this line on is not read one way") {
		t.Fatalf("spec_body = %+v, want it refused with line 8 named", c)
	}
}

// TestOpenQuestionsReadsASubheading: a question asked under a sub-heading of
// `## Open Questions` is a question; the section ends at a heading of its own
// depth.
func TestOpenQuestionsReadsASubheading(t *testing.T) {
	content := "# T\n\n## Open Questions\n\n### Relay\n\n- How does the relay start?\n\n## Decisions\n\n- not a question\n"
	if got := openQuestions(content); len(got) != 1 || got[0] != "How does the relay start?" {
		t.Fatalf("openQuestions = %q, want the question under the sub-heading alone", got)
	}
}
