package ahoy

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/mdrecord"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// README parts: the named sibling of marker.go's one unnamed block
// (spc-2610031704338417 step 1, for itd-2610031348087517).
//
// The conventions block is one per file and is compared byte for byte with an
// embedded template. A README part's text differs per project, so that
// comparison cannot tell abcd's text from the owner's. Each part therefore
// sits in its own named place, and the opening fence carries the hash of the
// text abcd last wrote there:
//
//	<!-- BEGIN ABCD:licence sha256=<64 hex digits> -->
//	Licensed under the MIT License; see [LICENSE](LICENSE).
//	<!-- END ABCD:licence -->
//
// A body that no longer matches its hash was edited by hand, and is named and
// left. A named fence never matches markerBlockRe, whose pattern requires
// " -->" straight after "ABCD", and the conventions fences never match the
// patterns below, which require a colon there; neither reading sees the
// other's blocks.
//
// Where the owner puts a part decides its form (iss-2610311544004033): a part
// whose opening fence continues an open block-level HTML block, such as a
// centred badge row in a <p>, is written in its inline form, as HTML on lines
// of its own, since the forge renders no Markdown there; anywhere else it is
// written in its block form.
//
// readme_derive.go derives each part's body, readme_check.go checks it, and
// readme_apply.go wires setup (`abcd ahoy install`, from the stored
// `readme.parts`) and uninstall to the write and the removal here.

// readmePartOrder is the parts' fixed order. It places an absent part beside
// the nearest present one, and only these names may be written.
var readmePartOrder = []string{"release-badge", "build-badge", "abcd-badge", "licence", "links", "contents"}

// readmePartIndex is name's place in the fixed order, or -1 for a name outside
// it (a fence the owner wrote, or one from a later abcd).
func readmePartIndex(name string) int {
	for i, n := range readmePartOrder {
		if n == name {
			return i
		}
	}
	return -1
}

var (
	// readmeOpenRe is a part's opening fence: the name, then attributes. A
	// fence counts only on a line of its own, at up to three spaces of indent
	// (deeper, it would be an indented code block).
	readmeOpenRe = regexp.MustCompile(`^ {0,3}<!-- BEGIN ABCD:([a-z0-9][a-z0-9-]*)((?: [^ >]+)*) -->[ \t]*$`)
	// readmeCloseRe is a part's closing fence.
	readmeCloseRe = regexp.MustCompile(`^ {0,3}<!-- END ABCD:([a-z0-9][a-z0-9-]*) -->[ \t]*$`)
	// readmeHashAttrRe reads the hash attribute out of an opening fence's
	// attributes. An opening fence without one, or with a malformed one, reads
	// as edited by hand: abcd cannot prove it wrote the body.
	readmeHashAttrRe = regexp.MustCompile(`(?:^| )sha256=([0-9a-f]{64})(?: |$)`)
)

// readmePartState is what the file holds for one part, measured against the
// part's fresh derivation. The six states are the spec's table.
type readmePartState string

const (
	// readmePartAbsent: no fence for the part. Setup writes it.
	readmePartAbsent readmePartState = "absent"
	// readmePartCurrent: the hash matches the body and the body equals the
	// fresh derivation. Setup writes nothing.
	readmePartCurrent readmePartState = "current"
	// readmePartStale: the hash matches the body and the body differs from the
	// fresh derivation. Setup rewrites the body and the hash where it stands.
	readmePartStale readmePartState = "stale"
	// readmePartEdited: the hash does not match the body. Setup names the part
	// and its line, and leaves it.
	readmePartEdited readmePartState = "edited by hand"
	// readmePartDuplicated: two opening fences with one name. The whole write
	// is refused, naming both lines.
	readmePartDuplicated readmePartState = "duplicated"
	// readmePartUnclosed: an opening fence with no closing fence, or a closing
	// fence with no opening fence. The whole write is refused, naming the line.
	readmePartUnclosed readmePartState = "unclosed"
	// readmePartUnknown: a well-formed part whose name is not in the fixed
	// order, such as one a later abcd wrote. This abcd cannot tell whether it
	// is still wanted, so a write and a removal alike name it and its line,
	// and leave it, whether or not its body matches its hash.
	readmePartUnknown readmePartState = "not one this abcd writes"
)

