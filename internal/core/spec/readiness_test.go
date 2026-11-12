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
// the spec speaking about itself. The frontmatter line opens with the phrase,
// so it would be reported if the frontmatter were read.
func TestUnsettledReadsTheBodyAlone(t *testing.T) {
	content := "---\nid: spc-1\nNot READY: in frontmatter\n---\n# T\n\n```\nNot READY: fenced\n```\n"
	if got := Unsettled(content); len(got) != 0 {
		t.Fatalf("want nothing outside the body's prose, got %+v", got)
	}
}

// TestUnsettledForms is every form Unsettled reports and every place it does
// not read, each on line 2 of a body after a one-line frontmatter.
func TestUnsettledForms(t *testing.T) {
	const fm = "---\nid: spc-1\n---\n# T\n"
	cases := []struct {
		name string
		body string
		want []Statement // Line and Kind only
	}{
		{"a line opening Not READY", "Not READY: a ruling is owed.\n", []Statement{{Line: 5, Kind: UnsettledNotReady}}},
		{"in capitals", "NOT READY until the cache is settled.\n", []Statement{{Line: 5, Kind: UnsettledNotReady}}},
		{"after a list marker", "- Not READY: owed.\n", []Statement{{Line: 5, Kind: UnsettledNotReady}}},
		{"after a numbered marker", "3. Not READY: owed.\n", []Statement{{Line: 5, Kind: UnsettledNotReady}}},
		{"after a blockquote", "> Not READY: owed.\n", []Statement{{Line: 5, Kind: UnsettledNotReady}}},
		{"in bold", "**Not READY**\n", []Statement{{Line: 5, Kind: UnsettledNotReady}}},
		{"in underscore emphasis", "_Not READY_\n", []Statement{{Line: 5, Kind: UnsettledNotReady}}},
		{"as a heading", "## Not READY\n", []Statement{{Line: 5, Kind: UnsettledNotReady}}},
		{"after a task box", "- [ ] Not READY: owed.\n", []Statement{{Line: 5, Kind: UnsettledNotReady}}},
		{"after a sentence", "Written. Not READY: owed.\n", []Statement{{Line: 5, Kind: UnsettledNotReady}}},
		{"led by a colon", "Status: Not READY\n", []Statement{{Line: 5, Kind: UnsettledNotReady}}},
		{"led by a bold colon", "**Status:** Not READY\n", []Statement{{Line: 5, Kind: UnsettledNotReady}}},
		{"in brackets", "The design (Not READY: owed) waits.\n", []Statement{{Line: 5, Kind: UnsettledNotReady}}},
		{"in square brackets", "[Not READY] owed.\n", []Statement{{Line: 5, Kind: UnsettledNotReady}}},
		{"a hard wrap between the words", "The step is written. Not\nREADY: a ruling is owed.\n", []Statement{{Line: 5, Kind: UnsettledNotReady}}},
		{"a hard wrap into a list continuation", "- Written. Not\n  READY: owed.\n", []Statement{{Line: 5, Kind: UnsettledNotReady}}},
		{"inside Open Questions too", "## Open Questions\n\n- resolved: Not READY: was owed.\n", []Statement{{Line: 7, Kind: UnsettledNotReady}}},
		{"after a comment closes on the line", "<!--\nparked\n--> Not READY: owed.\n", []Statement{{Line: 7, Kind: UnsettledNotReady}}},
		{"before a comment opens on the line and closes in its paragraph", "Not READY: owed. <!-- note\n-->\n", []Statement{{Line: 5, Kind: UnsettledNotReady}}},
		{"before a comment opens on the line and runs past a blank line", "Not READY: owed. <!-- note\n\n-->\n", []Statement{{Line: 5, Kind: UnsettledNotReady}, {Line: 5, Kind: UnsettledLayout}}},
		{"before a comment opens and closes on the line", "Not READY: owed. <!-- note -->\n", []Statement{{Line: 5, Kind: UnsettledNotReady}}},
		{"a bold Open label", "- **Open:** one cache file, or a rollup?\n", []Statement{{Line: 5, Kind: UnsettledOpenLabel}}},
		{"a bold Open label with the colon after", "**Open**: which cache?\n", []Statement{{Line: 5, Kind: UnsettledOpenLabel}}},
		{"an underscore-bold Open label", "__Open:__ which cache?\n", []Statement{{Line: 5, Kind: UnsettledOpenLabel}}},
		{"an Open question label", "**Open questions:** which cache?\n", []Statement{{Line: 5, Kind: UnsettledOpenLabel}}},
		{"an Open point label", "**Open point:** which cache?\n", []Statement{{Line: 5, Kind: UnsettledOpenLabel}}},
		{"an Open design question label", "**Open design questions**: which cache?\n", []Statement{{Line: 5, Kind: UnsettledOpenLabel}}},
		{"an Open label under a sub-heading of Open Questions", "## Open Questions\n\n### Relay\n\n- **Open:** which relay?\n", nil},
		{"an Open label after Open Questions ends", "## Open Questions\n\n- a?\n\n## Decisions\n\n**Open:** b\n", []Statement{{Line: 11, Kind: UnsettledOpenLabel}}},
		{"an unclosed fence hides the rest and is reported", "```\nNot READY: hidden\n## Open Questions\n- q?\n", []Statement{{Line: 5, Kind: UnsettledUnclosed}}},
		{"an unclosed comment hides the rest and is reported", "<!--\nNot READY: hidden\n", []Statement{{Line: 5, Kind: UnsettledUnclosed}}},
		{"an unclosed nested fence is read and reported", "- item\n\n    ```\n    Not READY: hidden\n", []Statement{{Line: 7, Kind: UnsettledLayout}, {Line: 8, Kind: UnsettledNotReady}}},

		{"after an exclamation", "Done! Not READY: owed.\n", []Statement{{Line: 5, Kind: UnsettledNotReady}}},
		{"after a question mark", "Done? Not READY: owed.\n", []Statement{{Line: 5, Kind: UnsettledNotReady}}},
		{"after a semicolon", "The step is written; Not READY until the ruling.\n", []Statement{{Line: 5, Kind: UnsettledNotReady}}},
		{"after an em dash", "_Draft — Not READY: a ruling is owed._\n", []Statement{{Line: 5, Kind: UnsettledNotReady}}},
		{"after an en dash", "Draft – NOT READY.\n", []Statement{{Line: 5, Kind: UnsettledNotReady}}},
		{"after a spaced hyphen", "Draft - Not READY.\n", []Statement{{Line: 5, Kind: UnsettledNotReady}}},
		{"in a table's first cell", "| Not READY | owed |\n", []Statement{{Line: 5, Kind: UnsettledNotReady}}},
		{"in a later table cell", "| status | **Not READY** |\n", []Statement{{Line: 5, Kind: UnsettledNotReady}}},
		{"an Open label in capitals", "**OPEN:** which cache?\n", []Statement{{Line: 5, Kind: UnsettledOpenLabel}}},
		{"an Open label in lower case", "**open:** which cache?\n", []Statement{{Line: 5, Kind: UnsettledOpenLabel}}},
		{"an Open label in mixed case", "**oPeN Question:** which cache?\n", []Statement{{Line: 5, Kind: UnsettledOpenLabel}}},
		{"an Open label in a paragraph under Open Questions", "## Open Questions\n\n**Open:** which cache?\n", []Statement{{Line: 7, Kind: UnsettledOpenLabel}}},
		{"an Open label on an indented bullet under Open Questions", "## Open Questions\n\n- which store?\n  - **Open:** which cache?\n", []Statement{{Line: 8, Kind: UnsettledOpenLabel}}},
		{"an Open label in a table row under Open Questions", "## Open Questions\n\n| q | **Open:** which cache? |\n", []Statement{{Line: 7, Kind: UnsettledOpenLabel}}},
		{"an Open label on a settled item", "## Open Questions\n\n- **Resolved — LRU.** **Open:** which size?\n", []Statement{{Line: 7, Kind: UnsettledOpenLabel}}},
		{"an indented run continuing a paragraph", "The design is written\n    ```\n    Not READY: a ruling is owed.\nNext paragraph.\n", []Statement{{Line: 6, Kind: UnsettledLayout}, {Line: 7, Kind: UnsettledNotReady}}},
		{"a list-item fence a column-0 line ends", "- item\n  ```\n  code\nNot READY: a ruling is owed.\n\n```\nlater\n```\n", []Statement{{Line: 8, Kind: UnsettledNotReady}, {Line: 8, Kind: UnsettledLayout}}},
		{"a nested opener closed nearer the margin", "- item\n\n    ```\n    code\n  ```\n\n## Open Questions\n\n- which cache?\n\n```go\nx\n```\n", []Statement{{Line: 7, Kind: UnsettledLayout}}},
		{"an opener four columns past the item's content", "- item\n\n      ```\n  Not READY: a ruling is owed.\n\nNext paragraph.\n", []Statement{{Line: 7, Kind: UnsettledLayout}, {Line: 8, Kind: UnsettledNotReady}}},
		{"a line left of the item inside a nested fence", "- item\n\n    ```\n Not READY: a ruling is owed.\n    ```\n", []Statement{{Line: 7, Kind: UnsettledLayout}, {Line: 8, Kind: UnsettledNotReady}}},
		{"a line left of the item inside its fence", "- item\n  ```\n Not READY: a ruling is owed.\n  ```\n", []Statement{{Line: 7, Kind: UnsettledNotReady}, {Line: 7, Kind: UnsettledLayout}}},
		{"a fence opener on a list marker's line", "- ```\n  code\n  ```\nNot READY: a ruling is owed.\n```\n", []Statement{{Line: 5, Kind: UnsettledLayout}, {Line: 8, Kind: UnsettledNotReady}}},
		{"a comment opened after text runs past a blank line", "Text <!-- note\n\n## Open Questions\n\n- which cache?\n\n-->\n", []Statement{{Line: 5, Kind: UnsettledLayout}}},
		{"a settled marker in a comment opened after text", "## Open Questions\n\n- which cache? <!-- note\n\n  resolved: LRU -->\n", []Statement{{Line: 7, Kind: UnsettledLayout}}},
		{"a fence four columns in under a list item is read and reported", "- item\n\n    ```\n    Not READY: sample\n    **Open:** sample\n    ```\n", []Statement{{Line: 7, Kind: UnsettledLayout}, {Line: 8, Kind: UnsettledNotReady}, {Line: 9, Kind: UnsettledOpenLabel}}},

		{"a one-line comment is not read", "<!-- Done. Not READY: parked -->\n<!-- - **Open:** which cache? -->\n", nil},
		{"inside a multi-line comment", "<!--\nNot READY: parked\n-->\n", nil},
		{"a tilde fence", "~~~\nNot READY: example\n~~~\n", nil},
		{"a fence at the item's content column", "- item\n\n  ```\n  Not READY: sample\n  **Open:** sample\n  ```\n", nil},
		{"inline code", "The board prints `Not READY` and `**Open:**`.\n", nil},
		{"mid-sentence prose", "A record that was READY before is NOT READY after it.\n", nil},
		{"not in capitals", "Not ready: lower case. And not READY either.\n", nil},
		{"lower-case not after a stop", "It is done. not READY is not matched.\n", nil},
		{"a hyphen with no spaces", "half-Not READY\n", nil},
		{"a longer word", "Not READYING anything.\n", nil},
		{"a bold span opening with Open", "**Open pull requests** are listed; **Open, and not gating** too.\n", nil},
		{"a wrap with a blank line between", "Written. Not\n\nREADY: a new paragraph.\n", nil},
		{"an Open label under Open Questions", "## Open Questions\n\n- **Open:** which cache?\n", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Unsettled(fm + c.body)
			if len(got) != len(c.want) {
				t.Fatalf("Unsettled = %+v, want %+v", got, c.want)
			}
			for i := range got {
				if got[i].Line != c.want[i].Line || got[i].Kind != c.want[i].Kind {
					t.Fatalf("statement %d = %+v, want line %d %s", i, got[i], c.want[i].Line, c.want[i].Kind)
				}
			}
		})
	}
}

