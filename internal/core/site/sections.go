package site

// The markdown structure walk, ported from `sections`/`blocks`/`slug`/
// `strip_frontmatter` in `.abcd/development/research/abcdev-site/compose.py` —
// the executable spec of the composition. The walk is deliberately naive in the
// same places the script is: it splits on headings and blank lines while
// honouring fenced code, and knows nothing about inline markdown. Everything the
// layouts do (which section becomes a card, which block is a lead-in, which
// image is the figure) is expressed in terms of these two shapes, so a
// divergence here would move every layout at once.
//
// The one thing added: every section and block carries the 1-based source line
// it starts at, because a build that refuses an out-of-subset construct has to
// be able to say where it is.

import (
	"regexp"
	"strings"
)

var (
	headingRe   = regexp.MustCompile(`^(#{1,6})\s+(.*)$`)
	nonSlugRe   = regexp.MustCompile(`[^a-z0-9]+`)
	slugStripRe = regexp.MustCompile("[`*_]")
)

// Section is one heading and the body that follows it, down to the next
// heading. The first section of a document is the implicit preamble: level 0,
// no title, and whatever text precedes the first heading. A document whose
// first line is an H1 therefore yields no preamble section at all, because the
// walk drops a level-0 section with an empty body.
type Section struct {
	// Level is the heading depth (1..6), or 0 for the implicit preamble.
	Level int
	// Title is the heading text, trimmed.
	Title string
	// Anchor is Slug(Title).
	Anchor string
	// Body is the text between this heading and the next, with leading and
	// trailing blank lines removed.
	Body string
	// Line is the 1-based source line of the heading (0 for the preamble).
	Line int
	// BodyLine is the 1-based source line Body starts at.
	BodyLine int
}

// Block is one top-level markdown block: a paragraph, a table, a fence, an
// image line, a list, or a blockquote — whatever sits between two blank lines.
type Block struct {
	Text string
	// Line is the 1-based source line the block starts at.
	Line int
}

// StripFrontmatter removes a leading YAML frontmatter block and reports how many
// lines it consumed, so a caller can keep reporting source lines against the
// original file. It is the script's `strip_frontmatter`: a document that does
// not open with `---`, or whose frontmatter never closes, is returned untouched.
//
// Blank lines and complete HTML comments AHEAD of the opening `---` are skipped
// with it. A record file may carry an attribution comment above its frontmatter
// — every glossary term file does, from the template it was adapted from — and a
// stripper that recognised frontmatter only at byte zero published the whole
// block as prose: the fields arrived on the page as a heading nobody wrote. The
// three record readers that already tolerate that comment (`lint`, `glossary`
// and this) now agree about where a file's frontmatter begins.
func StripFrontmatter(t string) (string, int) {
	lead := frontmatterLead(t)
	rest := t[lead:]
	if !strings.HasPrefix(rest, "---") {
		return t, 0
	}
	end := strings.Index(rest[3:], "\n---")
	if end < 0 {
		return t, 0
	}
	end += 3
	cut := end + 4
	if cut > len(rest) {
		return t, 0
	}
	return rest[cut:], strings.Count(t[:lead+cut], "\n")
}

// frontmatterLead is the byte offset of the first line that is neither blank nor
// part of an HTML comment. It is 0 for a document that opens with anything else,
// which is every document that carries no such preamble.
func frontmatterLead(t string) int {
	at, inComment := 0, false
	for at < len(t) {
		nl := strings.IndexByte(t[at:], '\n')
		lineEnd := len(t)
		next := len(t)
		if nl >= 0 {
			lineEnd = at + nl
			next = lineEnd + 1
		}
		line := strings.TrimSpace(t[at:lineEnd])
		switch {
		case inComment:
			if strings.Contains(line, "-->") {
				inComment = false
			}
		case line == "":
			// a blank line: skip.
		case strings.HasPrefix(line, "<!--") && strings.HasSuffix(line, "-->"):
			// a complete single-line comment: skip.
		case strings.HasPrefix(line, "<!--"):
			inComment = true
		default:
			return at
		}
		at = next
	}
	// Nothing but blanks and comments: there is no frontmatter to find, and the
	// caller's HasPrefix check returns the document untouched.
	return at
}