// readmePartAction is what a write or a removal did with one part.
type readmePartAction string

const (
	readmeActionWritten   readmePartAction = "written"
	readmeActionRewritten readmePartAction = "rewritten"
	readmeActionRemoved   readmePartAction = "removed"
	readmeActionKept      readmePartAction = "kept"    // current: nothing to do
	readmeActionLeft      readmePartAction = "left"    // edited by hand
	readmeActionRefused   readmePartAction = "refused" // a broken file, or a body that would break its marks
)

// readmePartWant is one chosen part with its freshly derived bodies: Body, the
// block form, for a part that stands on its own, and Inline, the inline form,
// for a part whose opening mark continues an open HTML block (readmeInlineLines).
// Neither carries a trailing newline; CRLF in them reads as LF.
type readmePartWant struct{ Name, Body, Inline string }

// readmePartOutcome says what happened to one part, for the summary. Lines are
// 1-based lines of the file as it was read.
type readmePartOutcome struct {
	Name   string
	State  readmePartState
	Action readmePartAction
	Lines  []int
	Note   string
}

// readmePartsResult is a write's or a removal's outcome. Refused means a
// duplicated or unclosed fence left the whole file as it was; Created means
// the write made a README that was not there.
type readmePartsResult struct {
	Wrote   bool
	Refused bool
	Created bool
	Parts   []readmePartOutcome
}

// bodyFor is the want's body for a place of the given form. A want with no
// inline form (only a test builds one) is written as its block form.
func (w readmePartWant) bodyFor(inline bool) string {
	if inline && w.Inline != "" {
		return normaliseReadmeBody(w.Inline)
	}
	return normaliseReadmeBody(w.Body)
}

// readmePartSpan is one well-formed part: its fences' 0-based line indices,
// the hash its opening fence records ("" when it records none), its body, the
// lines between the fences joined with LF, and whether its opening fence
// continues an open HTML block, so it is written in its inline form
// (readmeInlineLines).
type readmePartSpan struct {
	Name       string
	Begin, End int
	Hash, Body string
	Inline     bool
}

// hashMatches reports whether the body is the text abcd recorded writing.
func (s readmePartSpan) hashMatches() bool { return s.Hash != "" && s.Hash == readmePartHash(s.Body) }

// readmeFault is a duplicated or unclosed fence. Lines are 0-based; closer
// marks an unclosed fault found at a closing fence with no opening fence.
type readmeFault struct {
	Name   string
	State  readmePartState
	Lines  []int
	closer bool
}

// readmeDoc is one reading of a README: its lines with their byte offsets, the
// well-formed parts in file order, and the structural faults.
type readmeDoc struct {
	starts, ends []int
	parts        []readmePartSpan
	faults       []readmeFault
}

// readmePartHash is SHA-256 over the body's lines joined with LF, with no
// trailing newline, whatever the file's own line ending.
func readmePartHash(body string) string {
	sum := sha256.Sum256([]byte(normaliseReadmeBody(body)))
	return hex.EncodeToString(sum[:])
}

func normaliseReadmeBody(body string) string { return strings.ReplaceAll(body, "\r\n", "\n") }

// renderReadmePart builds one part's fences and body for a file whose line
// ending is eol, with no line ending after the closing fence.
func renderReadmePart(name, body string, eol []byte) []byte {
	body = normaliseReadmeBody(body)
	var b bytes.Buffer
	b.WriteString("<!-- BEGIN ABCD:" + name + " sha256=" + readmePartHash(body) + " -->")
	if body != "" {
		for _, ln := range strings.Split(body, "\n") {
			b.Write(eol)
			b.WriteString(ln)
		}
	}
	b.Write(eol)
	b.WriteString("<!-- END ABCD:" + name + " -->")
	return b.Bytes()
}

