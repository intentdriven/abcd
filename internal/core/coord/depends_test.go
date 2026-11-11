package coord

import (
	"reflect"
	"strings"
	"testing"
)

// TestParseDependsOnAccepts: exact-case Depends-On lines are read, joined
// across lines, ascending and once each, nil for none, with one trailing
// carriage return stripped from each line.
func TestParseDependsOnAccepts(t *testing.T) {
	t.Parallel()
	ten := "Depends-On: #1, #2, #3, #4, #5, #6, #7, #8, #9, #10"
	cases := []struct {
		name string
		body string
		want []int
	}{
		{"an empty body", "", nil},
		{"a body with no dependency line", "Fixes the parser.\n\nRefs: iss-1\n", nil},
		{"one number", "Depends-On: #12", []int{12}},
		{"a tab, a list, and no space after a comma", "Depends-On:\t#3, #1,#2", []int{1, 2, 3}},
		{"spaces and tabs at the end", "Depends-On: #5 \t", []int{5}},
		{"lines joined, ascending and once each", "Depends-On: #9\nbody text\nDepends-On: #2, #9", []int{2, 9}},
		{"CRLF line ends", "Depends-On: #4\r\nDepends-On: #5\r\n", []int{4, 5}},
		{"a nine-digit number", "Depends-On: #999999999", []int{999999999}},
		{"ten numbers", ten, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
		{"eleven read, ten distinct", ten + "\nDepends-On: #10", []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
		{"a body of exactly the most bytes", padTo("Depends-On: #7\n", maxBodyBytes), []int{7}},
		{"a space before a comma and a tab after it", "Depends-On: #1 ,\t#2", []int{1, 2}},
		{"a well-formed line inside a code fence of several lines", "```\nDepends-On: #3\n```", []int{3}},
		{"a well-formed line inside an HTML comment of several lines", "<!--\nDepends-On: #3\n-->", []int{3}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseDependsOn(tc.body)
			if err != nil {
				t.Fatalf("ParseDependsOn: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("ParseDependsOn = %#v, want %#v", got, tc.want)
			}
		})
	}
}

// padTo is head followed by plain text up to exactly n bytes.
func padTo(head string, n int) string { return head + strings.Repeat("x", n-len(head)) }

// TestParseDependsOnRefuses: any other line that looks like a Depends-On line
// after scanForm, in any case and anywhere on the line, is refused by its line
// number alone, honest prose and paths included; so are more than maxDepends
// distinct numbers and a body over maxBodyBytes, never truncated.
func TestParseDependsOnRefuses(t *testing.T) {
	t.Parallel()
	if maxBodyBytes != 65536 || maxDepends != 10 {
		t.Fatalf("maxBodyBytes %d, maxDepends %d; want 65536 and 10", maxBodyBytes, maxDepends)
	}
	line := refusedLine
	cases := []struct {
		name string
		body string
		want string
	}{
		{"number zero", "Depends-On: #0", line("1")},
		{"a leading zero", "Depends-On: #01", line("1")},
		{"a ten-digit number", "Depends-On: #1234567890", line("1")},
		{"lower case", "depends-on: #1", line("1")},
		{"upper case", "DEPENDS-ON: #1", line("1")},
		{"indented", "  Depends-On: #1", line("1")},
		{"a list item", "- Depends-On: #1", line("1")},
		{"a quote", "> Depends-On: #1", line("1")},
		{"a code fence", "```Depends-On: #1```", line("1")},
		{"another repository's number", "Depends-On: owner/repo#1", line("1")},
		{"a URL", "see https://example.invalid/pull/depends-on/3", line("1")},
		{"mid-line", "this Depends-On: #4 too", line("1")},
		{"no hash", "Depends-On: 1", line("1")},
		{"no space after the colon", "Depends-On:#1", line("1")},
		{"no number", "Depends-On:", line("1")},
		{"two numbers with no comma", "Depends-On: #1 #2", line("1")},
		{"a trailing comma", "Depends-On: #1,", line("1")},
		{"two carriage returns", "Depends-On: #1\r\r\n", line("1")},
		{"a lone carriage return between two lines", "Depends-On: #1\rDepends-On: #2", line("1")},
		{"a byte-order mark before the line", "\ufeffDepends-On: #5", line("1")},
		{"a NUL at the end of the line", "Depends-On: #5\x00", line("1")},
		{"a no-break space after the colon", "Depends-On:\u00a0#5", line("1")},
		{"a template's placeholder", "<!-- Depends-On: #N -->", line("1")},
		{"a space for the hyphen", "Depends On: #5", line("1")},
		{"a space for the hyphen, lower-case on", "Depends on: #5", line("1")},
		{"spaces and a tab before the colon", "Depends on \t : #5", line("1")},
		{"a tab for the hyphen", "Depends\tOn: #5", line("1")},
		{"tabs around a hyphen", "Depends\t-\tOn: #5", line("1")},
		{"an underscore for the hyphen", "Depends_On: #5", line("1")},
		{"no hyphen", "DependsOn: #5", line("1")},
		{"no hyphen, lower-case on", "Dependson: #5", line("1")},
		{"depend, with no s", "depend-on: #5", line("1")},
		{"depend on, with no s", "Depend on: #5", line("1")},
		{"a Unicode hyphen", "Depends\u2010On: #5", line("1")},
		{"a non-breaking hyphen", "Depends\u2011On: #5", line("1")},
		{"an en dash", "Depends\u2013On: #5", line("1")},
		{"an em dash and spaces", "Depends \u2014 On: #5", line("1")},
		{"a zero-width space inside the word", "Depen\u200bds-On: #5", line("1")},
		{"a zero-width joiner after the hyphen", "Depends-\u200dOn: #5", line("1")},
		{"a word joiner inside the word", "Depends-O\u2060n: #5", line("1")},
		{"a soft hyphen for the hyphen", "Depends\u00adOn: #5", line("1")},
		{"a soft hyphen inside the word", "De\u00adpends-On: #5", line("1")},
		{"a byte-order mark inside the word", "Depends\ufeff-On: #5", line("1")},
		{"a NUL inside the word", "Depe\x00nds-On: #5", line("1")},
		{"a markdown escape of the hyphen", "Depends\\-On: #5", line("1")},
		{"a markdown escape of the hash", "Depends-On: \\#5", line("1")},
		{"a combining grapheme joiner after the hyphen", "Depends-\u034fOn: #5", line("1")},
		{"a variation selector inside the word", "Depen\ufe0fds-On: #5", line("1")},
		{"a Mongolian free variation selector inside the word", "Depends-O\u180bn: #5", line("1")},
		{"an enclosing mark inside the word", "Depends\u20dd-On: #5", line("1")},
		{"a Hangul choseong filler inside the word", "Dep\u115fends-On: #5", line("1")},
		{"a Hangul jungseong filler after the hyphen", "Depends-\u1160On: #5", line("1")},
		{"a Hangul filler inside the word", "Depends\u3164-On: #5", line("1")},
		{"a minus sign for the hyphen", "Depends\u2212On: #5", line("1")},
		{"a hyphen bullet for the hyphen", "Depends\u2043On: #5", line("1")},
		{"honest prose with a colon after depends on", "The order depends on: the queue", line("1")},
		{"a quoted compose key", "Set `depends_on:` for the service", line("1")},
		{"a compose key in a YAML snippet", "    depends_on:", line("1")},
		{"a path holding depends-on", "see docs/depends-on.md", line("1")},
		{"a word beginning depends-on", "this depends-only on the flag", line("1")},
		{"a Windows path, its backslashes dropped", "see C:\\depends\\on: here", line("1")},
		{"a decomposed accent, its combining mark dropped", "Cela de\u0301pend on: the queue", line("1")},
		{"a long s for the s, no colon", "Depend\u017f-On #5", line("1")},
		{"a long s for the s, lower case, a space and a colon", "depend\u017f on: x", line("1")},
		{"the line number is the first bad line's, counted from one",
			"Title\n\nDepends-On: #2\nmore\n  depends-on #3\ndepends-on #4", line("5")},
		{"eleven distinct numbers", "Depends-On: #1, #2, #3, #4, #5, #6, #7, #8, #9, #10\nDepends-On: #11",
			"coord: the body names 11 dependencies, above the most of 10"},
		{"a body one byte over the most", padTo("Depends-On: #7\n", maxBodyBytes+1),
			"coord: the body is 65537 bytes, above the most of 65536"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseDependsOn(tc.body)
			if err == nil {
				t.Fatalf("ParseDependsOn accepted the body and gave %#v", got)
			}
			if err.Error() != tc.want {
				t.Fatalf("error\n got  %q\n want %q", err, tc.want)
			}
			if got != nil {
				t.Fatalf("a refusal also gave %#v", got)
			}
		})
	}
}

// refusedLine is the refusal of the 1-based line n that holds the words of a
// dependency line in a wrong shape.
func refusedLine(n string) string {
	return "coord: line " + n + ` holds the words of a dependency line but is not one in the accepted shape, which is "Depends-On: #<number>" alone on a line, more numbers separated by commas; prose in the body must not use those words, so reword that sentence, or write the line in that shape`
}

// TestParseDependsOnRefusalSaysHowToReword: the refusal names the line by its
// number, gives the accepted shape, and tells the author that prose must not
// use the words, so the sentence is reworded.
func TestParseDependsOnRefusalSaysHowToReword(t *testing.T) {
	t.Parallel()
	_, err := ParseDependsOn("Fixes the parser.\n\nThe order depends on: the queue")
	if err == nil {
		t.Fatal("ParseDependsOn read a mention in prose past")
	}
	for _, want := range []string{"line 3 ", `"Depends-On: #<number>" alone on a line`, "prose in the body must not use those words", "reword that sentence"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("refusal %q\n does not say %q", err, want)
		}
	}
}

