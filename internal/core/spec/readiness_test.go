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

// unsettledCase is a spec body and the statements Unsettled reports in it, by
// line and kind.
type unsettledCase struct {
	name string
	body string
	want []Statement // Line and Kind only
}

func checkUnsettled(t *testing.T, cases []unsettledCase) {
	t.Helper()
	const fm = "---\nid: spc-1\n---\n# T\n"
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Unsettled(fm + c.body)
			if len(got) != len(c.want) {
				t.Fatalf("Unsettled(%q) = %+v, want %+v", c.body, got, c.want)
			}
			for i := range got {
				if got[i].Line != c.want[i].Line || got[i].Kind != c.want[i].Kind {
					t.Fatalf("Unsettled(%q): statement %d = %+v, want line %d %s", c.body, i, got[i], c.want[i].Line, c.want[i].Kind)
				}
			}
		})
	}
}

// notReady is one not-READY statement on line n.
func notReady(n int) []Statement { return []Statement{{Line: n, Kind: UnsettledNotReady}} }

// TestUnsettledBlanksASpanOnlyWhereEveryReadingAgrees: a code span is blanked
// where it opens and closes on one line and no line above it in its stretch
// leaves a backtick run open, as when spans were paired a line at a time; a
// span over a line break, and every line below a run left open to the next
// blank line, is read.
func TestUnsettledBlanksASpanOnlyWhereEveryReadingAgrees(t *testing.T) {
	checkUnsettled(t, []unsettledCase{
		{"a span on one line", "Quote `x. Not READY: owed` here.\n", nil},
		{"a balanced line after a balanced line", "Quote `a` b.\nc `x. Not READY: owed` d.\n", nil},
		{"the phrase outside a span", "Quote `x` y.\nNot READY: owed.\n", notReady(6)},
		{"the phrase inside a span over a line break", "Quote `x.\nNot READY: owed` here.\n", notReady(6)},
		{"an unpaired backtick, then a line of one backtick and the phrase", "The cache `x\ny`. Not READY: owed.\n", notReady(6)},
		{"an unpaired backtick, then the phrase after a span's close", "The cache `x\ny`. Not READY: owed `z`.\n", notReady(6)},
		{"a comment opener inside a span over a line break", "Intro `a <!-- b\nc`. Not READY: owed. -->\n", []Statement{layoutAt(5), notReady(6)[0]}},
		{"a comment opener after the run on its line", "Intro `a <!-- b\nc `. Not READY: owed. -->\n", []Statement{layoutAt(5), notReady(6)[0]}},
		{"a blank line ends the stretch", "Quote `x.\n\nc `x. Not READY: owed` d.\n", nil},
		{"a line of spaces and tabs ends the stretch", "Quote `x.\n \t \nc `x. Not READY: owed` d.\n", nil},
		{"a line of other white space does not", "Quote `x.\n\u00a0\ny`. Not READY: owed `z`.\n", notReady(7)},
		{"a heading does not", "Quote `x.\n## h\nc `x. Not READY: owed` d.\n", notReady(7)},
		{"a list item does not", "Quote `x.\n- c `x. Not READY: owed` d.\n", notReady(6)},
		{"a fence does not", "Quote `x.\n```\ncode\n```\nc `x. Not READY: owed` d.\n", notReady(9)},
		{"a fence leaves a sure stretch sure", "Quote `x` y.\n```\ncode `\n```\nc `x. Not READY: owed` d.\n", nil},
		{"a run of two left open", "Quote ``x` y.\nc `x. Not READY: owed` d.\n", notReady(6)},
		{"spans either side of the run on its line", "`x. Not READY: a` `` `x. Not READY: b`\nc.\n", nil},
		{"an escaped backtick leaves nothing open", "Quote \\` y.\nc `x. Not READY: owed` d.\n", nil},
		{"an escaped backtick pairs with nothing", "Use \\` then. Not READY: owed `x`.\n", notReady(5)},
	})
}