// parseReadmeParts reads every part and every structural fault in content. A
// fence line counts only where mdrecord.Mask does not set MaskFence, so a
// README that shows the fences inside a code example holds no part.
func parseReadmeParts(content []byte) readmeDoc {
	lines, starts, ends := markerLines(content)
	mask := mdrecord.Mask(lines)
	inline := readmeInlineLines(content)
	d := readmeDoc{starts: starts, ends: ends}
	openers := map[string][]int{}
	var open *readmePartSpan
	for i, ln := range lines {
		if mask[i]&mdrecord.MaskFence != 0 {
			continue
		}
		if m := readmeOpenRe.FindStringSubmatch(ln); m != nil {
			openers[m[1]] = append(openers[m[1]], i)
			// A part left open by a different name's opening fence never
			// closes; one left open by its own name's is the duplicate, which
			// names it already.
			if open != nil && open.Name != m[1] {
				d.faults = append(d.faults, readmeFault{Name: open.Name, State: readmePartUnclosed, Lines: []int{open.Begin}})
			}
			hash := ""
			if h := readmeHashAttrRe.FindStringSubmatch(m[2]); h != nil {
				hash = h[1]
			}
			open = &readmePartSpan{Name: m[1], Begin: i, Hash: hash, Inline: inline[i]}
			continue
		}
		if m := readmeCloseRe.FindStringSubmatch(ln); m != nil {
			if open == nil || open.Name != m[1] {
				d.faults = append(d.faults, readmeFault{Name: m[1], State: readmePartUnclosed, Lines: []int{i}, closer: true})
				continue
			}
			open.End = i
			open.Body = strings.Join(lines[open.Begin+1:i], "\n")
			d.parts = append(d.parts, *open)
			open = nil
		}
	}
	if open != nil {
		d.faults = append(d.faults, readmeFault{Name: open.Name, State: readmePartUnclosed, Lines: []int{open.Begin}})
	}
	for name, at := range openers {
		if len(at) > 1 {
			d.faults = append(d.faults, readmeFault{Name: name, State: readmePartDuplicated, Lines: at})
		}
	}
	sort.SliceStable(d.faults, func(i, j int) bool { return d.faults[i].Lines[0] < d.faults[j].Lines[0] })
	return d
}

// find returns the well-formed part named name.
func (d readmeDoc) find(name string) (readmePartSpan, bool) {
	for _, p := range d.parts {
		if p.Name == name {
			return p, true
		}
	}
	return readmePartSpan{}, false
}

// classify is the state of the part named name against its fresh derivation,
// with the 1-based lines a summary names (none for absent, current or stale).
func (d readmeDoc) classify(name, fresh string) (readmePartState, []int) {
	for _, want := range []readmePartState{readmePartDuplicated, readmePartUnclosed} {
		for _, f := range d.faults {
			if f.Name == name && f.State == want {
				return f.State, oneBased(f.Lines)
			}
		}
	}
	p, ok := d.find(name)
	switch {
	case !ok:
		return readmePartAbsent, nil
	case !p.hashMatches():
		return readmePartEdited, []int{p.Begin + 1}
	case p.Body == normaliseReadmeBody(fresh):
		return readmePartCurrent, nil
	}
	return readmePartStale, nil
}

func oneBased(lines []int) []int {
	out := make([]int, len(lines))
	for i, l := range lines {
		out[i] = l + 1
	}
	return out
}

// spanBytes is a part's byte range, from its opening fence to the end of its
// closing fence, the closing fence's line ending excluded.
func (d readmeDoc) spanBytes(content []byte, p readmePartSpan) (start, end int) {
	start, end = d.starts[p.Begin], d.ends[p.End]
	for end > start && (content[end-1] == '\n' || content[end-1] == '\r') {
		end--
	}
	return start, end
}

