package reading

import (
	"slices"
	"strings"
)

// The floor's readers of the raw HTML a heading title carries. Each is a walk
// over the title rather than a regular expression, because the title is the
// document's to choose and these walks must cost a bounded multiple of its
// length on any input. A pattern that reads a quoted attribute value whole
// cannot promise that under Go's leftmost-first search: a tag opened with a
// quote it never closes keeps the search alive to the end of the title before
// a later tag is reported, once per such opener, so a title of them took
// seconds at forty kilobytes and grows with the square of its length.

// quotedTagEnds answers where a tag ends when its attributes are read the way
// CommonMark and a browser read them: a quoted attribute value, in double or
// single quotes, is taken whole, `>` and all, and the tag ends at the first `>`
// outside a value. A tag ending at the first `>` stopped inside
// `<b title=">x">` and left `x">` in the title, so `## Audit <b title=">x">
// Notes`, which renders as the excluded heading, travelled.
//
// The answer from a position depends only on the first `>` or quote at or
// after it, so it is remembered against that byte, and every such byte is
// followed at most once whatever the number of tags that reach it.
type quotedTagEnds struct {
	s string
	// specials, doubles and singles list the positions of every `>` or quote,
	// every double quote and every single quote, ascending.
	specials, doubles, singles []int
	// ends maps a special byte reached outside a quoted value to the position
	// just past the tag's `>`, or -1 when a quote opened there never closes.
	ends map[int]int
}

// newQuotedTagEnds indexes a title's `>` and quote bytes in one pass.
func newQuotedTagEnds(s string) *quotedTagEnds {
	q := &quotedTagEnds{s: s, ends: map[int]int{}}
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '>':
			q.specials = append(q.specials, i)
		case '"':
			q.specials = append(q.specials, i)
			q.doubles = append(q.doubles, i)
		case '\'':
			q.specials = append(q.specials, i)
			q.singles = append(q.singles, i)
		}
	}
	return q
}

// nextIn returns the first listed position at or after i, or -1.
func nextIn(list []int, i int) int {
	k, _ := slices.BinarySearch(list, i)
	if k == len(list) {
		return -1
	}
	return list[k]
}

// endFrom returns the position just past the `>` that ends a tag whose
// attributes are read from i outside any quoted value, or -1 when none does:
// no `>` follows, or a quote opened on the way never closes.
func (q *quotedTagEnds) endFrom(i int) int {
	var path []int
	end := -1
	for k := nextIn(q.specials, i); k >= 0; {
		if v, ok := q.ends[k]; ok {
			end = v
			break
		}
		path = append(path, k)
		if q.s[k] == '>' {
			end = k + 1
			break
		}
		list := q.doubles
		if q.s[k] == '\'' {
			list = q.singles
		}
		closeAt := nextIn(list, k+1)
		if closeAt < 0 {
			break
		}
		k = nextIn(q.specials, closeAt+1)
	}
	for _, k := range path {
		q.ends[k] = end
	}
	return end
}

// tagAt returns the position just past the HTML tag that begins at s[i], or
// -1 when none does. The tag opens on htmlTagOpenRe's rule, a `<` or `</`, a
// bounded name, and a space, a slash or a `>` after it, so an autolink is left
// alone: stripping `<https://x>` turns a heading carrying a URL into a
// different heading. After a space its attributes run to the first `>` outside
// a quoted value (quotedTagEnds); after a slash the next byte must be the `>`.
func tagAt(s string, i int, q *quotedTagEnds) int {
	if s[i] != '<' {
		return -1
	}
	m := htmlTagOpenRe.FindStringIndex(s[i:])
	if m == nil {
		return -1
	}
	after := i + m[1] - 1
	switch s[after] {
	case '>':
		return after + 1
	case '/':
		if after+1 < len(s) && s[after+1] == '>' {
			return after + 2
		}
		return -1
	}
	return q.endFrom(after)
}

// stripTags replaces every HTML tag in a title with repl, read left to right
// the way a leftmost search reads it: a `<` that opens no tag is kept, and the
// walk resumes after each tag it replaces. A quote a tag opens and never
// closes leaves the tag in the title, which is how CommonMark reads it too: as
// text, a remnant the backstop judges (unreadMarkupNames).
func stripTags(s, repl string) string {
	if strings.IndexByte(s, '<') < 0 {
		return s
	}
	q := newQuotedTagEnds(s)
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		j := strings.IndexByte(s[i:], '<')
		if j < 0 {
			b.WriteString(s[i:])
			break
		}
		b.WriteString(s[i : i+j])
		i += j
		if end := tagAt(s, i, q); end > i {
			b.WriteString(repl)
			i = end
			continue
		}
		b.WriteByte('<')
		i++
	}
	return b.String()
}

