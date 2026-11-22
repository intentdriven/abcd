package positioning

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// diffContext is how many unchanged lines flank each hunk.
const diffContext = 3

// SurfaceDiff is one surface's proposed correction: the whole proposed file
// content and the unified diff that turns the current file into it.
type SurfaceDiff struct {
	SurfaceID string `json:"surfaceId"`
	File      string `json:"file"`
	// Proposed is the surface file as it would read once the proposal is
	// adopted. It is returned, never written.
	Proposed string `json:"-"`
	// Unified is the human-readable patch: the exact drifted line removed, the
	// canonical line added, with surrounding context.
	Unified string `json:"unified"`
}

// Proposal is a re-render of every drifted surface from the identity block. It
// is a PROPOSAL: nothing here touches the working tree, and there is no flag
// that makes it. A deliberate identity change is an edit to the block, after
// which the same proposal flow chases the surfaces.
type Proposal struct {
	Report Report        `json:"report"`
	Diffs  []SurfaceDiff `json:"diffs"`
}

// Unified concatenates every surface's patch into one diff a human can read or
// pipe to `git apply`.
func (p Proposal) Unified() string {
	var b strings.Builder
	for _, d := range p.Diffs {
		b.WriteString(d.Unified)
	}
	return b.String()
}

// Propose runs the positioning check and, for every surface that drifted,
// renders what the surface would say if it were re-rendered from the block.
// It performs no writes whatsoever.
//
// A surface the locator could not find yields no diff — there is no region to
// replace, and inventing one would propose a patch against a line that was
// never located. It still reports as drifted in the embedded Report.
func Propose(root string, cfg Config) (Proposal, error) {
	rep, err := Check(root, cfg)
	if err != nil {
		return Proposal{}, err
	}
	byID := map[string]Surface{}
	for _, s := range cfg.surfaces() {
		byID[s.ID] = s
	}

	// The re-read is contained exactly as the check's read was: a proposal
	// carries the surface's current bytes into its diff, so an uncontained read
	// here would leak outside content through the patch instead of the report.
	r, err := openRepoRoot(root)
	if err != nil {
		return Proposal{}, err
	}
	defer r.Close()

	p := Proposal{Report: rep}
	for _, res := range rep.Surfaces {
		if res.Status != StatusDrifted {
			continue
		}
		s, ok := byID[res.ID]
		if !ok {
			continue
		}
		_, data, present, rerr := readFirstPresent(r, []string{res.File})
		if rerr != nil {
			return Proposal{}, rerr
		}
		if !present {
			continue
		}
		text := normalizeNewlines(data)
		sp, located, lerr := s.locateSpan(text)
		if lerr != nil {
			return Proposal{}, lerr
		}
		if !located || sp.start < 0 {
			continue
		}

		replacement := s.render(rep.Block)
		if sp.escaped {
			// Splicing back into a quoted string: the canonical line must be
			// re-escaped or the proposed file would not parse.
			esc, merr := escapeQuoted(replacement)
			if merr != nil {
				return Proposal{}, merr
			}
			replacement = esc
		}
		proposed := text[:sp.start] + replacement + text[sp.end:]
		// The locators work over newline-normalised text, but the proposal has
		// to match the bytes on disk or no patch tool will apply it: a file
		// written with CRLF gets its line endings back.
		crlf := bytes.Contains(data, []byte("\r\n"))
		onDisk := proposed
		if crlf {
			onDisk = strings.ReplaceAll(proposed, "\n", "\r\n")
		}
		p.Diffs = append(p.Diffs, SurfaceDiff{
			SurfaceID: res.ID,
			File:      res.File,
			Proposed:  onDisk,
			Unified:   unifiedDiff(res.File, text, sp.start, sp.end, replacement, crlf),
		})
	}
	return p, nil
}

// escapeQuoted spells s as the body of a JSON string literal. The same body is
// valid inside a YAML double-quoted scalar, whose escapes are a superset of
// JSON's, so one spelling serves a JSON manifest field, an EscapeJSON surface
// and an EscapeYAML surface alike.
func escapeQuoted(s string) (string, error) {
	enc, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return string(enc[1 : len(enc)-1]), nil
}