// TestParseDependsOnRefusesMentions: each row holds the words of a dependency
// line and is refused by its line number, a mention in prose included. The
// rows guard against a later loosening that would let some of them through:
// each is a shape a reader can take for a dependency, or one a rule telling
// prose from a declaration would have to judge.
func TestParseDependsOnRefusesMentions(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ name, body string }{
		{"a line beginning with a word that begins depends-on", "depends-only on the flag"},
		{"prose that begins with depends on and a colon", "Depends on: the queue"},
		{"a decomposed accent at the line's start, its combining mark dropped", "De\u0301pend on: the queue"},
		{"a task-list item", "- [ ] Depends-On: #1"},
		{"a checked task-list item", "* [x] Depends-On: #1"},
		{"a numbered list item", "1. Depends-On: #1"},
		{"a heading", "## Depends-On: #1"},
		{"a table cell", "| Depends-On: #1 |"},
		{"bold", "**Depends-On:** #1"},
		{"inline code", "`Depends-On: #1`"},
		{"a quoted list item", "> - depends on: #1"},
		{"a no-break space before the line", "\u00a0Depends-On: #1"},
		{"an ideographic space before the line", "\u3000Depends-On: #1"},
		{"the trailer mid-line", "this Depends-On: #4 too"},
		{"a literal backslash-n before the line", "Summary\\n\\nDepends-On: #5"},
		{"a literal backslash-r-backslash-n before the line", "Summary\\r\\nDepends-On: #5"},
		{"two trailers joined on one line", "Refs: #3 Depends-On: #5"},
		{"a lone carriage return for a line end", "Title\rDepends-On: #5"},
		{"a carriage return before the line", "\rDepends-On: #5"},
		{"a vertical tab before the line", "\vDepends-On: #5"},
		{"a form feed before the line", "\fDepends-On: #5"},
		{"a vertical tab for a line end", "Title\vdepends on: the queue"},
		{"a form feed for a line end", "Title\fdepends on: the queue"},
		{"a next-line character for a line end", "Title\u0085depends on: the queue"},
		{"a line separator for a line end", "Title\u2028depends on: the queue"},
		{"a paragraph separator for a line end", "Title\u2029depends on: the queue"},
		{"a next-line character before the line", "\u0085Depends-On: #5"},
		{"a paragraph tag", "<p>Depends-On: #5</p>"},
		{"a bold tag", "<b>Depends-On:</b> #5"},
		{"a list item tag", "<li>Depends-On: #5</li>"},
		{"a code tag", "<code>Depends-On: #5</code>"},
		{"a line break tag", "<br>Depends-On: #5"},
		{"a details summary", "<details><summary>Depends-On: #5"},
		{"a blockquote tag", "<blockquote>Depends-On: #5"},
		{"a table data tag", "<td>Depends-On: #5</td>"},
		{"a tag with an attribute", "<p class=\"note\">depends on: the queue"},
		{"a tag with a closing mark inside an attribute", "<a title=\"x>y\">Depends-On: the parser PR"},
		{"a quote entity", "&gt; Depends-On: #5"},
		{"a no-break space entity", "&nbsp;Depends-On: #5"},
		{"a numeric entity", "&#32;Depends-On: #5"},
		{"a hexadecimal entity", "&#x20;depends on: the queue"},
		{"a named entity with no semicolon", "&nbsp depends on: the queue"},
		{"parentheses", "(Depends-On: #5)"},
		{"a link label", "[Depends-On: #5](https://example.invalid/5)"},
		{"double quotes", "\"Depends-On: #5\""},
		{"single quotes", "'Depends-On: #5'"},
		{"curly quotes", "\u201cDepends-On: #5\u201d"},
		{"braces", "{Depends-On: #5}"},
		{"a bullet", "\u2022 Depends-On: #5"},
		{"a middle dot", "\u00b7 Depends-On: #5"},
		{"an arrow", "\u2192 Depends-On: #5"},
		{"an ASCII arrow", "=> Depends-On: #5"},
		{"an en dash bullet", "\u2013 Depends-On: #5"},
		{"an em dash bullet", "\u2014 Depends-On: #5"},
		{"a small square bullet", "\u25aa Depends-On: #5"},
		{"a definition-list mark", ": Depends-On: #5"},
		{"a lettered item with a full stop", "a. Depends-On: #5"},
		{"a lettered item with a parenthesis", "a) depends on: the queue"},
		{"a roman item with a full stop", "iv. depends on: the queue"},
		{"a roman item with a parenthesis", "i) Depends-On: #5"},
		{"a number in parentheses", "(1) Depends-On: #5"},
		{"a lettered item in parentheses", "(b) depends on: the queue"},
		{"a section number", "3.2 depends on: the queue"},
		{"a numbered and lettered item", "1a. Depends-On: PR 5"},
		{"a dashed task box", "[-] Depends-On: #5"},
		{"a ticked task box", "[\u2713] Depends-On: #5"},
		{"a checked task box with prose after the words", "- [x] depends on: the queue"},
		{"an emoji", "\U0001f517 Depends-On: #5"},
		{"an emoji with a variation selector", "\u26a0\ufe0f Depends-On: #5"},
		{"a warning sign", "\u26a0 Depends-On: #5"},
		{"an emoji shortcode", ":link: depends on: the queue"},
		{"a halfwidth Hangul filler before the line", "\uffa0Depends-On: #5"},
		{"a braille blank before the line", "\u2800Depends-On: #5"},
		{"a start-of-heading control before the line", "\x01Depends-On: #5"},
		{"an escape control before the line", "\x1bdepends on: the queue"},
		{"a record separator for a line end", "Title\x1edepends on: the queue"},
		{"a label", "Note: Depends-On: #5"},
		{"a to-do label", "TODO: Depends-On: #5"},
		{"a bold label", "**Note:** Depends-On: #5"},
		{"a bold stacked label", "**Stacked:** Depends-On: #5"},
		{"a quoted bold label", "> **Blocked.** Depends-On: #5"},
		{"a later table cell", "| Stack | Depends-On: #5 |"},
		{"a later table cell, another heading", "| Dependency | Depends-On: #5 |"},
		{"a placeholder after prose", "Note: Depends-On: #N"},
		{"a template placeholder after a label", "<!-- template: Depends-On: N -->"},
		{"a to-do inside a comment", "<!-- TODO Depends-On: the parser PR -->"},
		{"a fullwidth number sign after prose", "see Depends-On: \uff035"},
		{"a small number sign after prose", "see Depends-On: \ufe5f5"},
		{"a bare number after prose and a colon", "Note: Depends-On: 5"},
		{"a bold number after prose and a colon", "Note: depends on: **5**"},
		{"a number after the words with no colon", "depends-on 5"},
		{"a URL after prose", "This PR Depends-On: https://example.invalid/o/r/pull/5"},
		{"a URL after a label and the spaced words", "Stacked: depends on: https://example.invalid/o/r/pull/5"},
		{"a URL after a line break tag", "Intro<br>Depends-On: https://example.invalid/o/r/pull/5"},
		{"GH- and a number", "Note: Depends-On: GH-5"},
		{"PR, a space and a number", "Note: Depends-On: PR 5"},
		{"PR and a number", "Note: Depends-On: PR5"},
		{"pr- and a number", "Stacked: Depends-On: pr-5"},
		{"pull request and a number", "Stacked: Depends-On: pull request 5"},
		{"pull/ and a number", "Note: Depends-On: pull/5"},
		{"no. and a number", "Note: Depends-On: no. 5"},
		{"a numero sign", "Note: Depends-On: \u21165"},
		{"a pull request named in words", "This PR Depends-On: the parser PR"},
		{"a number named in words", "Blocked. Depends-On: five"},
		{"a number in parentheses after the words", "Note: Depends-On: (5)"},
		{"a number in brackets", "Note: Depends-On: [5]"},
		{"a number in quotes", "Note: Depends-On: \"5\""},
		{"a number after a hyphen", "Note: Depends-On: -5"},
		{"a number after an equals sign", "Note: Depends-On: =5"},
		{"a number after a full stop", "Note: Depends-On: .5"},
		{"a number in a tag", "Note: Depends-On: <b>5</b>"},
		{"a number in angle brackets", "Note: Depends-On: <5>"},
		{"a percent-encoded number sign", "Note: Depends-On: %235"},
		{"a second colon", "Note: Depends-On:: 5"},
		{"a dash for the colon", "Note: Depends-On - 5"},
		{"an equals sign for the colon", "Note: Depends-On = 5"},
		{"an arrow for the colon", "Note: Depends-On \u2192 5"},
		{"no colon and an equals sign", "Note: depends-on = 5"},
		{"no colon and a dash", "Note: depends-on - 5"},
		{"no colon and parentheses", "Note: depends-on (5)"},
		{"a circled digit", "Note: Depends-On: \u2464"},
		{"a superscript digit", "Note: Depends-On: \u2075"},
		{"a roman numeral", "Note: Depends-On: \u2164"},
		{"a named entity for the number sign", "Note: Depends-On: &num;5"},
		{"a decimal entity for the number sign", "Note: Depends-On: &#35;5"},
		{"a hexadecimal entity for the number sign", "Note: Depends-On: &#x23;5"},
		{"a no-break space entity before the number", "Note: Depends-On:&nbsp;5"},
		{"a music sharp sign", "Note: Depends-On: \u266f5"},
		{"a viewdata square", "Note: Depends-On: \u23175"},
		{"the number on the next line", "Note: Depends-On:\n#5"},
		{"the number after a lone carriage return", "Note: Depends-On:\r#5"},
		{"the number after a line separator", "Note: Depends-On:\u2028#5"},
		{"the number after a literal backslash-n", "Note: Depends-On:\\n#5"},
		{"two mentions on a line", "See the Depends-On docs. Depends-On: 5"},
		{"two mentions in table cells", "see depends-on docs | Depends-On: 5"},
		{"a backslash and n inside the first word", "Depe\\nds-On: #5"},
		{"a backslash and n inside the second word", "Depends-O\\n: #5"},
		{"prose naming the trailer", "land reads a Depends-On: line only where it starts the line"},
		{"prose naming the trailer with no colon", "the Depends-On trailer names a pull request"},
		{"prose naming the trailer in code", "see how `Depends-On:` lines are read"},
		{"a label before prose", "Note: depends on: the queue"},
		{"a later table cell of prose", "| Order | depends on: the queue |"},
		{"a URL holding depends-on", "see https://example.invalid/pull/depends-on/3"},
		{"a path holding depends-on", "see docs/depends-on.md"},
		{"a quoted compose key mid-line", "Set `depends_on:` for the service"},
		{"a word beginning depends-on mid-line", "this depends-only on the flag"},
		{"a Windows path, its backslashes dropped", "see C:\\depends\\on: here"},
		{"a decomposed accent mid-line, its combining mark dropped", "Cela de\u0301pend on: the queue"},
	} {
		got, err := ParseDependsOn(c.body)
		if err == nil || err.Error() != refusedLine("1") || got != nil {
			t.Errorf("%s: ParseDependsOn = %#v, %v\n want the refusal of line 1", c.name, got, err)
		}
	}
}

