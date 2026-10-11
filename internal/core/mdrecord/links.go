package mdrecord

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// UnwrapLinkPass makes one left-to-right pass over a line of prose, a heading
// title or a principle's statement, and replaces each outermost link or image
// it can read with its label, following the CommonMark inline link grammar
// (spec 6.3 and 6.4) closely enough for either:
//
//   - the label runs to its matching `]`, with brackets balanced and a
//     backslash escape read as a literal character (matchBrackets);
//   - the inline form's destination is `<...>`, on one line with `\>` escaped,
//     or a run of non-space bytes whose unescaped parentheses balance, and an
//     optional title follows in `"..."`, `'...'` or `(...)` (inlineTail);
//   - the full and collapsed reference forms, `[label][ref]` and `[label][]`,
//     drop the second bracket pair. Whether a definition exists is not asked;
//   - the shortcut `[label]` drops its brackets when shortcuts is true, the
//     reading floor's choice: a bracketed title with no definition renders
//     with its brackets, and reading it as its label too redacts it, which is
//     the side a floor errs on. When shortcuts is false a shortcut is left as
//     written, the principle statement's choice, since without its definition
//     it is literal text; the links inside its brackets are still unwrapped;
//   - an image, `!` before any of these, is unwrapped to its alt text: the
//     heading a screen reader announces and a page without images shows.
//
// A label followed by an inline tail the scanner cannot read is left as it is
// written, brackets and all, rather than read as a shortcut. Reading it as a
// shortcut would take its brackets away and hide from the reading floor's
// backstop a link the scanner failed to read.
//
// The scanner was the reading floor's alone, and the principle statement's
// unwrap kept a pattern that ended a destination at its first `)`, so
// `[label](https://x/Audit_(finance))` left `)` and the address's tail in the
// projected statement. Both read through this one pass now.
//
// The pass is linear in the title by construction. The bracket table is one
// stack walk; the main walk visits each byte once, jumping past what it
// unwraps; and the only re-reading is an inline tail that fails, which is
// charged to a budget of the title's own length. Once the budget is spent the
// rest of the title is copied as written, so its brackets stay for the
// backstop. The work done, in bytes, is returned for the test that holds the
// bound.
func UnwrapLinkPass(s string, shortcuts bool) (string, int) {
	n := len(s)
	match := matchBrackets(s)
	var b strings.Builder
	b.Grow(n)
	work, spent := n, 0
	for i := 0; i < n; {
		c := s[i]
		if c == '\\' && i+1 < n && isASCIIPunct(s[i+1]) {
			b.WriteString(s[i : i+2])
			i += 2
			continue
		}
		open := -1
		switch {
		case c == '[':
			open = i
		case c == '!' && i+1 < n && s[i+1] == '[':
			open = i + 1
		}
		if open < 0 || match[open] < 0 {
			b.WriteByte(c)
			i++
			continue
		}
		closeAt := match[open]
		end := closeAt + 1
		switch {
		case end < n && s[end] == '(':
			if spent >= n {
				b.WriteString(s[i:])
				return b.String(), work
			}
			k, ok := inlineTail(s, end)
			if !ok {
				spent += k - end
				work += k - end
				b.WriteString(s[i : open+1])
				i = open + 1
				continue
			}
			end = k
		case end < n && s[end] == '[' && match[end] >= 0:
			end = match[end] + 1
		case !shortcuts:
			b.WriteString(s[i : open+1])
			i = open + 1
			continue
		}
		b.WriteString(s[open+1 : closeAt])
		i = end
	}
	return b.String(), work
}

// matchBrackets pairs every unescaped `[` with the `]` that closes it, in one
// stack walk: match[i] is the closing position for an opening bracket at i, and
// -1 for one never closed and for every other byte.
func matchBrackets(s string) []int {
	match := make([]int, len(s))
	var open []int
	for i := 0; i < len(s); i++ {
		match[i] = -1
		switch s[i] {
		case '\\':
			if i+1 < len(s) && isASCIIPunct(s[i+1]) {
				i++
				match[i] = -1
			}
		case '[':
			open = append(open, i)
		case ']':
			if len(open) > 0 {
				match[open[len(open)-1]] = i
				open = open[:len(open)-1]
			}
		}
	}
	return match
}

// inlineTail reads an inline link's tail, the `(destination "title")` after its
// label, starting at the `(` at j. It returns the position just past the
// closing `)` and true, or the position it stopped at and false when the tail
// is not one CommonMark reads as a link.
func inlineTail(s string, j int) (int, bool) {
	n := len(s)
	k := skipLinkSpace(s, j+1)
	if k < n && s[k] == '<' {
		for k++; ; k++ {
			if k >= n || s[k] == '\n' || s[k] == '<' {
				return k, false
			}
			if s[k] == '\\' && k+1 < n && isASCIIPunct(s[k+1]) {
				k++
				continue
			}
			if s[k] == '>' {
				k++
				break
			}
		}
	} else {
		depth := 0
		for k < n {
			c := s[k]
			if c == '\\' && k+1 < n && isASCIIPunct(s[k+1]) {
				k += 2
				continue
			}
			if c <= ' ' || c == 0x7f {
				break
			}
			if c == '(' {
				depth++
			} else if c == ')' {
				if depth == 0 {
					break
				}
				depth--
			}
			k++
		}
		if depth != 0 {
			return k, false
		}
	}
	t := skipLinkSpace(s, k)
	if t < n && s[t] == ')' {
		return t + 1, true
	}
	if t == k || t >= n {
		return t, false
	}
	closer := s[t]
	switch closer {
	case '"', '\'':
	case '(':
		closer = ')'
	default:
		return t, false
	}
	for t++; ; t++ {
		if t >= n || (closer == ')' && s[t] == '(') {
			return t, false
		}
		if s[t] == '\\' && t+1 < n && isASCIIPunct(s[t+1]) {
			t++
			continue
		}
		if s[t] == closer {
			break
		}
	}
	t = skipLinkSpace(s, t+1)
	if t < n && s[t] == ')' {
		return t + 1, true
	}
	return t, false
}

// skipLinkSpace returns the first position at or after i that is not the
// white space CommonMark allows around a link's destination and title.
func skipLinkSpace(s string, i int) int {
	for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
		i++
	}
	return i
}

// isASCIIPunct reports whether a byte is ASCII punctuation, the class a
// backslash escapes (CommonMark 2.4). Below 0x80 the union of
// Unicode's punctuation and symbol classes is exactly those 32 bytes.
func isASCIIPunct(c byte) bool {
	return c < utf8.RuneSelf && (unicode.IsPunct(rune(c)) || unicode.IsSymbol(rune(c)))
}
