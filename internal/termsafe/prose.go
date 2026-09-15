package termsafe

// prose.go is the shared home for the untrusted-prose cleaner every host-delegated
// ingest boundary needs before a model-supplied string is written into a durable
// markdown or JSON record.
//
// It sits beside Sanitize because it IS Sanitize plus the two neutralisations a
// terminal sanitiser has no reason to make, but a FILE writer always does:
//
//   - line breaks become spaces, so one prose field can never forge a second line
//     (a changelog bullet, a markdown table row, a list item) in the record;
//   - anything that could OPEN raw HTML is broken apart — a tag, a closing tag, a
//     declaration, a processing instruction, or a comment — so prose can neither
//     open nor close an HTML construct and swallow the record around it;
//   - markdown link syntax is broken apart — an inline link `[t](d)`, an image,
//     and a reference link `[t][l]` — so a faithful quotation of code can never
//     land as a live link in a committed record (an autolink `<https://…>` is
//     already caught by the HTML rule, since its scheme starts with a letter).
//
// The HTML rule is not cosmetic. In CommonMark a `<` followed by a letter, `/`,
// `!`, or `?` at the start of a line begins an HTML BLOCK, and several of those
// block types run to the end of the document if their closing condition never
// arrives: one `<script>` in a claim makes every later section of a verdict record
// — the falsified claims, the grill hits, the adversary's findings — render as
// inert text inside an unclosed element, while a forged table above it renders
// normally. An artefact whose whole value is that a later session trusts it must
// not be able to hide its own evidence.
//
// The rule fires ANYWHERE in the string, not only at a line start, and that is
// deliberate rather than an over-reach: the same swallow was demonstrated from
// INSIDE a markdown table cell, where the opener is mid-line, and the cleaner
// cannot know what surrounds the field it is handed. The one place the TAG half
// of it does not fire is inside a CommonMark code span, where no HTML is parsed
// and the neutralisation only corrupts: a documented `--reading-json <path>` was
// written as `< path>`, a shell input redirection (iss-2609011217083577). Span
// boundaries follow the spec — a run of N backticks opens a span closed by the
// next run of exactly N, a backslash-escaped backtick is literal, and a run
// with no closer is literal too. The remaining cost is a fidelity regression on
// angle-bracket prose OUTSIDE a span — a bare `<repo>` placeholder reads
// `< repo>` — which is accepted: a slightly uglier placeholder is cheaper than a
// record that can conceal its own contents.
//
// The HTML-COMMENT delimiters take NO such exemption: `<!` and `-->` are broken
// apart everywhere, span or not. They are the same asymmetry the link rule has
// below — the tag rule defends a RENDER, and a renderer parses no HTML inside a
// span, while the comment delimiters defend a GATE that does not read CommonMark
// at all. The intent audit parks its review state as
// `<!-- abcd-review: <STATE> receipt=<rcp> -->` lines and finds them with a plain
// regex over the record's bytes, so backticks around a marker mean nothing to it:
// exempting spans let an untrusted verdict field write a WORKING marker into a
// committed intent record, claiming another intent's outstanding receipt was
// already INGESTED and turning that receipt's genuine review into a silent
// no-op. A gate that cannot see a code span cannot be given a code-span
// exemption. (That matcher is now line-anchored too — second defence, not a
// substitute for this one: it is still a byte pattern and not a grammar.)
//
// THE INVARIANT THE EXEMPTION RESTS ON: a cleaned field is parsed as CommonMark
// as the exact string it was cleaned as. The cleaner decides what is sheltered
// from the FIELD's span structure, but the shelter is only real if the RENDERED
// line draws the same boundaries — and the field is never rendered alone. Two
// things broke it, both closed here:
//
//   - An UNPAIRED backtick run. Two independently cleaned fields land adjacent on
//     one line (the audit's `%s:%s@%s`, a bullet's `- **%s** — %s`), and a stray
//     run in the first re-pairs with the opening run of a genuine span in the
//     second: the boundary moves and the content the cleaner judged sheltered is
//     prose again with its openers live. So an unpaired run is backslash-escaped.
//     That is the CommonMark-faithful choice of the two available: an unclosed run
//     ALREADY renders as literal backticks, and `\`` renders as the same literal
//     backtick, so the reader sees what the payload wrote — whereas synthesising a
//     closer would turn text the author wrote as prose into code. An escaped
//     backtick is also no longer a delimiter, so it can pair with nothing.
//   - A caller re-escaping the value it was handed. ideate's blockText escaped a
//     leading backtick unconditionally, killing the span the exemption relied on
//     and republishing a sheltered `<details>` as live markup. Its rule now
//     escapes only an UNBALANCED leading run, which is the only one that opens a
//     block: a backtick fence's info string may not contain backticks, so a
//     leading run with a matching closer on the same line is an inline span by
//     construction.
//
// The link rule is not a spoofing defence but a gate one (iss-2608311504353427):
// an auditor quoting an element path of the form `items[0](itm-0001)` writes an
// inline link whose target resolves to nothing, and record-lint's links_resolve
// rule then refuses the whole tree on a record the ingest itself just wrote — a
// failure no care in the author can prevent, because the offending text is a
// faithful quotation of the code under review. The neutralisation is the one the
// comment delimiters get, a single space breaking the adjacency CommonMark
// requires, so `items[0] (itm-0001)` still reads as the code it quotes. Only the
// adjacency is touched: a bracket or a parenthesis on its own is left alone, and
// a shortcut reference `[label]` is not neutralised, because doing so would have
// to rewrite every bracket in every field and it opens no link unless a matching
// definition exists in the surrounding document.
//
// The link rule does NOT take the code-span exemption the HTML TAG rule takes,
// and the asymmetry follows from what each one defends. The tag rule defends the
// RENDER, and a renderer parses no raw HTML inside a code span, so neutralising
// there buys nothing and corrupts the quoted content. The link rule defends a
// GATE, and the gate does not read the same grammar: record-lint's `checkLinks`
// masks fenced blocks only, so an inline span is scanned like any other prose
// and a `](` left inside one still refuses the whole tree. Until the gate
// exempts spans, the cleaner cannot.
//
// Every rule inserts a form the same rule no longer matches, so the cleaner is
// idempotent: a field cleaned twice is the field cleaned once, and a downstream
// caller that re-cleans what it was handed does not drift it further.
//
// Two forms exist because two callers legitimately want different whitespace
// handling, and the difference is visible in the record: CleanProse trims,
// CleanProseLine collapses every run to one space. A field landing in a
// line-structured file (a bullet, a table cell) wants the line form.
//
// This is the canonical home: internal/core/lifeboat and internal/core/release
// route through it rather than keep divergent copies, and a new trust boundary
// (internal/core/ideate) reuses it instead of writing a third.

