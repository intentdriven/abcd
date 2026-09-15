package termsafe

import (
	"strings"
	"testing"
)

// TestCleanProseLeavesCodeSpansAlone: the HTML-opener neutralisation exists
// because a `<` can begin raw HTML in rendered markdown. Inside a CommonMark
// code span it cannot — the span's content is literal — so neutralising there
// buys nothing and corrupts the content: the v0.7.0 changelog's documented
// `--reading-json <path>` was written as `--reading-json < path>`, which is a
// shell input redirection from a file named `path` (iss-2609011217083577).
// Code spans are left byte-for-byte; every `<` outside one is still neutralised.
func TestCleanProseLeavesCodeSpansAlone(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"placeholder-in-span", "run `abcd reading ingest --reading-json <path>` first",
			"run `abcd reading ingest --reading-json <path>` first"},
		{"tag-in-span", "the literal `<script>` token", "the literal `<script>` token"},
		// The comment delimiters are the ONE part of the HTML rule that keeps
		// firing inside a span: they defend a byte-level marker grammar, not a
		// render (see TestCleanProseBreaksCommentDelimitersInsideACodeSpan).
		{"comment-in-span", "write `<!-- note -->` here", "write `< !-- note -- >` here"},
		{"double-backtick-span", "use `` a ` <b> `` here", "use `` a ` <b> `` here"},
		{"prose-around-span-still-neutralised", "<b>bold</b> and `<b>`",
			"< b>bold< /b> and `<b>`"},
		{"two-spans", "`<a>` then <a> then `<a>`", "`<a>` then < a> then `<a>`"},
		// An unbalanced backtick opens no span in CommonMark, so what follows it is
		// prose and stays neutralised — the primitive fails closed. The run itself
		// is backslash-escaped, which renders as the same literal backticks it
		// already rendered as and can no longer re-pair with a neighbouring field
		// (TestCleanProseEmitsNoUnpairedBacktickRun).
		{"unbalanced-backtick", "`<script>alert(1)", "\\`< script>alert(1)"},
		{"mismatched-run-lengths", "``<script>` still prose", "\\`\\`< script>\\` still prose"},
		// A backslash-escaped backtick is a literal backtick, not a delimiter.
		{"escaped-backtick", `\` + "`<script>" + `\` + "`", `\` + "`< script>" + `\` + "`"},
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

// TestCleanProseCapCannotExposeASpan: the cap is applied after neutralisation,
// so a cut landing inside a code span would leave an unbalanced backtick and
// turn the span's untouched content back into prose — with its openers live.
// A truncated result must therefore carry no opener outside a span, and must
// still respect the cap.
func TestCleanProseCapCannotExposeASpan(t *testing.T) {
	in := "`aaaa <script> bbbbb`"
	for capBytes := 1; capBytes <= len(in); capBytes++ {
		got := CleanProse(in, capBytes)
		if len(got) > capBytes {
			t.Errorf("cap %d: CleanProse = %q, %d bytes", capBytes, got, len(got))
		}
		if strings.Contains(got, "<s") && strings.Count(got, "`") != 2 {
			t.Errorf("cap %d: CleanProse = %q exposes an opener outside a span", capBytes, got)
		}
	}
}

// TestCleanProseNeutralisesLinkSyntaxInsideACodeSpan pins the asymmetry between
// the two rules the cleaner applies. The HTML rule takes a code-span exemption
// because a renderer parses no raw HTML inside a span, so neutralising there
// only corrupts the quoted content. The link rule takes no such exemption,
// because the thing it defends is not the render but record-lint's
// links_resolve gate — and checkLinks masks fenced blocks only, so a `](` left
// inside an inline span is scanned like any other prose and refuses the whole
// tree on a record the ingest itself just wrote (iss-2608311504353427). One
// string carries both cases, so exempting spans wholesale fails here.
func TestCleanProseNeutralisesLinkSyntaxInsideACodeSpan(t *testing.T) {
	const in = "quoting `items[0](itm-0001) <path>` verbatim"
	const want = "quoting `items[0] (itm-0001) <path>` verbatim"
	if got := CleanProse(in, 4096); got != want {
		t.Errorf("CleanProse(%q) = %q, want %q", in, got, want)
	}
}