// noNewlineMarker is the unified-diff notation for a final line that the file
// does not terminate with a newline. Omitting it makes every patch tool refuse.
const noNewlineMarker = "\\ No newline at end of file\n"

// unifiedDiff renders the single-region replacement as a unified diff. The
// change is always one contiguous span, so the patch is exactly one hunk: the
// lines the span touches, replaced, flanked by up to diffContext unchanged
// lines. That is enough to be a real patch without pulling in a diff algorithm.
//
// old is the newline-normalised file text and start/end are byte offsets into
// it; crlf says the file on disk uses CRLF, in which case the emitted lines
// carry the carriage return back so the patch matches the real bytes.
//
// The CRLF handling assumes a file's line endings are uniform: one CRLF
// anywhere makes every emitted line carry a carriage return, so a mixed-ending
// file yields a hunk whose LF-only lines gain a CR they do not have on disk.
// That is left as it is because the failure is loud rather than silent — the
// context lines no longer match, `git apply` refuses the patch, and nothing is
// written on the strength of a wrong diff.
func unifiedDiff(file, old string, start, end int, replacement string, crlf bool) string {
	eol := "\n"
	if crlf {
		eol = "\r\n"
	}
	// A file ending in a newline has N lines, not N+1: splitting the raw text
	// yields a phantom trailing empty line that is not in the file, and a hunk
	// carrying it as context can never apply. Where the file really does not
	// end in a newline, the hunk says so with the marker instead.
	finalNewline := strings.HasSuffix(old, "\n")
	oldLines := strings.Split(strings.TrimSuffix(old, "\n"), "\n")

	// Clamped: a span that reached past the last content line (a locator that
	// swallowed the file's trailing newline) must not index past oldLines.
	firstLine := min(strings.Count(old[:start], "\n"), len(oldLines)-1)
	lastLine := min(strings.Count(old[:end], "\n"), len(oldLines)-1)
	end = min(end, lineEndOffset(old, lastLine))

	// The replaced lines, rebuilt: everything on the first line before the
	// span, the replacement, everything on the last line after it.
	lineStart := lineStartOffset(old, firstLine)
	lineEnd := lineEndOffset(old, lastLine)
	newBlock := old[lineStart:start] + replacement + old[end:lineEnd]

	removed := oldLines[firstLine : lastLine+1]
	added := strings.Split(newBlock, "\n")

	ctxStart := max(0, firstLine-diffContext)
	ctxEnd := min(len(oldLines)-1, lastLine+diffContext)
	before := oldLines[ctxStart:firstLine]
	after := oldLines[lastLine+1 : ctxEnd+1]

	// The hunk reaches the file's last line, so an absent final newline has to
	// be declared — against the shared context line if there is one after the
	// change, otherwise against both sides of the change itself.
	truncated := !finalNewline && ctxEnd == len(oldLines)-1

	var b strings.Builder
	fmt.Fprintf(&b, "--- a/%s\n+++ b/%s\n", file, file)
	fmt.Fprintf(&b, "@@ -%d,%d +%d,%d @@\n",
		ctxStart+1, len(before)+len(removed)+len(after),
		ctxStart+1, len(before)+len(added)+len(after))
	for _, l := range before {
		b.WriteString(" " + l + eol)
	}
	writeSide := func(prefix string, lines []string, mark bool) {
		for i, l := range lines {
			b.WriteString(prefix + l + eol)
			if mark && i == len(lines)-1 {
				b.WriteString(noNewlineMarker)
			}
		}
	}
	writeSide("-", removed, truncated && len(after) == 0)
	writeSide("+", added, truncated && len(after) == 0)
	writeSide(" ", after, truncated && len(after) > 0)
	return b.String()
}

// lineStartOffset returns the byte offset of the start of 0-based line n.
func lineStartOffset(s string, n int) int {
	off := 0
	for i := 0; i < n; i++ {
		idx := strings.IndexByte(s[off:], '\n')
		if idx < 0 {
			return off
		}
		off += idx + 1
	}
	return off
}

// lineEndOffset returns the byte offset just past the end of 0-based line n
// (excluding its newline).
func lineEndOffset(s string, n int) int {
	off := lineStartOffset(s, n)
	if idx := strings.IndexByte(s[off:], '\n'); idx >= 0 {
		return off + idx
	}
	return len(s)
}