// planReadmeParts composes content holding exactly the wanted parts: a current
// part is kept, a stale one rewritten where it stands, an absent one placed,
// and a part present but not wanted removed. A part edited by hand is left,
// wanted or not, and the others are still written. A duplicated or unclosed
// fence refuses everything and returns content unchanged. A part whose name
// is not in the fixed order is left, and named, as an edited one is. A want
// naming a part outside the fixed order, or naming one twice, is the caller's
// error.
func planReadmeParts(content []byte, want []readmePartWant) ([]byte, readmePartsResult, error) {
	fresh := map[string]readmePartWant{}
	for _, w := range want {
		if readmePartIndex(w.Name) < 0 {
			return nil, readmePartsResult{}, fmt.Errorf("unknown README part %q (known: %s)", w.Name, strings.Join(readmePartOrder, ", "))
		}
		if _, dup := fresh[w.Name]; dup {
			return nil, readmePartsResult{}, fmt.Errorf("README part %q chosen twice", w.Name)
		}
		fresh[w.Name] = w
	}
	var res readmePartsResult
	doc := parseReadmeParts(content)
	if len(doc.faults) > 0 {
		res.Refused = true
		for _, f := range doc.faults {
			res.Parts = append(res.Parts, readmeFaultOutcome(f))
		}
		return content, res, nil
	}

	// A body that would not read back as its own part (a line that reads as a
	// fence, a code block it never closes), or an inline one holding a blank
	// line, which would end the HTML block it stands in, is refused, and the
	// part is left as the file holds it.
	refused := func(name string, inline bool) bool {
		body := fresh[name].bodyFor(inline)
		return !readmeBodyReadsBack(name, body) || (inline && readmeBodyHasBlankLine(body))
	}

	eol := detectEOL(content)
	out := content
	present := map[string]bool{}
	var outcomes []readmePartOutcome
	// Present parts, last first, so the byte offsets of the earlier ones hold
	// while the later ones change.
	for i := len(doc.parts) - 1; i >= 0; i-- {
		p := doc.parts[i]
		present[p.Name] = true
		w, wanted := fresh[p.Name]
		body := w.bodyFor(p.Inline)
		o := readmePartOutcome{Name: p.Name}
		start, end := doc.spanBytes(out, p)
		switch {
		case readmePartIndex(p.Name) < 0:
			o.State, o.Action, o.Lines = readmePartUnknown, readmeActionLeft, []int{p.Begin + 1}
			o.Note = fmt.Sprintf("%s part at line %d is not one this abcd writes; left as it is", p.Name, p.Begin+1)
		case !p.hashMatches():
			o.State, o.Action, o.Lines = readmePartEdited, readmeActionLeft, []int{p.Begin + 1}
			o.Note = fmt.Sprintf("%s part edited by hand at line %d; left as it is", p.Name, p.Begin+1)
		case wanted && refused(p.Name, p.Inline):
			o.State, o.Action = readmePartStale, readmeActionRefused
			if p.Body == body {
				o.State = readmePartCurrent
			}
			o.Note = readmeBodyRefusal(p.Name)
		case !wanted && p.Inline:
			// The part's own lines go, and no blank line takes their place:
			// one would end the HTML block it stood in.
			out = concat(out[:doc.starts[p.Begin]], out[doc.ends[p.End]:])
			o.State, o.Action = readmePartCurrent, readmeActionRemoved
			o.Note = p.Name + " part removed"
		case !wanted:
			out = composeMarkerRemoval(out, [][]int{{start, end}})
			o.State, o.Action = readmePartCurrent, readmeActionRemoved
			o.Note = p.Name + " part removed"
		case p.Body == body:
			o.State, o.Action = readmePartCurrent, readmeActionKept
			o.Note = p.Name + " part current"
		default:
			out = concat(out[:start], renderReadmePart(p.Name, body, eol), out[end:])
			o.State, o.Action = readmePartStale, readmeActionRewritten
			o.Note = p.Name + " part rewritten"
		}
		outcomes = append([]readmePartOutcome{o}, outcomes...)
	}

	// Absent parts, in the fixed order, so each one placed is there for the
	// next one to stand beside.
	for _, name := range readmePartOrder {
		w, wanted := fresh[name]
		if !wanted || present[name] {
			continue
		}
		o := readmePartOutcome{Name: name, State: readmePartAbsent}
		place := placeReadmePart(out, name)
		if refused(name, place.inline) {
			o.Action, o.Note = readmeActionRefused, readmeBodyRefusal(name)
		} else {
			out = place.insert(out, renderReadmePart(name, w.bodyFor(place.inline), eol), eol)
			o.Action, o.Note = readmeActionWritten, name+" part written"
		}
		outcomes = append(outcomes, o)
	}
	res.Parts = outcomes

	// What was written must read back as written, or nothing is.
	check := parseReadmeParts(out)
	if len(check.faults) > 0 {
		return nil, readmePartsResult{}, fmt.Errorf("the composed README would not read back: %s", readmeFaultOutcome(check.faults[0]).Note)
	}
	for _, o := range outcomes {
		if o.Action != readmeActionWritten && o.Action != readmeActionRewritten {
			continue
		}
		p, _ := check.find(o.Name)
		if s, _ := check.classify(o.Name, fresh[o.Name].bodyFor(p.Inline)); s != readmePartCurrent {
			return nil, readmePartsResult{}, fmt.Errorf("the composed README would not read back: the %s part reads %s", o.Name, s)
		}
	}
	res.Wrote = !bytes.Equal(out, content)
	return out, res, nil
}

