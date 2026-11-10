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
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseDependsOn(tc.body)
			if err != nil {
				t.Fatalf("parseDependsOn: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("parseDependsOn = %#v, want %#v", got, tc.want)
			}
		})
	}
}

// padTo is head followed by plain text up to exactly n bytes.
func padTo(head string, n int) string { return head + strings.Repeat("x", n-len(head)) }

// TestParseDependsOnRefuses: any other line naming depends-on, in any case and
// anywhere on the line, is refused by its line number alone; so are more than
// maxDepends distinct numbers and a body over maxBodyBytes, never truncated.
func TestParseDependsOnRefuses(t *testing.T) {
	t.Parallel()
	if maxBodyBytes != 65536 || maxDepends != 10 {
		t.Fatalf("maxBodyBytes %d, maxDepends %d; want 65536 and 10", maxBodyBytes, maxDepends)
	}
	const shape = `coord: line %s names depends-on but is not a line of the form "Depends-On: #N, #N"`
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
			got, err := parseDependsOn(tc.body)
			if err == nil {
				t.Fatalf("parseDependsOn accepted the body and gave %#v", got)
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
		_, err := parseDependsOn(body)
		if err == nil {
			t.Fatalf("parseDependsOn accepted %q", body)
		}
		if strings.Contains(err.Error(), canary) || strings.Contains(err.Error(), "7f3a") {
			t.Fatalf("the error quotes the body: %q", err)
		}
	}
}
