package scanner

import "strings"

// maxPercentDecodePasses bounds the percent-decode pre-pass. One pass reverses a
// single layer of URL encoding (%3D -> '='); a second reaches a double-encoded
// delimiter (%253D -> %3D -> '='); the third is slack. The alternating chain
// (alternatingLayers) is bounded separately at four passes and may spend all of
// them on one decoder, so it reads one percent layer deeper than this pre-pass.
// The bound is deliberate:
// each pass strictly shrinks the string (three bytes collapse to one) so a fixed
// point is reached quickly, and capping the passes keeps a crafted deeply-nested
// input from turning one line into unbounded work. A token buried under more
// layers than this stays raw — the same bounded-work trade the adjacency probes
// already make — but the realistic OAuth/redirect/magic-link leak this closes is
// single- or double-encoded.
const maxPercentDecodePasses = 3

// decodedLineFindings recovers tokens a percent-encoded delimiter hid from the
// leading \b. When a URL/JSON body is URL-encoded, every non-word delimiter
// becomes a %XX triple whose final byte is a hex digit — itself a word char —
// so the byte immediately before a literal token is a word char and the leading
// \b every bundled token pattern anchors on can never hold. The token text is
// unreserved and stays literal in the clear, so decoding the delimiters and
// re-scanning the decoded copy finds it (gh-370).
//
// Each hit on the decoded copy is mapped back to its byte span in the ORIGINAL
// raw line, so the Finding a caller redacts masks the LIVE token where it
// actually sits on disk — the delimiter bytes are left untouched, only the
// secret is masked. Skip/SkipAt run against the decoded line (the docs-example
// AWS key and the like are suppressed there just as on a plaintext line).
//
// Callers get this automatically: ScanText folds these findings in alongside the
// raw-line findings, so every consumer of the one canonical scanner — the
// history transcript store, the issue-ledger redactor, the lifeboat packer and
// the launch bundler — inherits the coverage from a single definition.
//
// BOTH detector families run over the decoded copy: the token matchers
// (scanAllPatterns) AND the identity matchers (matchers.findings — home path,
// $HOME backstop, email, name, username). A percent-encoded home path or email
// (`%2Fhome%2Falice`, `%61lice%40host`) never appears literally on the raw line,
// so it defeats every identity matcher there too; running them on the decoded
// copy and mapping each hit back to its raw span is what stops such an identity
// leak surviving into a committed memory/intent/capture artifact
// (iss-2608270720336165).
//
// The glued sweep (glued.go) runs over each decoded view too: a token glued
// behind a word byte whose OWN bytes are escaped (`notes_%67hp_…`, a JSON
// \u escape of its first letter) is whole only on the decoded view, and there
// the bounded patterns' leading \b cannot hold (iss-2609290743362554).
func decodedLineFindings(patterns []Pattern, probes []matcher, junctions junctionSet, glued gluedSweep, matchers identityMatchers, id2sev map[string]Severity, rawLine string, lineno int, file string) []Finding {
	var out []Finding
	for _, v := range lineViews(rawLine) {
		out = append(out, viewFindings(patterns, probes, junctions, glued, matchers, id2sev, rawLine, v, lineno, file)...)
	}
	return out
}

// lineViews is every decoded view of one line the scan reads, and the one
// place that list is made: the percent pre-pass's fully decoded copy, then
// each layer of the line's JSON string escapes (jsonescape.go), outermost
// first, so a value an escape hid from an anchor or spelled with escaped
// bytes is found where it sits on disk (iss-2609261647358395,
// iss-2609251639263391). The literal-home backstop (residual.go) and, through
// DecodedViews, the committed-text lint rules read the same list.
//
// The two decoders also run over each other's output, alternating to a fixed
// point, so a value spelled with both stacked is read: a JSON escape of the
// percent sign (\u0025 before two hex digits) becomes a percent escape only
// once the JSON layer is decoded, and a percent encoding of the backslash
// (%5C before u0067) becomes a JSON escape only once the percent view is,
// and either can be stacked on the other again (iss-2610090821491948). Two
// chains run, one opening with each decoder; each chain decodes one pass at a
// time, turning to the other decoder after every pass and staying with the
// same one only when the other decodes nothing, and stops at a pass that
// changes nothing or at maxAlternatingLayers. Every pass of each chain is a
// view, mapped back to the raw line, and a view whose text an earlier one
// already holds is dropped. A spelling that needs more than four passes in
// all stays raw: the bounded-work residual, the same trade the layer caps
// make, and the work per line stays a constant number of linear passes.
func lineViews(line string) []decodedView {
	var views []decodedView
	if decoded, posMap := percentDecodeBounded(line); posMap != nil {
		views = append(views, decodedView{decoded, posMap})
	}
	views = append(views, jsonEscapeLayers(line)...)
	seen := make(map[string]bool, len(views))
	for _, v := range views {
		seen[v.text] = true
	}
	for _, percentFirst := range []bool{true, false} {
		for _, v := range alternatingLayers(line, percentFirst) {
			if !seen[v.text] {
				seen[v.text] = true
				views = append(views, v)
			}
		}
	}
	return views
}