import (
	"regexp"
	"strings"
)

// htmlTagRe matches the ways CommonMark lets a `<` begin raw HTML that a code
// span makes inert: a tag, a closing tag, or a processing instruction (`<?`). A
// bare `<` with anything else after it — `a < b`, `<-`, a trailing one — opens
// nothing and is left alone, so ordinary prose reads normally. The declaration
// and comment opener `<!` is deliberately NOT here: it belongs to
// neutraliseCommentDelimiters, which takes no code-span exemption.
var htmlTagRe = regexp.MustCompile(`<[A-Za-z/?]`)

// CleanProse neutralises one untrusted prose field and caps it at capBytes,
// preserving interior whitespace runs. The cap is applied last and the result is
// re-trimmed, so a cut landing mid-word leaves no dangling space; a cut landing
// mid-rune drops the partial rune rather than emitting replacement bytes.
func CleanProse(s string, capBytes int) string {
	return cleanProse(s, capBytes, strings.TrimSpace)
}

// CleanProseLine is CleanProse for a field that must occupy exactly one line:
// every whitespace run collapses to a single space before the cap is applied. Use
// it wherever the prose lands in a file whose line structure is machine-read.
func CleanProseLine(s string, capBytes int) string {
	return cleanProse(s, capBytes, func(v string) string { return strings.Join(strings.Fields(v), " ") })
}

