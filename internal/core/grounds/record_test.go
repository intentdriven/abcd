package grounds

import (
	"strings"
	"testing"
)

func mustGrounds(t *testing.T) Grounds {
	t.Helper()
	g, err := New(Pursued, conjecture)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

// Two live Grounds headings are refused, each named by its body line and its
// own text, and without claiming a depth the second one does not have: the
// pattern matches every depth, so a `### Grounds` is one of the two
// (iss-2608301908288212).
func TestAmbiguousRefusalNamesEachHeadingWhereItIs(t *testing.T) {
	file := "---\nid: x\n---\n\n# T\n\n## Grounds\n\n- pursued: " + conjecture + "\n\n### Grounds\n\n- deferred: " + conjecture + "\n"
	_, err := AppendToRecord(file, mustGrounds(t))
	if err == nil {
		t.Fatal("a body with two live Grounds headings was appended to")
	}
	msg := err.Error()
	if strings.Contains(msg, "`## Grounds`") {
		t.Errorf("the refusal names a `## Grounds` depth the second heading does not have: %s", msg)
	}
	// Counted in the body a reader renders: "# T" is line 1.
	for _, want := range []string{`body line 3, "## Grounds"`, `body line 7, "### Grounds"`} {
		if !strings.Contains(msg, want) {
			t.Errorf("the refusal does not locate %s: %s", want, msg)
		}
	}
}

// The unclosed opener is numbered in the body a reader renders, which starts
// below the blank separator, and quoted clipped, since a record can carry a
// line of any length (iss-2608301908288212).
func TestReadBackRefusalNumbersAndClipsTheOpener(t *testing.T) {
	opener := "```" + strings.Repeat("x", 5000)
	file := "---\nid: x\n---\n\n# T\n\n" + opener + "\n"
	_, err := AppendToRecord(file, mustGrounds(t))
	if err == nil {
		t.Fatal("an append below an unclosed fence read back")
	}
	msg := err.Error()
	if !strings.Contains(msg, "body line 3,") {
		t.Errorf("the opener is line 3 of the rendered body: %s", msg)
	}
	if strings.Contains(msg, strings.Repeat("x", 100)) || len(msg) > 1000 {
		t.Errorf("the refusal quotes the opener unclipped (%d bytes)", len(msg))
	}
}

// A record with frontmatter and no body gains the section one blank line below
// the closing delimiter, as a record with prose gains it below the prose
// (iss-2608301908288212).
func TestAppendToAFrontmatterOnlyRecordLeavesOneBlankLine(t *testing.T) {
	g := mustGrounds(t)
	got, err := AppendToRecord("---\nid: x\n---\n", g)
	if err != nil {
		t.Fatal(err)
	}
	want := "---\nid: x\n---\n\n## Grounds\n\n" + g.Bullet() + "\n"
	if got != want {
		t.Errorf("AppendToRecord = %q, want %q", got, want)
	}
}