// TestUnsettledReadsTheLayoutsTheReviewsNamed: in each layout the page shows
// the phrase as prose, and the reader reads it: a backtick run left open on
// the line above leaves every line below it, to the next blank line, read
// whole, whatever block the page reads there.
func TestUnsettledReadsTheLayoutsTheReviewsNamed(t *testing.T) {
	checkUnsettled(t, []unsettledCase{
		{"an indented code block after a blank line", "intro\n\n    a `b\n    Not READY: owed\n    c ` d\n", notReady(8)},
		{"an indented code block over three lines", "intro\n\n    a `b\n    Not READY: owed\n    more\n    c ` d\n", notReady(8)},
		{"a heading in a blockquote", "> # t`\nNot READY: owed\n` tail\n", notReady(6)},
		{"a heading in a list item", "- # t`\nNot READY: owed\n` tail\n", notReady(6)},
		{"a block start four columns in", "    - a `\nNot READY: owed\n` tail\n", notReady(6)},
		{"a link reference definition's title", "[a]: /u \"`\"\nNot READY: owed\n` tail\n", notReady(6)},
		{"a link reference definition's destination", "[a]: /u`x\nNot READY: owed `y`\n", notReady(6)},
		{"a table under a list item", "- h ` | i\n  --- | ---\n  Not READY: owed | x\n  ` | y\n", notReady(7)},
		{"a line of no-break spaces", "- a `x\n \ny` c\nNot READY: owed\nd ` e\n", notReady(8)},
		{"a line of em spaces", "- a `x\n \ny` c\nNot READY: owed\nd ` e\n", notReady(8)},
		{"a setext underline under a list item", "- a `x\n===\ny` c\nNot READY: owed\nd ` e\n", notReady(8)},
		{"a comment after a marker numbered 2", "- a `x\n2. <!-- c -->\ny` c\nNot READY: owed\nd ` e\n", notReady(8)},
		{"a marker of more than nine digits", "- a `x\n0000000001. x\ny` c\nNot READY: owed\nd ` e\n", notReady(8)},
		{"a processing instruction", "a <?p ` ?> b\nNot READY: owed\nc ` d\n", notReady(6)},
		{"a declaration", "a <!X ` > b\nNot READY: owed\nc ` d\n", notReady(6)},
		{"a CDATA section", "a <![CDATA[ ` ]]> b\nNot READY: owed\nc ` d\n", notReady(6)},
		{"an email autolink", "a <1`x@e.org> b\nNot READY: owed\nc ` d\n", notReady(6)},
		{"a tag left open on the line above", "text <a title=\"\n2. ` \">\nNot READY: owed\n` tail\n", notReady(7)},
		{"a tag whose quoted value holds a >", "<a title=\">`\">x\nNot READY: owed `y`\n", notReady(6)},
	})
}

// TestOpenQuestionsReadUnderACommentOpenedAfterASpan: an unpaired backtick,
// then a line holding a backtick and an HTML comment opener, the comment
// running over the Open Questions section: the questions are read, and the
// opener's line, from which the page may read the comment or none, is named.
func TestOpenQuestionsReadUnderACommentOpenedAfterASpan(t *testing.T) {
	const fm = "---\nid: spc-1\n---\n# T\n"
	for _, c := range []struct {
		name, body string
		line       int
	}{
		{"the opener after the second backtick", "The cache ` is named\n` <!-- parked\n\n## Open Questions\n\n- Which cache?\n\n-->\n", 6},
		{"the opener between the backticks", "The cache `a <!-- parked\nb` named.\n\n## Open Questions\n\n- Which cache?\n\n-->\n", 5},
		{"the opener on the line below the run", "The cache `a\nb <!-- parked\n\n## Open Questions\n\n- Which cache?\n\n-->\n", 6},
	} {
		t.Run(c.name, func(t *testing.T) {
			if q := OpenQuestions(fm + c.body); len(q) != 1 || q[0].Text != "Which cache?" {
				t.Fatalf("OpenQuestions = %+v, want the one question", q)
			}
			if st := Unsettled(fm + c.body); len(st) != 1 || st[0] != (Statement{Line: c.line, Kind: UnsettledLayout, Text: st[0].Text}) {
				t.Fatalf("Unsettled = %+v, want the layout named on line %d", st, c.line)
			}
		})
	}
}

// layoutAt is one layout statement on line n.
func layoutAt(n int) Statement { return Statement{Line: n, Kind: UnsettledLayout} }