// cleanProse is the shared body: neutralise, sanitise, normalise whitespace the
// caller's way, then cap. Written once so the two forms can only differ in the
// one dimension they are meant to differ in.
// The HTML neutralisation runs AFTER Sanitize, and the order is load-bearing:
// Sanitize SUBSTITUTES a '?' for each masked rune rather than deleting it, so a
// `<` followed by an escape byte becomes `<?` — a processing instruction, which
// is an HTML block that runs until `?>`. Neutralising before Sanitize would
// therefore let a masked control character forge the very construct this closes.
func cleanProse(s string, capBytes int, normalise func(string) string) string {
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = Sanitize(s)
	s = neutraliseSpanAware(s)
	// The link rule fires everywhere, code spans included, and that asymmetry
	// with the HTML rule is the difference between the two defences. The HTML
	// rule protects the RENDER, and a renderer parses no HTML inside a span, so
	// firing there would corrupt content for nothing. The link rule protects a
	// GATE, and record-lint's checkLinks masks fenced blocks only — an inline
	// span is scanned like any other prose — so a `](` left inside a span still
	// refuses the tree. A space after the `]` is enough: CommonMark requires the
	// destination `(` or the label `[` to follow the link text immediately, and
	// the links_resolve pattern requires the same adjacency.
	s = strings.ReplaceAll(s, "](", "] (")
	s = strings.ReplaceAll(s, "][", "] [")
	s = normalise(s)
	// The cap is applied last, and a cut can land inside a code span: the closing
	// backticks go, the span stops being one, and its untouched content is prose
	// again with its openers live — and its opening run is now unpaired. So every
	// cut is followed by another span-aware pass, and the pair repeats until the
	// result fits: the pass is idempotent and each cut retains strictly fewer of
	// the original bytes, so it ends. A cut can also land between a backslash and
	// the backtick it escapes, leaving a dangling escape that would consume
	// whatever the embedding puts next; the half-escape is dropped before the
	// pass. The link rule needs no second pass: a cut removes trailing bytes and
	// trims the ends, neither of which can make two characters adjacent that were
	// not adjacent before. A cut landing mid-rune drops the partial rune rather
	// than emitting replacement bytes.
	for len(s) > capBytes {
		s = strings.TrimSpace(strings.ToValidUTF8(s[:capBytes], ""))
		// Dropping the escape can expose the whitespace that preceded it, so the
		// end is trimmed again: the loop's contract is that a cut leaves no
		// dangling space, and the package's idempotence rests on it.
		s = strings.TrimSpace(trimDanglingEscape(s))
		s = neutraliseSpanAware(s)
	}
	return s
}

// trimDanglingEscape drops a trailing backslash left unpaired by a cut, so the
// result never ends in an escape looking for a character to consume.
func trimDanglingEscape(s string) string {
	n := 0
	for n < len(s) && s[len(s)-1-n] == '\\' {
		n++
	}
	if n%2 == 1 {
		return s[:len(s)-1]
	}
	return s
}

// neutraliseHTML is the whole opener rule over one run of prose: the tag half
// plus the comment delimiters. A space after the `<` is enough — CommonMark
// needs the name, `/`, `!`, or `?` to follow immediately.
func neutraliseHTML(prose string) string {
	prose = htmlTagRe.ReplaceAllStringFunc(prose, func(m string) string { return "< " + m[1:] })
	return neutraliseCommentDelimiters(prose)
}