// setAsideQuotedSpans removes every span from a `<` to the first `>` outside a
// quoted value, the second extent the backstop sets aside for markup no
// stripper models (unreadMarkupNames): `<x:y a=">z">` names no tag CommonMark
// reads, a browser reads it as one, and a span ending at the first `>` left
// `z` between the excluded words. A `<` whose span never ends is kept.
func setAsideQuotedSpans(s string) string {
	if strings.IndexByte(s, '<') < 0 {
		return s
	}
	q := newQuotedTagEnds(s)
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		j := strings.IndexByte(s[i:], '<')
		if j < 0 {
			b.WriteString(s[i:])
			break
		}
		b.WriteString(s[i : i+j])
		i += j
		if end := q.endFrom(i + 1); end > i {
			i = end
			continue
		}
		b.WriteByte('<')
		i++
	}
	return b.String()
}

// terminatorFinder finds a fixed terminator at or after a position in one
// title, for a walk whose searches start at positions that only advance. A
// search that fails is remembered, so a title of many openers that never close
// is searched once rather than once per opener; and a search that succeeds is
// reused by the next one that starts before the terminator it found.
type terminatorFinder struct {
	s, term string
	// noneFrom is where a search found nothing, -1 until one has failed.
	noneFrom int
	// from and at are the last successful search's start and its answer.
	from, at int
}

func newTerminatorFinder(s, term string) *terminatorFinder {
	return &terminatorFinder{s: s, term: term, noneFrom: -1, from: -1, at: -1}
}

// next returns the position of the first terminator at or after i, or -1.
func (f *terminatorFinder) next(i int) int {
	if f.noneFrom >= 0 && i >= f.noneFrom {
		return -1
	}
	if f.at >= 0 && f.from <= i && i <= f.at {
		return f.at
	}
	k := strings.Index(f.s[i:], f.term)
	if k < 0 {
		f.noneFrom = i
		return -1
	}
	f.from, f.at = i, i+k
	return f.at
}

// hiddenHTMLFinders holds one title's terminator searches.
type hiddenHTMLFinders struct {
	comment, bangComment, pi, cdata, gt *terminatorFinder
}

func newHiddenHTMLFinders(s string) *hiddenHTMLFinders {
	return &hiddenHTMLFinders{
		comment:     newTerminatorFinder(s, "-->"),
		bangComment: newTerminatorFinder(s, "--!>"),
		pi:          newTerminatorFinder(s, "?>"),
		cdata:       newTerminatorFinder(s, "]]>"),
		gt:          newTerminatorFinder(s, ">"),
	}
}

// past returns the position just past a terminator found at k, or -1.
func past(k int, term string) int {
	if k < 0 {
		return -1
	}
	return k + len(term)
}

// commonMarkHiddenAt returns the position just past the raw HTML a renderer
// passes through and a browser never shows that begins at s[i], read to the end
// CommonMark gives it (6.6), or -1: a comment, which may be `<!-->` or `<!--->`
// alone and otherwise ends at the first `-->`; a processing instruction, to the
// first `?>`; a CDATA section, to the first `]]>`; and a declaration, `<!` and
// a letter, to the first `>`. One left unclosed is not raw HTML to CommonMark,
// which renders it as text.
func commonMarkHiddenAt(s string, i int, f *hiddenHTMLFinders) int {
	rest := s[i:]
	switch {
	case strings.HasPrefix(rest, "<!-->"):
		return i + 5
	case strings.HasPrefix(rest, "<!--->"):
		return i + 6
	case strings.HasPrefix(rest, "<!--"):
		return past(f.comment.next(i+4), "-->")
	case strings.HasPrefix(rest, "<?"):
		return past(f.pi.next(i+2), "?>")
	case strings.HasPrefix(rest, "<![CDATA["):
		return past(f.cdata.next(i+9), "]]>")
	case len(rest) > 2 && rest[1] == '!' && isASCIILetter(rest[2]):
		return past(f.gt.next(i+2), ">")
	}
	return -1
}