// maxAlternatingLayers bounds one alternating chain: four passes in all, each
// one percent pass or one JSON layer, enough for a value stacked three or four
// decodes deep in either order.
const maxAlternatingLayers = 4

// alternatingLayers is one chain of lineViews' alternation, opening with the
// percent decoder or the JSON one. A chain whose opening decoder changes
// nothing is empty, because the other chain is the one that opens there.
func alternatingLayers(s string, percentFirst bool) []decodedView {
	var out []decodedView
	cur := s
	var m []int // offsets in cur -> offsets in s; nil means identity
	percent := percentFirst
	for layer := 0; layer < maxAlternatingLayers; layer++ {
		next, step, ok := decodeLayerOnce(cur, percent)
		if !ok && layer > 0 {
			percent = !percent
			next, step, ok = decodeLayerOnce(cur, percent)
		}
		if !ok {
			break
		}
		composed := make([]int, len(next)+1)
		for i := range composed {
			if m == nil {
				composed[i] = step[i]
			} else {
				composed[i] = m[step[i]]
			}
		}
		out = append(out, decodedView{text: next, posMap: composed})
		cur, m = next, composed
		percent = !percent
	}
	return out
}

// decodeLayerOnce runs one pass of the percent decoder or one layer of the
// JSON one over s, reporting whether it changed anything.
func decodeLayerOnce(s string, percent bool) (string, []int, bool) {
	if percent {
		if strings.IndexByte(s, '%') < 0 {
			return s, nil, false
		}
		next, step := percentDecodeOnce(s)
		return next, step, next != s
	}
	if strings.IndexByte(s, '\\') < 0 {
		return s, nil, false
	}
	return jsonUnescapeOnce(s)
}

// DecodedViews returns the decoded spellings of one line that the scan reads
// beside the line as written (lineViews), without their position maps: nil
// for a line with nothing to decode. It is the seam for a surface that judges
// committed lines with its own matchers — the repolint privacy rule and the
// harness_leak lint rule — so a JSON fixture, export or transcript that
// writes a home path, an address or a session URL behind an escape is read
// in the spelling the store-before-commit redactors read it in
// (iss-2609261658553101). A decoded view can carry line breaks the escapes
// stood for; a line-scoped check reads each one as the start of a line.
func DecodedViews(line string) []string {
	views := lineViews(line)
	if len(views) == 0 {
		return nil
	}
	out := make([]string, len(views))
	for i, v := range views {
		out[i] = v.text
	}
	return out
}

// viewFindings runs every detector over one decoded view of rawLine and maps
// each hit back to the raw bytes it came from. A view with nothing decoded in
// it is never handed here; the raw scan already covers the raw line.
func viewFindings(patterns []Pattern, probes []matcher, junctions junctionSet, glued gluedSweep, matchers identityMatchers, id2sev map[string]Severity, rawLine string, v decodedView, lineno int, file string) []Finding {
	decoded, posMap := v.text, v.posMap
	out := viewTokenFindings(patterns, probes, junctions, rawLine, v, lineno, file)
	if len(glued.patterns) > 0 {
		out = append(out, viewTokenFindings(glued.patterns, glued.probes, glued.junctions, rawLine, v, lineno, file)...)
	}
	// Identity matchers over the decoded copy. matchers.findings runs its whole
	// suppression discipline (URL spans, home/email suppression of a username,
	// docs allowlists) self-consistently on the DECODED line; each surviving
	// finding is then re-homed onto the raw line by mapping its decoded byte span
	// (Column-1 .. +len(Matched)) back through posMap, exactly as the token path
	// above maps its match. The remapped Matched is the LIVE encoded bytes on
	// disk, so Redact masks the token where it actually sits and the delimiter
	// bytes around it are left intact.
	for _, f := range matchers.findings(decoded, lineno, id2sev, file) {
		dStart := f.Column - 1
		dEnd := dStart + len(f.Matched)
		if dStart < 0 || dEnd > len(decoded) {
			continue
		}
		rawStart, rawEnd, ok := mapDecodedSpan(posMap, dStart, dEnd, len(rawLine))
		if !ok {
			continue
		}
		f.Column = rawStart + 1
		f.Matched = rawLine[rawStart:rawEnd]
		f.Snippet = snippet(rawLine)
		f.line = rawLine
		out = append(out, f)
	}
	return out
}