// neutraliseCommentDelimiters breaks both HTML-comment delimiters, and it is the
// half of the rule that fires inside a code span as well as outside one: what it
// defends is not a renderer but the byte-level marker grammars the record's gates
// read (see the invariant note above). `<!` also covers a declaration, which is
// the same construct class and the same delimiter.
func neutraliseCommentDelimiters(s string) string {
	s = strings.ReplaceAll(s, "<!", "< !")
	return strings.ReplaceAll(s, "-->", "-- >")
}

// neutraliseSpanAware is the single pass over s that draws CommonMark's code-span
// boundaries and applies the right rule on each side of them. A span is a run of
// N backticks followed, anywhere later, by the next run of exactly N; a run with
// no such closer opens nothing. Backslash escapes are honoured outside a span (an
// escaped backtick opens nothing) and ignored inside one (the spec parses none
// there), so the boundary the renderer will draw is the boundary this draws.
//
// Three outcomes, one per region:
//
//   - prose outside a span: the full HTML rule, tag and comment delimiters both;
//   - the content of a balanced span: the comment delimiters only, so a quoted
//     placeholder or tag survives while a marker cannot be forged;
//   - an unpaired run: backslash-escaped, so the field cannot re-pair with a
//     neighbouring field once both are embedded on one line. The escape renders
//     as the same literal backticks the unclosed run already rendered as.
func neutraliseSpanAware(s string) string {
	var out strings.Builder
	i := 0
	for i < len(s) {
		open := nextBacktickRun(s, i)
		if open < 0 {
			out.WriteString(neutraliseHTML(s[i:]))
			break
		}
		openEnd := open
		for openEnd < len(s) && s[openEnd] == '`' {
			openEnd++
		}
		n := openEnd - open
		out.WriteString(neutraliseHTML(s[i:open]))
		closeAt := closingRun(s, openEnd, n)
		if closeAt < 0 {
			out.WriteString(strings.Repeat(`\`+"`", n))
			i = openEnd
			continue
		}
		out.WriteString(s[open:openEnd])
		out.WriteString(neutraliseCommentDelimiters(s[openEnd:closeAt]))
		out.WriteString(s[closeAt : closeAt+n])
		i = closeAt + n
	}
	return out.String()
}

// nextBacktickRun returns the index of the first backtick at or after from that
// is not backslash-escaped — the start of a run that may open a span — or -1.
// An odd number of preceding backslashes escapes the backtick; an even number
// is escaped backslashes followed by a live one.
func nextBacktickRun(s string, from int) int {
	for j := from; j < len(s); j++ {
		if s[j] != '`' {
			continue
		}
		backslashes := 0
		for k := j; k > 0 && s[k-1] == '\\'; k-- {
			backslashes++
		}
		if backslashes%2 == 0 {
			return j
		}
	}
	return -1
}

// OpensBalancedCodeSpan reports whether s BEGINS with a backtick run that a
// later run of exactly the same length closes — that is, whether s starts with a
// code span rather than with literal backticks.
//
// It exists so a caller that escapes leading block markers can ask the cleaner's
// own grammar instead of guessing. A leading run that opens a balanced span opens
// no block: a backtick fence's info string may not contain backticks, so a run
// with a matching closer on the same line is an inline span by construction, and
// escaping it would kill the shelter the cleaner's exemption relies on. Only an
// unbalanced leading run is a fence, and the cleaner no longer emits one.
func OpensBalancedCodeSpan(s string) bool {
	if s == "" || s[0] != '`' {
		return false
	}
	n := 0
	for n < len(s) && s[n] == '`' {
		n++
	}
	return closingRun(s, n, n) >= 0
}

// closingRun returns the index of the first run of exactly n backticks at or
// after from, or -1. Runs of any other length are span content and skipped
// whole, so a longer run never matches by its prefix.
func closingRun(s string, from, n int) int {
	for j := from; j < len(s); {
		if s[j] != '`' {
			j++
			continue
		}
		k := j
		for k < len(s) && s[k] == '`' {
			k++
		}
		if k-j == n {
			return j
		}
		j = k
	}
	return -1
}