// browserHiddenAt is commonMarkHiddenAt read to where a browser's tokenizer
// ends the same markup instead: a comment also at `--!>`, and a processing
// instruction, a declaration or a CDATA section, each a bogus comment to the
// browser outside SVG and MathML, at the first `>`. Left unclosed, each runs to
// the end of the title, as the tokenizer runs it to the end of the input.
func browserHiddenAt(s string, i int, f *hiddenHTMLFinders) int {
	rest := s[i:]
	end := -1
	switch {
	case strings.HasPrefix(rest, "<!-->"):
		return i + 5
	case strings.HasPrefix(rest, "<!--->"):
		return i + 6
	case strings.HasPrefix(rest, "<!--"):
		end = past(f.comment.next(i+4), "-->")
		if bang := past(f.bangComment.next(i+4), "--!>"); bang >= 0 && (end < 0 || bang < end) {
			end = bang
		}
	case strings.HasPrefix(rest, "<?"), strings.HasPrefix(rest, "<!"):
		end = past(f.gt.next(i+2), ">")
	default:
		return -1
	}
	if end < 0 {
		return len(s)
	}
	return end
}

// isASCIILetter reports whether a byte is an ASCII letter.
func isASCIILetter(c byte) bool {
	return 'a' <= c|0x20 && c|0x20 <= 'z'
}

// stripHidden removes the hidden raw HTML hiddenAt finds, walking the title
// left to right and resuming after each removal, the way a leftmost search
// reads it.
func stripHidden(s string, hiddenAt func(string, int, *hiddenHTMLFinders) int) string {
	if strings.IndexByte(s, '<') < 0 {
		return s
	}
	f := newHiddenHTMLFinders(s)
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		j := strings.IndexByte(s[i:], '<')
		if j < 0 {
			b.WriteString(s[i:])
			break
		}
		b.WriteString(s[i : i+j])
		i += j
		if end := hiddenAt(s, i, f); end > i {
			i = end
			continue
		}
		b.WriteByte('<')
		i++
	}
	return b.String()
}

// rawTextElements are the elements whose content a browser reads as text up
// to the element's own end tag, markup and all: script, style, iframe,
// noembed, noframes and noscript (while scripts run) as raw text, title and
// textarea as escapable text, xmp, and plaintext, which nothing ends. A walk
// through hidden content steps over theirs, so a `</template>` written inside
// a textarea in a template ends nothing.
var rawTextElements = map[string]bool{
	"script": true, "style": true, "iframe": true, "noembed": true, "noframes": true,
	"noscript": true, "title": true, "textarea": true, "xmp": true, "plaintext": true,
}

// unrenderedElements are the elements whose content a browser does not render,
// none of them needing an attribute to hide it: template, whose content is
// never in the document; script, style, title, noembed, noframes and datalist,
// which the rendering section of HTML sets to display none; iframe, whose
// content is fallback text no current browser shows; noscript, hidden while
// scripts run; and video, audio, canvas and object, whose content is fallback
// shown only where the element itself cannot be. The last are listed although
// an object with nothing to show does show its fallback: removing content
// adds a reading and takes none away, so listing an element can only redact
// more. Not listed, because a browser shows their text: textarea, xmp,
// plaintext, select and its options, and details, whose summary shows while
// the rest folds away.
var unrenderedElements = map[string]bool{
	"template": true, "script": true, "style": true, "title": true, "noembed": true,
	"noframes": true, "datalist": true, "iframe": true, "noscript": true,
	"video": true, "audio": true, "canvas": true, "object": true, "rp": true,
}

// voidElements open nothing a later end tag could be held behind, so the
// parsed walk steps over them (parsedEnd).
var voidElements = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "embed": true, "hr": true,
	"img": true, "input": true, "link": true, "meta": true, "param": true,
	"source": true, "track": true, "wbr": true,
}

// elementWalk is one title read the way a browser hides its unrendered
// elements' content (unrenderedElements). Its searches only advance, so it
// is linear in the title.
type elementWalk struct {
	s, lower string
	q        *quotedTagEnds
	f        *hiddenHTMLFinders
	closes   map[string]*terminatorFinder
	// unmodelled is set when an element the walk does not model opened
	// inside an unrendered one (parsedEnd).
	unmodelled bool
}

func newElementWalk(s string) *elementWalk {
	lower := []byte(s)
	for i, c := range lower {
		if 'A' <= c && c <= 'Z' {
			lower[i] = c + 'a' - 'A'
		}
	}
	return &elementWalk{s: s, lower: string(lower), q: newQuotedTagEnds(s),
		f: newHiddenHTMLFinders(s), closes: map[string]*terminatorFinder{}}
}

// tagAt reads the tag at s[i] (tagAt), returning its lower-cased name, whether
// it closes, and the position just past it, or -1 when no tag begins there.
func (w *elementWalk) tagAt(i int) (string, bool, int) {
	end := tagAt(w.s, i, w.q)
	if end < 0 {
		return "", false, -1
	}
	m := htmlTagOpenRe.FindStringIndex(w.s[i:])
	from, closing := i+1, w.s[i+1] == '/'
	if closing {
		from++
	}
	return w.lower[from : i+m[1]-1], closing, end
}