// viewTokenFindings runs one pattern set over one decoded view of rawLine,
// applies each pattern's Skip and SkipAt on the decoded text, and maps every
// surviving hit back to the raw bytes it came from. The bounded patterns and
// the glued sweep's boundary-free set both run through it.
func viewTokenFindings(patterns []Pattern, probes []matcher, junctions junctionSet, rawLine string, v decodedView, lineno int, file string) []Finding {
	decoded, posMap := v.text, v.posMap
	var out []Finding
	for _, m := range scanAllPatterns(patterns, probes, junctions, decoded) {
		cp := patterns[m.patIdx]
		matchedDecoded := decoded[m.start:m.end]
		scanMeter.charge(stageSkip, len(matchedDecoded))
		if cp.Skip != nil && cp.Skip(matchedDecoded) {
			continue
		}
		if cp.SkipAt != nil && cp.SkipAt(decoded, m.start, m.end) {
			continue
		}
		rawStart, rawEnd, ok := mapDecodedSpan(posMap, m.start, m.end, len(rawLine))
		if !ok {
			continue
		}
		out = append(out, Finding{
			File: file, Line: lineno, Column: rawStart + 1, Kind: cp.Kind,
			Severity: cp.Severity, Snippet: snippet(rawLine), Matched: rawLine[rawStart:rawEnd],
			Suggested: cp.Suggestion, line: rawLine,
		})
	}
	return out
}

// mapDecodedSpan translates a half-open [start,end) byte span on the decoded
// copy back to the corresponding span on the original raw line via posMap,
// returning ok=false when the mapped span is degenerate or out of range (so a
// caller drops the finding rather than slicing rawLine out of bounds). posMap
// carries a sentinel at len(decoded), so a match's half-open end maps cleanly.
func mapDecodedSpan(posMap []int, start, end, rawLen int) (int, int, bool) {
	if start < 0 || end >= len(posMap) {
		return 0, 0, false
	}
	rawStart, rawEnd := posMap[start], posMap[end]
	if rawStart < 0 || rawEnd > rawLen || rawStart >= rawEnd {
		return 0, 0, false
	}
	return rawStart, rawEnd, true
}

// percentDecodeBounded percent-decodes s up to maxPercentDecodePasses times and
// returns the fully-decoded string together with a position map: posMap[i] is
// the byte offset into the ORIGINAL s at which decoded byte i began, with a
// sentinel posMap[len(decoded)] == len(s) so a match's half-open end maps back
// cleanly. It returns (s, nil) when s carries no decodable %XX sequence, so the
// caller can skip the decoded scan entirely on the common (unencoded) line.
func percentDecodeBounded(s string) (string, []int) {
	cur := s
	// m maps a byte offset in cur back to a byte offset in the original s.
	m := make([]int, len(s)+1)
	for i := range m {
		m[i] = i
	}
	changed := false
	for pass := 0; pass < maxPercentDecodePasses; pass++ {
		next, step := percentDecodeOnce(cur)
		if next == cur {
			break
		}
		changed = true
		composed := make([]int, len(next)+1)
		for i := 0; i <= len(next); i++ {
			composed[i] = m[step[i]]
		}
		cur, m = next, composed
	}
	if !changed {
		return s, nil
	}
	return cur, m
}

// percentDecodeOnce decodes one layer of %XX sequences in s, returning the
// decoded bytes and a position map from each decoded byte offset back to the
// offset in s it came from (with a trailing sentinel == len(s)). A '%' not
// followed by two hex digits is copied literally.
func percentDecodeOnce(s string) (string, []int) {
	scanMeter.charge(stagePercent, len(s))
	b := make([]byte, 0, len(s))
	pos := make([]int, 0, len(s)+1)
	for i := 0; i < len(s); {
		if s[i] == '%' && i+2 < len(s) && isHexDigit(s[i+1]) && isHexDigit(s[i+2]) {
			pos = append(pos, i)
			b = append(b, hexNibble(s[i+1])<<4|hexNibble(s[i+2]))
			i += 3
			continue
		}
		pos = append(pos, i)
		b = append(b, s[i])
		i++
	}
	pos = append(pos, len(s))
	return string(b), pos
}

func hexNibble(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	default: // A-F (isHexDigit already vetted the byte)
		return c - 'A' + 10
	}
}

// dedupFindings drops findings that are byte-for-byte the same span, kind and
// file — the decode pre-pass re-finds a plaintext token that also sat on a line
// carrying an unrelated %XX sequence, and a doubled finding would double-count
// in the write path's audit tally. Order is preserved; the first of each span
// wins.
func dedupFindings(findings []Finding) []Finding {
	if len(findings) < 2 {
		return findings
	}
	type key struct {
		file, kind   string
		line, column int
		matchLen     int
	}
	seen := make(map[key]struct{}, len(findings))
	out := findings[:0:0]
	for _, f := range findings {
		k := key{f.File, f.Kind, f.Line, f.Column, len(f.Matched)}
		if _, dup := seen[k]; dup {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, f)
	}
	return out
}
