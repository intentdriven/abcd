package termsafe

import (
	"strings"
	"testing"
)

// TestCleanProseNeutralisesStructure proves the shared prose cleaner does the
// four things every untrusted-prose boundary needs: line breaks cannot forge
// structure, HTML comment markers cannot open or close a comment, terminal
// display attacks are masked, and the result is capped.
func TestCleanProseNeutralisesStructure(t *testing.T) {
	cases := []struct {
		name string
		in   string
		cap  int
		want string
	}{
		{"newline-to-space", "one\ntwo", 100, "one two"},
		{"carriage-return-to-space", "one\rtwo", 100, "one two"},
		{"comment-open-broken", "text <!-- hidden", 100, "text < !-- hidden"},
		{"comment-close-broken", "hidden --> text", 100, "hidden -- > text"},
		{"escape-masked", "red" + string(rune(0x1b)) + "[31m", 100, "red?[31m"},
		{"trimmed", "   padded   ", 100, "padded"},
		{"capped", strings.Repeat("a", 20), 8, "aaaaaaaa"},
		{"cap-trims-trailing-space", "abcdefg hijk", 8, "abcdefg"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := CleanProse(c.in, c.cap); got != c.want {
				t.Errorf("CleanProse(%q, %d) = %q, want %q", c.in, c.cap, got, c.want)
			}
		})
	}
}

// TestCleanProseNeutralisesRawHTML proves no prose field can open raw HTML in the
// record it lands in. In CommonMark a `<` followed by a letter, `/`, `!`, or `?`
// begins an HTML block, and several of those block types run to the end of the
// document — so one unclosed tag can make every later section of a durable record
// render as inert text while a forged section above it renders normally.
func TestCleanProseNeutralisesRawHTML(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"script-tag", "<script>alert(1)</script>", "< script>alert(1)< /script>"},
		{"table-tag", "<table><tr><td>forged", "< table>< tr>< td>forged"},
		{"declaration", "<!DOCTYPE html>", "< !DOCTYPE html>"},
		{"comment-open", "text <!-- hidden", "text < !-- hidden"},
		{"processing-instruction", "<?php echo", "< ?php echo"},
		{"closing-tag", "</div>", "< /div>"},
		// A masked control byte becomes '?', so a `<` before one would otherwise
		// FORM a processing instruction after sanitisation. Neutralising after
		// Sanitize is what closes this.
		{"escape-forms-instruction", "<" + string(rune(0x1b)) + "php", "< ?php"},
		// Ordinary prose keeps its angle brackets: these open nothing.
		{"comparison-kept", "a < b and c > d", "a < b and c > d"},
		{"arrow-kept", "x <- y", "x <- y"},
		{"trailing-kept", "ends with <", "ends with <"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := CleanProse(c.in, 200); got != c.want {
				t.Errorf("CleanProse(%q) = %q, want %q", c.in, got, c.want)
			}
			if got := CleanProseLine(c.in, 200); strings.Contains(got, "<script") ||
				strings.Contains(got, "<table") || strings.Contains(got, "<!") || strings.Contains(got, "<?") {
				t.Errorf("CleanProseLine(%q) = %q still opens raw HTML", c.in, got)
			}
		})
	}
}

// TestCleanProseKeepsInteriorRuns proves CleanProse leaves interior whitespace
// alone — it trims, it does not collapse. That is the difference from
// CleanProseLine, and a caller picking the wrong one would silently reflow prose.
func TestCleanProseKeepsInteriorRuns(t *testing.T) {
	if got, want := CleanProse("a  b\tc", 100), "a  b c"; got != want {
		t.Errorf("CleanProse kept-runs = %q, want %q", got, want)
	}
}