// rawTextEnd returns the position just past the end tag that closes a
// raw-text element whose content starts at c, or the end of the title when
// none does: the browser reads it to the end of the input. The end tag is the
// element's own name in any case, followed by a space, a slash or a `>`.
func (w *elementWalk) rawTextEnd(name string, c int) int {
	f := w.closes[name]
	if f == nil {
		f = newTerminatorFinder(w.lower, "</"+name)
		w.closes[name] = f
	}
	for k := f.next(c); k >= 0; k = f.next(k + 1) {
		b := k + 2 + len(name)
		if b >= len(w.s) {
			break
		}
		switch w.s[b] {
		case ' ', '\t', '\n', '\f', '\r', '/', '>':
			if end := w.q.endFrom(b); end >= 0 {
				return end
			}
			return len(w.s)
		}
	}
	return len(w.s)
}

// parsedEnd returns the position just past the end tag that closes a parsed
// unrendered element, a template or a video, whose content starts at c, or
// the end of the title when none does. Inside it, hidden raw HTML is stepped
// over (browserHiddenAt), a raw-text element's content too (rawTextEnd), an
// unrendered element of the same kind nests, and an end tag closes the
// innermost open element it names, as a browser's parser closes it.
func (w *elementWalk) parsedEnd(name string, c int) int {
	stack := []string{name}
	open := map[string]int{name: 1}
	for i := c; i < len(w.s); {
		j := strings.IndexByte(w.s[i:], '<')
		if j < 0 {
			break
		}
		i += j
		if end := browserHiddenAt(w.s, i, w.f); end > i {
			i = end
			continue
		}
		n, closing, end := w.tagAt(i)
		switch {
		case end < 0:
			i++
			continue
		case closing && open[n] > 0:
			for {
				top := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				open[top]--
				if top == n {
					break
				}
			}
			if len(stack) == 0 {
				return end
			}
		case closing:
		case rawTextElements[n]:
			end = w.rawTextEnd(n, end)
		case unrenderedElements[n]:
			stack = append(stack, n)
			open[n]++
		case !voidElements[n] && stack[0] != "template":
			// An element the walk does not model can leave the parser
			// ignoring the unrendered element's end tag (an audio end tag
			// while a div is open is dropped), so content after that end tag
			// may stay hidden. The walk cannot say how far, so it flags the
			// title and the floor refuses it (unmodelledHides). A template's
			// content is its own fragment, closed by its first end tag, so it
			// is exempt.
			w.unmodelled = true
		}
		i = end
	}
	return len(w.s)
}

// browserView is a title as a browser shows it: its hidden raw HTML removed
// where a browser ends it (browserHiddenAt), and every unrendered element
// removed with its content (unrenderedElements), each unclosed one to the end
// of the title.
func browserView(s string) string {
	v, _ := browserWalk(s)
	return v
}

// browserWalk is browserView, also reporting whether the walk met an element
// it does not model inside an unrendered one, where the parser may hide more
// than the walk can tell (parsedEnd).
func browserWalk(s string) (string, bool) {
	if strings.IndexByte(s, '<') < 0 {
		return s, false
	}
	w := newElementWalk(s)
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		j := strings.IndexByte(s[i:], '<')
		if j < 0 {
			b.WriteString(s[i:])
			break
		}
		b.WriteString(s[i : i+j])
		i += j
		if end := browserHiddenAt(s, i, w.f); end > i {
			i = end
			continue
		}
		if n, closing, end := w.tagAt(i); end > i && !closing && unrenderedElements[n] {
			if rawTextElements[n] {
				i = w.rawTextEnd(n, end)
			} else {
				i = w.parsedEnd(n, end)
			}
			continue
		}
		b.WriteByte('<')
		i++
	}
	return b.String(), w.unmodelled
}

// hiddenViews returns a title with its hidden markup removed under each
// reading the floor takes of it. CommonMark decides what is raw HTML at all
// and a browser decides how much of it is hidden, and the two end a comment,
// an instruction or a CDATA section in different places; and a browser shows
// no content for an unrendered element where a sanitizing renderer drops the
// tag and shows its content. So the title is read three ways — hidden HTML to
// CommonMark's end, to a browser's end, and as a browser shows it with its
// unrendered elements' content removed (browserView) — and it is the excluded
// heading if ANY reading names it.
func hiddenViews(s string) []string {
	var views []string
	for _, v := range []string{
		stripHidden(s, commonMarkHiddenAt), stripHidden(s, browserHiddenAt), browserView(s),
	} {
		if !slices.Contains(views, v) {
			views = append(views, v)
		}
	}
	return views
}