// Sections splits markdown into its headings and their bodies, honouring fenced
// code so a `#` comment inside a shell block is never read as a heading.
// offset is the number of lines already consumed ahead of md (the frontmatter),
// so the reported lines are the ones a reader would find in the file.
//
// A fence that never closes is refused, naming the line that opened it. It is
// the quietest failure this walk has: once the fence is open no later heading is
// a heading, so every section after it silently ceases to exist and the page
// renders a document that is short in a way nobody notices. Refusing costs one
// edit; not refusing costs a missing chapter nobody is looking for.
func Sections(path, md string, offset int) ([]Section, error) {
	lines := strings.Split(md, "\n")
	var out []Section
	cur := Section{Line: 0}
	var body []string
	bodyStart := offset + 1
	fence := false
	fenceLine := 0

	flush := func() {
		cur.Body, cur.BodyLine = trimBlankLines(body, bodyStart)
		out = append(out, cur)
	}

	for i, line := range lines {
		if isFenceLine(line) {
			fence = !fence
			if fence {
				fenceLine = offset + i + 1
			}
		}
		var m []string
		if !fence {
			m = headingRe.FindStringSubmatch(line)
		}
		if m != nil {
			flush()
			title := strings.TrimSpace(m[2])
			cur = Section{Level: len(m[1]), Title: title, Anchor: Slug(title), Line: offset + i + 1}
			body = nil
			bodyStart = offset + i + 2
			continue
		}
		body = append(body, line)
	}
	flush()

	if fence {
		return nil, &UnsupportedError{path, fenceLine, "unterminated fenced code block",
			"it swallows every heading after it, so the rest of the document silently stops existing"}
	}

	// A section that is neither a heading nor text is nothing: the script drops
	// it so a document opening on its H1 does not grow an empty preamble.
	kept := out[:0]
	for _, s := range out {
		if s.Level != 0 || s.Body != "" {
			kept = append(kept, s)
		}
	}
	return kept, nil
}

// trimBlankLines removes leading and trailing blank lines from a body and
// returns it with the source line its first retained line sits on.
func trimBlankLines(body []string, start int) (string, int) {
	lo, hi := 0, len(body)
	for lo < hi && strings.TrimSpace(body[lo]) == "" {
		lo++
	}
	for hi > lo && strings.TrimSpace(body[hi-1]) == "" {
		hi--
	}
	if lo == hi {
		return "", start
	}
	return strings.Join(body[lo:hi], "\n"), start + lo
}

// isFenceLine reports whether a line opens or closes a fenced code block. An
// INDENTED fence counts: the record writes them inside list items, and a walk
// that only saw the left margin would read the fence's own `#` lines as headings
// and split the document at them.
func isFenceLine(line string) bool {
	return strings.HasPrefix(strings.TrimLeft(line, " \t"), "```")
}

// Slug renders a heading as its anchor: emphasis and code marks dropped,
// lower-cased, every other run of non-alphanumerics collapsed to a hyphen.
func Slug(t string) string {
	t = strings.ToLower(slugStripRe.ReplaceAllString(t, ""))
	return strings.Trim(nonSlugRe.ReplaceAllString(t, "-"), "-")
}

// Blocks splits a section body into its top-level blocks, honouring fenced code.
// start is the 1-based source line the body begins at.
func Blocks(md string, start int) []Block {
	if md == "" {
		return nil
	}
	var out []Block
	var buf []string
	bufLine := 0
	fence := false
	flush := func() {
		if len(buf) > 0 {
			out = append(out, Block{Text: strings.Join(buf, "\n"), Line: bufLine})
			buf = nil
		}
	}
	for i, line := range strings.Split(md, "\n") {
		if isFenceLine(line) {
			if !fence && len(buf) == 0 {
				bufLine = start + i
			}
			fence = !fence
			buf = append(buf, line)
			// A fence at the left margin closes its own block. An INDENTED one
			// belongs to whatever list item holds it, so the block runs on: the
			// item's renderer dedents it and reads it as a fence there. Ending
			// the block here instead would leave the fence's blank lines to
			// split the code into paragraphs, and its `#` lines to be read as
			// headings — which is a document silently losing sections.
			if !fence && indentOf(line) == 0 {
				flush()
			}
			continue
		}
		if fence {
			buf = append(buf, line)
			continue
		}
		if strings.TrimSpace(line) == "" {
			flush()
			continue
		}
		if len(buf) == 0 {
			bufLine = start + i
		}
		buf = append(buf, line)
	}
	flush()
	return out
}