func readmeBodyRefusal(name string) string {
	return name + " part refused: its text would break its own marks; left as it is"
}

// readmeBodyHasBlankLine reports whether body holds a blank line.
func readmeBodyHasBlankLine(body string) bool {
	for _, ln := range strings.Split(body, "\n") {
		if strings.TrimSpace(ln) == "" {
			return true
		}
	}
	return false
}

// readmeBodyReadsBack reports whether body, rendered alone, reads back as one
// current part named name.
func readmeBodyReadsBack(name, body string) bool {
	doc := parseReadmeParts(renderReadmePart(name, body, []byte("\n")))
	if len(doc.faults) != 0 || len(doc.parts) != 1 {
		return false
	}
	p := doc.parts[0]
	return p.Name == name && p.hashMatches() && p.Body == body
}

func readmeFaultOutcome(f readmeFault) readmePartOutcome {
	o := readmePartOutcome{Name: f.Name, State: f.State, Action: readmeActionRefused, Lines: oneBased(f.Lines)}
	switch {
	case f.State == readmePartDuplicated:
		o.Note = fmt.Sprintf("%s part duplicated at lines %s; nothing written", f.Name, readmeLineList(o.Lines))
	case f.closer:
		o.Note = fmt.Sprintf("%s part's closing mark at line %d has no opening mark; nothing written", f.Name, o.Lines[0])
	default:
		o.Note = fmt.Sprintf("%s part's opening mark at line %d has no closing mark; nothing written", f.Name, o.Lines[0])
	}
	return o
}

// readmeLineList renders 3, 9 and 12 as "3, 9 and 12".
func readmeLineList(lines []int) string {
	words := make([]string, len(lines))
	for i, l := range lines {
		words[i] = strconv.Itoa(l)
	}
	if len(words) < 2 {
		return strings.Join(words, "")
	}
	return strings.Join(words[:len(words)-1], ", ") + " and " + words[len(words)-1]
}

// readmePlace is where an absent part goes: the byte offset it is inserted
// at, and whether it goes there in its inline form.
type readmePlace struct {
	at     int
	inline bool
}

// placeReadmePart places an absent part: beside the nearest present part in
// the fixed order, after it when that part comes earlier and before it when it
// comes later (at equal distance, after the earlier one); else after a
// leading frontmatter block; else after the first live level-one heading, a
// Markdown one or an HTML <h1> (readmeTitleEnd); else at the top. Unlike the
// conventions block, a part is never appended at the end, so it can never land
// inside a span nothing closes. Beside an inline part it is inline too, on the
// line next to it, so it joins the same HTML block; anywhere else it is a
// block.
func placeReadmePart(content []byte, name string) readmePlace {
	doc := parseReadmeParts(content)
	k := readmePartIndex(name)
	at, bestDist, bestIdx, inline := -1, 0, 0, false
	for _, p := range doc.parts {
		j := readmePartIndex(p.Name)
		if j < 0 || j == k {
			continue
		}
		dist := j - k
		if dist < 0 {
			dist = -dist
		}
		if at != -1 && (dist > bestDist || (dist == bestDist && bestIdx < k)) {
			continue
		}
		bestDist, bestIdx, inline = dist, j, p.Inline
		if j < k {
			at = doc.ends[p.End]
		} else {
			at = doc.starts[p.Begin]
		}
	}
	if at == -1 {
		if loc := frontmatterRe.FindIndex(content); loc != nil {
			at = loc[1]
		} else if end := readmeTitleEnd(content); end != -1 {
			at = end
		} else {
			at = 0
		}
	}
	return readmePlace{at: at, inline: inline}
}