// TestOpenQuestionsReadBelowALineInitialComment: below a backtick run a line
// leaves open, a comment opener that is the first thing on its line, at up to
// three columns, opens a comment whatever the lines above leave open: a
// heading inside it does not end the Open Questions section, and
// the question below it is counted.
func TestOpenQuestionsReadBelowALineInitialComment(t *testing.T) {
	const fm = "---\nid: spc-1\n---\n# T\n"
	for _, c := range []struct{ name, body string }{
		{"at the margin", "## Open Questions\n\nStray ` tick\n<!--\n## Parked\n-->\n\n- Which cache?\n"},
		{"three columns in", "## Open Questions\n\nStray ` tick\n   <!--\n## Parked\n-->\n\n- Which cache?\n"},
		{"on the run's own stretch, no blank line after", "## Open Questions\n\nStray ` tick\n<!--\n## Parked\n-->\n- Which cache?\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			if q := OpenQuestions(fm + c.body); len(q) != 1 || q[0].Text != "Which cache?" {
				t.Fatalf("OpenQuestions = %+v, want the one question", q)
			}
			if st := Unsettled(fm + c.body); len(st) != 0 {
				t.Fatalf("Unsettled = %+v, want nothing", st)
			}
		})
	}
}

// TestUnsettledReadsProseBetweenTwoLineInitialComments: below a backtick run
// a line leaves open, two comments that each open first on their line and
// each hold a fence delimiter mask those delimiters, so they pair nothing and
// the prose between the comments is read.
func TestUnsettledReadsProseBetweenTwoLineInitialComments(t *testing.T) {
	checkUnsettled(t, []unsettledCase{
		{"backtick fences", "Stray ` tick\n<!--\n```\n-->\nNot READY: owed.\n<!--\n```\n-->\n", notReady(9)},
		{"tilde fences", "Stray ` tick\n<!--\n~~~\n-->\nNot READY: owed.\n<!--\n~~~\n-->\n", notReady(9)},
	})
}

// TestUnsettledNamesAnOpenerBelowARunLeftOpen: below a backtick run a line
// leaves open, and after the run on its own line, a comment opener that is
// not the first thing on its line, or comes first after list or blockquote
// markers, may sit inside a span the page pairs over the line break; where
// no closer follows it on the line the page is read whole from that line,
// which is named.
func TestUnsettledNamesAnOpenerBelowARunLeftOpen(t *testing.T) {
	checkUnsettled(t, []unsettledCase{
		{"after other text", "Stray ` tick\nsee <!-- parked\n## Parked\n-->\n", []Statement{layoutAt(6)}},
		{"after a list marker", "Stray ` tick\n- <!-- parked\n## Parked\n-->\n", []Statement{layoutAt(6)}},
		{"after a blockquote marker", "Stray ` tick\n> <!-- parked\n> ## Parked\n> -->\n", []Statement{layoutAt(6)}},
		{"four columns in", "Stray ` tick\n    <!-- parked\n## Parked\n-->\n", []Statement{layoutAt(6)}},
		{"after the run on its line", "Stray ` tick <!-- parked\n## Parked\n-->\n", []Statement{layoutAt(5)}},
		{"inside a one-line span", "Stray ` tick\nsee `<!-- parked` here\n## Parked\n-->\n", []Statement{layoutAt(6)}},
		{"closed on its line, the text is read", "Stray ` tick\nsee <!-- c. Not READY: owed. --> here\n", notReady(6)},
		{"after a blank line, as before", "Stray ` tick\n\nsee <!-- parked\n-->\n", nil},
	})
}

// TestUnsettledPinsTheOlderLimits: what the reader does today in layouts it
// does not mirror, held so a change to any of them is seen. In each the
// reader does not report the phrase. In the backtick and backslash cases the
// page shows it as prose; a one-line comment whose text holds two hyphens the
// reader reads as a comment, so its text is not read.
func TestUnsettledPinsTheOlderLimits(t *testing.T) {
	checkUnsettled(t, []unsettledCase{
		{"a backtick in an inline link's destination", "See [a](/u`x). Not READY: owed `y`.\n", nil},
		{"a backtick in an inline link's title", "See [a](/u \"t`x\"). Not READY: owed `y`.\n", nil},
		{"a backtick in a processing instruction", "See <?p ` ?>. Not READY: owed ` here.\n", nil},
		{"a backtick in a declaration", "See <!X ` >. Not READY: owed ` here.\n", nil},
		{"a backtick in a CDATA section", "See <![CDATA[ ` ]]>. Not READY: owed ` here.\n", nil},
		{"a one-line comment holding two hyphens", "See <!-- a -- b. Not READY: owed. --> here.\n", nil},
		{"a backslash in an indented code block", "intro\n\n    \\`a` b. Not READY: x `c`\n", nil},
		{"a backslash in an HTML block", "<div>\n\\`a` b. Not READY: x `c`\n</div>\n", nil},
		{"a backslash in a tag", "See <a title=\"\\`x`\">. Not READY: owed `y`.\n", nil},
	})
}