// TestParseDependsOnReadsPastToday: the shapes below hold no dependency
// line's words as the scan reads them, so they are read past, with no
// dependency and no error, as ParseDependsOn's comment says: an HTML entity,
// letters from other scripts that look like Latin ones, fullwidth letters, a
// form with no colon whose words are not depends-on with an ASCII hyphen (a
// space, a spaced hyphen, an underscore, a non-breaking hyphen U+2011 or an
// en dash between them), a bare "Depends:", other words for a dependency,
// another word or a space other than an ASCII space or tab between the words,
// a number or a non-ASCII space between on and the colon, an invisible
// character outside the stripped set, and the word split by a newline, a lone
// carriage return, U+2028, U+0085 or an invalid byte. The table records these
// and does not endorse them: each is pinned so that widening the scan changes
// this table with the comment.
func TestParseDependsOnReadsPastToday(t *testing.T) {
	t.Parallel()
	for name, body := range map[string]string{
		"an HTML entity for the hyphen":             "Depends&#45;On: #5",
		"a Cyrillic e":                              "D\u0435pends-On: #5",
		"fullwidth letters":                         "\uff24\uff45\uff50\uff45\uff4e\uff44\uff53-\uff2f\uff4e: #5",
		"no colon":                                  "Depends on #5",
		"no colon after prose":                      "This depends on PR #5",
		"blocked by":                                "Blocked by #5",
		"requires":                                  "Requires #5",
		"Depends with no on":                        "Depends: #5",
		"a no-break space between words":            "Depends\u00a0On: #5",
		"the word split across two lines":           "Depends-\nOn: #5",
		"depend-on in prose with no colon":          "this does not depend-on #5",
		"a halfwidth Hangul filler inside the word": "Depends\uffa0-On: #5",
		"a no-break space between on and the colon": "Depends On\u00a0: #5",
		"a line separator splitting the word":       "Depends\u2028On: #5",
		"a next-line character splitting the word":  "Depe\u0085nds-On: #5",
		"a lone carriage return splitting the word": "Depends\rOn: #5",
		"an invalid byte splitting the word":        "Depe\xffnds-On: #5",
		"a precomposed accent":                      "Cela dépend on: the queue",
		"a number between on and the colon":         "This depends on #5: it needs the schema",
		"a word between depends and on":             "It depends heavily on: the queue",
		"a non-breaking hyphen with no colon":       "Depends\u2011On #5",
		"an en dash with no colon":                  "Depends\u2013On #5",
		"a spaced ASCII hyphen with no colon":       "Depends - On #5",
		"an underscore with no colon":               "depends_on #5",
		"a spacing mark inside the word":            "Depe\u0903nds-On: #5",
	} {
		got, err := ParseDependsOn(body)
		if err != nil || got != nil {
			t.Errorf("%s: ParseDependsOn = %#v, %v; want nil, nil", name, got, err)
		}
	}
}

// TestParseDependsOnNeverQuotesTheBody: a refused line's text never reaches
// the error, only its number.
func TestParseDependsOnNeverQuotesTheBody(t *testing.T) {
	t.Parallel()
	const canary = "CANARY-7f3a"
	for _, body := range []string{
		"Depends-On: #1 " + canary,
		canary + " depends-on",
		"Depends-On: #" + canary,
	} {
		_, err := ParseDependsOn(body)
		if err == nil {
			t.Fatalf("ParseDependsOn accepted %q", body)
		}
		if strings.Contains(err.Error(), canary) || strings.Contains(err.Error(), "7f3a") {
			t.Fatalf("the error quotes the body: %q", err)
		}
	}
}
