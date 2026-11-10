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
	const shape = `coord: line %s looks like a Depends-On line but is not in the accepted form "Depends-On: #N, #N"; reword that line, or write it in that form`
	line := func(n string) string { return strings.Replace(shape, "%s", n, 1) }
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

// TestParseDependsOnReadsPast: what the scan does not decode is read past,
// with no dependency and no error, as ParseDependsOn's comment says: an HTML
// entity, letters from other scripts that look like Latin ones, fullwidth
// letters, a form with no colon, a bare "Depends:", a space other than an ASCII
// space or tab between the words or between on and the colon, an invisible
// character outside the stripped set, and the word split by a newline, a lone
// carriage return, U+2028, U+0085 or an invalid byte. Each is pinned so that
// widening the scan changes this table with the comment.
func TestParseDependsOnReadsPast(t *testing.T) {
	t.Parallel()
	for name, body := range map[string]string{
		"an HTML entity for the hyphen":             "Depends&#45;On: #5",
		"a Cyrillic e":                              "D\u0435pends-On: #5",
		"fullwidth letters":                         "\uff24\uff45\uff50\uff45\uff4e\uff44\uff53-\uff2f\uff4e: #5",
		"no colon":                                  "Depends on #5",
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
	} {
		got, err := ParseDependsOn(body)
		if err != nil || got != nil {
			t.Fatalf("%s: ParseDependsOn = %#v, %v; want nil, nil", name, got, err)
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
