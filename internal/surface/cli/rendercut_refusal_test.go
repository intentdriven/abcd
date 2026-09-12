package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/release"
)

// The detector iss-2609091046297503 owes. renderCut passed a refusal reason
// through termsafe.Sanitize, which masks a newline to '?', so the split that
// followed found nothing to split and a reason written as a heading plus one
// line per record — the surface guard's break list, the stale-intent refusal,
// the findings gate's enumeration — arrived as one run-on line. The JSON
// envelope carried it correctly throughout, which is why it survived: only the
// human render was wrong.
//
// Two halves, because the fix has two ways to be wrong. A multi-line reason must
// render as that many lines with no substitution, and a single-line reason must
// render unchanged — a block sanitiser applied without thought would be just as
// happy to introduce a break as to preserve one.

// refusedCut is a refused cut carrying exactly the refusals a case names, with
// every other field left at its zero value: this test judges the refusal block
// and nothing above it.
func refusedCut(refusals ...release.Refusal) release.Cut {
	return release.Cut{BaseTag: "v0.7.1", Refusals: refusals}
}

// refusalBlock returns the indented lines renderCut wrote under one refusal
// heading, so an assertion reads the lines rather than the whole report.
func refusalBlock(t *testing.T, cut release.Cut) []string {
	t.Helper()
	var buf bytes.Buffer
	renderCut(&buf, "abcd launch ship", cut)
	var out []string
	in := false
	for _, line := range strings.Split(buf.String(), "\n") {
		switch {
		case strings.HasPrefix(line, "  refused ("):
			in = true
		case in && strings.HasPrefix(line, "    "):
			out = append(out, strings.TrimPrefix(line, "    "))
		case in && line != "":
			in = false
		}
	}
	return out
}

func TestRenderCutKeepsAMultiLineRefusalsLineStructure(t *testing.T) {
	reason := "this cycle captured findings it has not answered — each was recorded since v0.7.1 " +
		"and is still open:\n  - iss-1 [major] a/b.md\n  - iss-2 [critical] c/d.md\nfix it and resolve " +
		"the record in this cut, record the decision not to fix it, or defer it OUT LOUD."
	want := strings.Split(reason, "\n")

	got := refusalBlock(t, refusedCut(release.Refusal{Kind: release.RefusalUnfixedFinding, Reason: reason}))
	if len(got) != len(want) {
		t.Fatalf("a %d-line refusal reason rendered as %d line(s): %q", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d rendered as %q, want %q", i+1, got[i], want[i])
		}
	}
	// The failure this record names was a '?' where a line break belonged, so
	// name it rather than leaving it to the length check alone: a reason that
	// collapses to one line AND keeps its length would still be the defect.
	for _, line := range got {
		if strings.Contains(line, "?") {
			t.Errorf("the render substituted a question mark into %q; the newlines are the render's "+
				"own line structure, not an injected control sequence", line)
		}
	}
}

func TestRenderCutLeavesASingleLineRefusalUnchanged(t *testing.T) {
	const reason = "no release tag: the cut has no immutable base to measure from"
	got := refusalBlock(t, refusedCut(release.Refusal{Kind: release.RefusalNoReleaseTag, Reason: reason}))
	if len(got) != 1 || got[0] != reason {
		t.Fatalf("a single-line reason rendered as %q, want exactly [%q]", got, reason)
	}
}

// TestRenderCutStillNeutralisesAControlSequenceInARefusal holds the property the
// single-line sanitiser was there for in the first place. A refusal quotes
// record frontmatter, which is author-supplied text reaching a terminal, so
// keeping the line structure must not also let an escape sequence through
// (brief invariant 13).
func TestRenderCutStillNeutralisesAControlSequenceInARefusal(t *testing.T) {
	got := refusalBlock(t, refusedCut(release.Refusal{
		Kind:   release.RefusalUnlabelled,
		Reason: "record \x1b[31miss-1\x1b[0m carries no impact\n  - second line",
	}))
	joined := strings.Join(got, "\n")
	if strings.Contains(joined, "\x1b") {
		t.Errorf("an escape byte survived the render: %q", joined)
	}
	if len(got) != 2 {
		t.Errorf("the sanitised reason rendered as %d line(s), want 2: %q", len(got), got)
	}
}