// TestUnsettledDescribesAnUnclosedOpener: the finding names the construct and
// what it hides, never the opener's own text.
func TestUnsettledDescribesAnUnclosedOpener(t *testing.T) {
	got := Unsettled("# T\n<!-- \x1b[31m secret\nNot READY\n")
	if len(got) != 1 || got[0].Kind != UnsettledUnclosed {
		t.Fatalf("want one unclosed statement, got %+v", got)
	}
	d := got[0].Describe()
	if !strings.Contains(d, "HTML comment") || !strings.Contains(d, "nothing below this line is read") || strings.Contains(d, "secret") {
		t.Fatalf("Describe = %q", d)
	}
}

// TestUnsettledReadsCodeSpansAsThePagePairsThem: a span the page pairs over a
// line break is not read, and text the page shows as prose around one is.
func TestUnsettledReadsCodeSpansAsThePagePairsThem(t *testing.T) {
	const fm = "---\nid: spc-1\n---\n# T\n"
	cases := []struct {
		name string
		body string
		want []Statement // Line and Kind only
	}{
		{"the phrase inside a span over a line break", "Quote `x.\nNot READY: owed` here.\n", nil},
		{"the same phrase outside one", "Quote `x` y.\nNot READY: owed.\n", []Statement{{Line: 6, Kind: UnsettledNotReady}}},
		{"an unpaired backtick, then the phrase after a span's close", "The cache `x\ny`. Not READY: owed `z`.\n", []Statement{{Line: 6, Kind: UnsettledNotReady}}},
		{"a comment opener inside a span over a line break", "Intro `a <!-- b\nc` Not READY: owed. -->\n", []Statement{{Line: 6, Kind: UnsettledNotReady}}},
		{"a blank line between the two backticks", "Quote `x.\n\nNot READY: owed` here.\n", []Statement{{Line: 7, Kind: UnsettledNotReady}}},
		{"a heading between the two backticks", "Quote `x.\n## Not READY: owed` here.\n", []Statement{{Line: 6, Kind: UnsettledNotReady}}},
		{"a list item between the two backticks", "Quote `x.\n- Not READY: owed` here.\n", []Statement{{Line: 6, Kind: UnsettledNotReady}}},
		{"a fence between the two backticks", "Quote `x.\n```\ncode\n```\nNot READY: owed` here.\n", []Statement{{Line: 9, Kind: UnsettledNotReady}}},
		{"a table cell between the two backticks", "| a | b |\n|---|---|\n| `x | Not READY: owed` |\n", []Statement{{Line: 7, Kind: UnsettledNotReady}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Unsettled(fm + c.body)
			if len(got) != len(c.want) {
				t.Fatalf("Unsettled = %+v, want %+v", got, c.want)
			}
			for i := range got {
				if got[i].Line != c.want[i].Line || got[i].Kind != c.want[i].Kind {
					t.Fatalf("statement %d = %+v, want line %d %s", i, got[i], c.want[i].Line, c.want[i].Kind)
				}
			}
		})
	}
}