// insert writes block at the place. An inline block takes a line of its own
// with no blank line on either side; any other is set off by one blank line
// from the text on each side.
func (pl readmePlace) insert(content, block, eol []byte) []byte {
	head, tail := content[:pl.at], content[pl.at:]
	if pl.inline {
		var pre []byte
		if len(head) > 0 && head[len(head)-1] != '\n' {
			pre = eol
		}
		return concat(head, pre, block, eol, tail)
	}
	var pre []byte
	if len(head) > 0 {
		if head[len(head)-1] != '\n' {
			pre = append(pre, eol...)
		}
		if !endsWithBlankLine(concat(head, pre)) {
			pre = append(pre, eol...)
		}
	}
	return concat(head, pre, block, trailingSep(tail, eol), tail)
}

var (
	// htmlH1Re opens an HTML level-one heading, after up to three spaces.
	htmlH1Re = regexp.MustCompile(`(?i)^ {0,3}<h1(?:[ \t>]|$)`)
	// htmlH1CloseRe closes one.
	htmlH1CloseRe = regexp.MustCompile(`(?i)</h1[ \t]*>`)
)

// readmeTitleEnd returns the byte offset just past the first live level-one
// heading: a Markdown `# ` line (as firstOutOfFenceH1 reads one) or an HTML
// <h1> line, past the line holding its closing tag when that follows before a
// blank line. A line in a code block or an HTML comment is not the title. It
// is -1 when there is none.
func readmeTitleEnd(content []byte) int {
	lines, starts, ends := markerLines(content)
	mask := mdrecord.Mask(lines)
	for i := range lines {
		if mask[i] != 0 {
			continue
		}
		if h1Re.Match(content[starts[i]:ends[i]]) {
			return ends[i]
		}
		if !htmlH1Re.MatchString(lines[i]) {
			continue
		}
		for j := i; j < len(lines) && (j == i || strings.TrimSpace(lines[j]) != ""); j++ {
			if htmlH1CloseRe.MatchString(lines[j]) {
				return ends[j]
			}
		}
		return ends[i]
	}
	return -1
}

// readmeInlineLines reports, per line of content as markerLines splits it,
// whether the line continues an open block-level HTML block: one opened by a
// block-level tag, or by a lone complete tag outside a paragraph (CommonMark's
// types 6 and 7), which runs to the next blank line. The forge renders no
// Markdown inside such a block, so a part whose opening fence is one of these
// lines is written in its inline form. The block's own first line is not a
// continuation, nor is a line in a leading frontmatter block, a code block, or
// a block of another type (a <pre>, a comment), and the lines are read as the
// forge's page reads them (readmePageLines).
func readmeInlineLines(content []byte) []bool {
	_, mstarts, _ := markerLines(content)
	out := make([]bool, len(mstarts))
	lines, starts := readmePageLines(content)
	mask := mdrecord.Mask(lines)
	fmEnd := 0
	if loc := frontmatterRe.FindIndex(content); loc != nil {
		fmEnd = loc[1]
	}
	cont := map[int]bool{} // a page line's start offset, for each continuation
	var block *readmeHTMLBlock
	para := false
	for i, ln := range lines {
		if starts[i] < fmEnd {
			continue
		}
		blank := strings.TrimSpace(ln) == ""
		if block != nil {
			if block.ends(ln, blank) {
				block, para = nil, false
				continue
			}
			if block.end == nil {
				cont[starts[i]] = true
			}
			continue
		}
		if mask[i] != 0 || blank {
			para = false
			continue
		}
		if b := openReadmeHTMLBlock(ln, para); b != nil {
			para = false
			if !b.ends(ln, false) {
				block = b
			}
			continue
		}
		// What is left is a heading, a rule or a setext underline, each
		// ending any paragraph; code indented outside one, which opens none;
		// or paragraph text, a list item's or a quotation's included.
		switch {
		case atxHeadingRe.MatchString(ln), thematicBreakRe.MatchString(ln), para && setextUnderlineRe.MatchString(ln):
			para = false
		case !para && indentedCodeRe.MatchString(ln):
		default:
			para = true
		}
	}
	for j, at := range mstarts {
		out[j] = cont[at]
	}
	return out
}

