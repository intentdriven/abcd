package mdrender

import (
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/core/mdrecord"
)

// HeadingRe matches an ATX heading at column 0, capturing its marker run and
// its title. The site's section walk and this renderer's heading block read one
// pattern, so what the walk splits on is what the renderer renders.
var HeadingRe = regexp.MustCompile(`^(#{1,6})\s+(.*)$`)

// Block is one top-level markdown block: a paragraph, a table, a fence, an
// image line, a list, or a blockquote — whatever sits between two blank lines.
type Block struct {
	Text string
	// Line is the 1-based source line the block starts at.
	Line int
}

// Blocks splits a section body into its top-level blocks, honouring fenced code
// by mdrecord's ListNested rule, the same reading Sections takes.
// start is the 1-based source line the body begins at.
func Blocks(md string, start int) []Block {
	if md == "" {
		return nil
	}
	lines := strings.Split(md, "\n")
	fences := mdrecord.Read(lines, mdrecord.ListNested).Fences
	var out []Block
	var buf []string
	bufLine := 0
	flush := func() {
		if len(buf) > 0 {
			out = append(out, Block{Text: strings.Join(buf, "\n"), Line: bufLine})
			buf = nil
		}
	}
	for i := 0; i < len(lines); i++ {
		if len(fences) > 0 && i == fences[0].Start {
			f := fences[0]
			fences = fences[1:]
			if len(buf) == 0 {
				bufLine = start + i
			}
			buf = append(buf, lines[f.Start:f.End]...)
			i = f.End - 1
			// A fence closed at the left margin closes its own block. An
			// INDENTED one belongs to whatever list item holds it, so the block
			// runs on: the item's renderer dedents it and reads it as a fence
			// there. Ending the block here instead would leave the fence's blank
			// lines to split the code into paragraphs, and its `#` lines to be
			// read as headings — which is a document silently losing sections.
			if f.Closed && f.End-1 > f.Start && IndentOf(lines[f.End-1]) == 0 {
				flush()
			}
			continue
		}
		line := lines[i]
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
