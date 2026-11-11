package spec

import (
	"strings"
	"testing"
)

// TestUnsettledQuotesNoTerminalControl: the spec is text agents and people
// write, so the line a statement quotes reaches a refusal with its terminal
// controls masked and its length capped.
func TestUnsettledQuotesNoTerminalControl(t *testing.T) {
	content := "---\nid: spc-1\n---\n# T\n\nNot READY: \x1b]0;owned\x07 " + strings.Repeat("x", 500) + "\n"
	got := Unsettled(content)
	if len(got) != 1 || got[0].Line != 6 || got[0].Kind != UnsettledNotReady {
		t.Fatalf("want one not-READY statement on line 6, got %+v", got)
	}
	d := got[0].Describe()
	if strings.ContainsAny(d, "\x1b\x07") {
		t.Fatalf("a statement quotes a terminal control raw: %q", d)
	}
	if n := len([]rune(got[0].Text)); n > maxStatementRunes+1 {
		t.Fatalf("a statement's text is capped at %d runes, got %d", maxStatementRunes, n)
	}
}

// TestUnsettledReadsTheBodyAlone: the frontmatter and a fenced block are not
// the spec speaking about itself.
func TestUnsettledReadsTheBodyAlone(t *testing.T) {
	content := "---\nid: spc-1\nnote: \"x. Not READY: in frontmatter\"\n---\n# T\n\n```\nNot READY: fenced\n```\n"
	if got := Unsettled(content); len(got) != 0 {
		t.Fatalf("want nothing outside the body's prose, got %+v", got)
	}
}
