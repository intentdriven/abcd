package mdrecord

import (
	"reflect"
	"testing"
)

// nbsp is a line CommonMark reads as text and strings.TrimSpace empties: a
// lone no-break space. Each reader below must carry its paragraph or block on
// through it, as the page does, and never end one there.
const nbsp = " "

func TestBlankLineIsCommonMarks(t *testing.T) {
	for ln, want := range map[string]bool{
		"": true, " ": true, "\t": true, " \t \r": true, "\r": true,
		nbsp: false, "\u0085": false, " ": false, "\v": false, "\f": false, " x": false,
	} {
		if got := BlankLine(ln); got != want {
			t.Errorf("BlankLine(%q) = %v, want %v", ln, got, want)
		}
	}
}

func TestANoBreakSpaceLineEndsNoBlock(t *testing.T) {
	// A fence closer followed by a no-break space is fence text, so the next
	// delimiter closes the fence rather than opening one that hides the rest.
	fenced := FencedUnderEveryRule([]string{"```", "a", "```" + nbsp, "<h2>x</h2>", "```", "<h2>after</h2>"})
	if want := []bool{true, true, true, true, true, false}; !reflect.DeepEqual(fenced, want) {
		t.Errorf("a closer trailed by a no-break space: fenced %v, want %v", fenced, want)
	}
	// A list item's fence ends at a non-blank line at column 0, which is then
	// live: the no-break-space line is that line.
	if fenced := FencedUnderEveryRule([]string{"- a", "  ```", nbsp, "<h2>x</h2>"}); fenced[2] || fenced[3] {
		t.Errorf("a no-break-space line at column 0 left the item's fence open: %v", fenced)
	}
	// An HTML block runs through the line and holds the fence below it.
	if _, ok := ReadPage([]string{"<div>", nbsp, "```", "<h2>x</h2>", "```"}).Unsure(); !ok {
		t.Error("a fence an HTML block holds past a no-break-space line was read as sure")
	}
	lines := []string{"**The rule.** a", nbsp, "b", "", "c"}
	if _, end, ok := LabelledParagraph(lines, "The rule"); !ok || end != 3 {
		t.Errorf("a labelled paragraph over a no-break-space line ends at %d (ok %v), want 3", end, ok)
	}
	bullets := []string{"- a", nbsp, "b", ""}
	if got := BulletBlocks(bullets, Mask(bullets), 0, len(bullets)); len(got) != 1 || got[0].End != 3 {
		t.Errorf("a bullet over a no-break-space line: %+v, want one block ending at 3", got)
	}
	section := []string{"text", nbsp, "[a]: /u"}
	if refs := PeelTrailingLinkRefs(&section); !reflect.DeepEqual(refs, []string{"[a]: /u"}) ||
		!reflect.DeepEqual(section, []string{"text", nbsp}) {
		t.Errorf("peeling past a no-break-space line: refs %q, section %q", refs, section)
	}
}
