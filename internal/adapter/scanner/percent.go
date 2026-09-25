package scanner

// maxPercentDecodePasses bounds the percent-decode pre-pass. One pass reverses a
// single layer of URL encoding (%3D -> '='); a second reaches a double-encoded
// delimiter (%253D -> %3D -> '='); the third is slack. The bound is deliberate:
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
func decodedLineFindings(patterns []Pattern, probes []matcher, junctions junctionSet, matchers identityMatchers, id2sev map[string]Severity, rawLine string, lineno int, file string) []Finding {
	decoded, posMap := percentDecodeBounded(rawLine)
	if posMap == nil {
		return nil // nothing was percent-encoded; the raw scan already covers it
	}
	var out []Finding
	for _, m := range scanAllPatterns(patterns, probes, junctions, decoded) {
		cp := patterns[m.patIdx]
		matchedDecoded := decoded[m.start:m.end]
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