// endsWithBlankLine reports whether b, which ends with a line ending, ends
// with a blank line (or is one).
func endsWithBlankLine(b []byte) bool {
	b = bytes.TrimSuffix(b, []byte("\n"))
	b = bytes.TrimSuffix(b, []byte("\r"))
	if i := bytes.LastIndexByte(b, '\n'); i >= 0 {
		b = b[i+1:]
	}
	return len(bytes.TrimSpace(b)) == 0
}

// syncReadmeParts makes the README at path hold exactly the wanted parts (see
// planReadmeParts), creating the file when it is absent and something is to be
// written. A run in which every part is current writes nothing, so the file's
// bytes and modification time are unchanged. The read and the write hold the
// file's lock (withRewriteLock); a symlinked or non-regular README, one past
// maxAhoyFileBytes, or one the parts would grow past it, is refused and left
// untouched.
//
// setup calls it from stepReadme (readme_apply.go).
func syncReadmeParts(path string, want []readmePartWant) (readmePartsResult, error) {
	if err := markerFolderRefusal(path); err != nil {
		return readmePartsResult{}, err
	}
	var res readmePartsResult
	err := withRewriteLock(path, func() error {
		var serr error
		res, serr = syncReadmePartsLocked(path, want)
		return serr
	})
	return res, withoutFolder(path, err)
}

// removeReadmeParts takes out every part whose body still matches its hash and
// names every part edited by hand, leaving it. A README with no part is never
// written. A duplicated or unclosed fence leaves the file untouched.
//
// uninstall calls it from uninstallReadme (readme_apply.go).
func removeReadmeParts(path string) (readmePartsResult, error) {
	var res readmePartsResult
	err := withRewriteLock(path, func() error {
		var rerr error
		res, rerr = syncReadmePartsLocked(path, nil)
		return rerr
	})
	return res, withoutFolder(path, err)
}

// syncReadmePartsLocked is syncReadmeParts' read, change and write.
func syncReadmePartsLocked(path string, want []readmePartWant) (readmePartsResult, error) {
	out, res, err := planReadmeFile(path, want)
	if err != nil || !res.Wrote {
		return res, err
	}
	if werr := fsutil.WriteFileAtomicPreserveMode(path, out); werr != nil {
		return readmePartsResult{}, fmt.Errorf("it could not be written: %w", werr)
	}
	return res, nil
}

// planReadmeFile reads the README at path and plans the wanted parts into it
// (planReadmeParts), writing nothing: the content a write would leave, and
// what it would do. setup's early return asks it whether a run has work to
// do, and syncReadmePartsLocked writes what it plans.
func planReadmeFile(path string, want []readmePartWant) ([]byte, readmePartsResult, error) {
	var existing []byte
	absent := false
	fi, lerr := os.Lstat(path)
	switch {
	case lerr == nil && fi.Mode()&os.ModeSymlink != 0:
		return nil, readmePartsResult{}, &ahoyError{"it is a symlink, and abcd never writes through one"}
	case lerr == nil && !fi.Mode().IsRegular():
		return nil, readmePartsResult{}, &ahoyError{"it is not a regular file"}
	case lerr == nil:
		data, rerr := fsutil.ReadGuarded(path, maxAhoyFileBytes)
		if rerr != nil {
			return nil, readmePartsResult{}, fmt.Errorf("it could not be read: %w", rerr)
		}
		existing = data
	case os.IsNotExist(lerr):
		absent = true
	default:
		return nil, readmePartsResult{}, fmt.Errorf("it could not be examined: %w", lerr)
	}
	out, res, err := planReadmeParts(existing, want)
	if err != nil || !res.Wrote {
		return nil, res, err
	}
	// The read is capped, so a write past the cap would leave a file abcd can
	// no longer read, and so can no longer take its own part back out of
	// (iss-2610111552464704). Measured before the write; the file is left as
	// it was.
	if len(out) > maxAhoyFileBytes {
		return nil, readmePartsResult{}, &ahoyError{fmt.Sprintf("writing its parts would grow it to %d bytes, past abcd's %d-byte cap for the files it rewrites; left as it is", len(out), maxAhoyFileBytes)}
	}
	res.Created = absent
	return out, res, nil
}