// TestCleanProseLineCollapsesWhitespace proves the line form collapses every
// whitespace run to one space, so prose landing in a line-structured file (a
// changelog bullet, a markdown table cell) occupies exactly one line.
func TestCleanProseLineCollapsesWhitespace(t *testing.T) {
	cases := []struct {
		name string
		in   string
		cap  int
		want string
	}{
		{"runs-collapsed", "a  b\tc", 100, "a b c"},
		{"newline-collapsed", "one\n\ntwo", 100, "one two"},
		{"comment-open-broken", "text <!-- hidden", 100, "text < !-- hidden"},
		{"capped-after-collapse", "aaaa  bbbb  cccc", 10, "aaaa bbbb"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := CleanProseLine(c.in, c.cap); got != c.want {
				t.Errorf("CleanProseLine(%q, %d) = %q, want %q", c.in, c.cap, got, c.want)
			}
		})
	}
}

// TestCleanProseTruncationStaysValidUTF8 proves a cap landing mid-rune does not
// leave a broken code point behind — the truncated tail is dropped, never emitted
// as replacement bytes.
func TestCleanProseTruncationStaysValidUTF8(t *testing.T) {
	// "é" is two bytes; a 3-byte cap over "aaé" lands mid-rune.
	got := CleanProse("aaé", 3)
	if got != "aa" {
		t.Errorf("CleanProse mid-rune cap = %q, want %q", got, "aa")
	}
	for _, r := range got {
		if r == '�' {
			t.Fatalf("CleanProse emitted a replacement rune: %q", got)
		}
	}
}

// TestCleanProseNeutralisesLinkSyntax proves no prose field can carry a live
// markdown link into the record it lands in (iss-2608311504353427). A faithful
// quotation of code such as `items[0](itm-0001)` is an inline link whose target
// resolves to nothing, and the links_resolve gate then refuses the whole tree on
// a record the ingest itself wrote. The neutralisation is the one the comment
// delimiters already get — a single space breaking the adjacency CommonMark
// requires — so the quoted text stays legible and the brackets themselves are
// left alone.
func TestCleanProseNeutralisesLinkSyntax(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"inline-link", "the path items[0](itm-0001) is quoted", "the path items[0] (itm-0001) is quoted"},
		{"inline-link-with-title", `[x](y "t")`, `[x] (y "t")`},
		{"image", "![alt](x.svg)", "![alt] (x.svg)"},
		{"full-reference-link", "see [text][label] here", "see [text] [label] here"},
		{"collapsed-reference-link", "see [label][] here", "see [label] [] here"},
		{"autolink", "<https://example.invalid/x>", "< https://example.invalid/x>"},
		{"email-autolink", "<someone@example.invalid>", "< someone@example.invalid>"},
		// Brackets and parentheses that open no link are untouched.
		{"index-kept", "items[0] and f(x)", "items[0] and f(x)"},
		{"shortcut-kept", "a [label] alone", "a [label] alone"},
		{"spaced-kept", "already [x] (y)", "already [x] (y)"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := CleanProse(c.in, 200); got != c.want {
				t.Errorf("CleanProse(%q) = %q, want %q", c.in, got, c.want)
			}
			if got := CleanProseLine(c.in, 200); got != c.want {
				t.Errorf("CleanProseLine(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// TestCleanProseIsIdempotent proves a second pass over cleaned prose changes
// nothing: every neutralisation inserts a form the same rule no longer matches,
// so a field that is re-ingested, re-rendered or re-cleaned by a downstream
// caller does not drift further from what was written.
func TestCleanProseIsIdempotent(t *testing.T) {
	inputs := []string{
		"items[0](itm-0001) and [t][l] and [l][] and <https://example.invalid>",
		"text <!-- hidden --> and <script>x</script>",
		"one\ntwo\r\nthree" + string(rune(0x1b)) + "[31m",
		"a  b\tc [d](e)",
	}
	for _, in := range inputs {
		once := CleanProse(in, 200)
		if twice := CleanProse(once, 200); twice != once {
			t.Errorf("CleanProse not idempotent on %q:\n once  %q\n twice %q", in, once, twice)
		}
		onceL := CleanProseLine(in, 200)
		if twiceL := CleanProseLine(onceL, 200); twiceL != onceL {
			t.Errorf("CleanProseLine not idempotent on %q:\n once  %q\n twice %q", in, onceL, twiceL)
		}
	}
}
