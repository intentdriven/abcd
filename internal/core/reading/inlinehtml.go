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