// TestOpenQuestionsReadUnderACommentOpenedAfterASpan: an unpaired backtick,
// then a line opening with a backtick and an HTML comment opener, the comment
// running over the Open Questions section: the page pairs the two backticks
// and shows the questions, and so does the reader. With the opener inside the
// span the page shows no comment at all, and the reader reports no layout.
func TestOpenQuestionsReadUnderACommentOpenedAfterASpan(t *testing.T) {
	const fm = "---\nid: spc-1\n---\n# T\n"
	for _, c := range []struct {
		name, body string
		layout     bool
	}{
		{"the opener after the span", "The cache ` is named\n` <!-- parked\n\n## Open Questions\n\n- Which cache?\n\n-->\n", true},
		{"the opener inside the span", "The cache `a <!-- parked\nb` named.\n\n## Open Questions\n\n- Which cache?\n\n-->\n", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			if q := OpenQuestions(fm + c.body); len(q) != 1 || q[0].Text != "Which cache?" {
				t.Fatalf("OpenQuestions = %+v, want the one question", q)
			}
			layout := false
			for _, st := range Unsettled(fm + c.body) {
				layout = layout || st.Kind == UnsettledLayout || st.Kind == UnsettledUnclosed
			}
			if layout != c.layout {
				t.Fatalf("Unsettled = %+v, want a layout statement %v", Unsettled(fm+c.body), c.layout)
			}
		})
	}
}
