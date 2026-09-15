package capture

import (
	"fmt"
	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"strconv"
	"strings"
)

// parseFrontmatterAndBody splits text into a frontmatter map and a body,
// mirroring _issue_lib._parse_text_with_body. The text MUST start with an
// opening --- line; the next un-indented --- closes the block. At most one
// leading blank line after the closing delimiter is stripped from the body.
//
// Delimiter lines are recognised by frontmatter.IsDelimiter, the ONE rule every
// reader of these bytes shares (GitHub #338). This parser used to match `---`
// byte-exact while record-lint's ledger gate and the lifeboat graveyard read the
// same file through the canonical trimming scanner, so a `--- ` close made a
// lint-green record that every capture verb refused. The strictness that IS
// deliberate here — the restricted YAML subset, indented lines refused,
// duplicate top-level keys refused — is about the block's CONTENT and is
// untouched; only the delimiter compare moves to the shared primitive.
//
// The frontmatter is parsed with a restricted YAML subset matching what
// buildIssueText/setScalarField emit: top-level `key: value` scalars, inline
// lists (`[]`, `[itd-4, fn-12]`, `["a", "b"]`), and a single level of nested
// object (used only by the optional resolved_by field). Values decode to
// string, int, []string, or map[string]any.
func parseFrontmatterAndBody(text string) (map[string]any, string, error) {
	lines := splitKeepEnds(text)
	// TrimBOM applies to lines[0] and nowhere else: U+FEFF is a byte-order mark
	// only at the file's first position, and a mid-file "\ufeff---" is an
	// ordinary body line (iss-2608270926036966).
	if len(lines) == 0 || !frontmatter.IsDelimiter(frontmatter.TrimBOM(lines[0])) {
		return nil, "", fmt.Errorf("%w: frontmatter must start with '---' on the first line", ErrMalformedFrontmatter)
	}
	closeIdx := -1
	for i := 1; i < len(lines); i++ {
		ln := lines[i]
		if strings.HasPrefix(ln, " ") || strings.HasPrefix(ln, "\t") {
			continue
		}
		if frontmatter.IsDelimiter(ln) {
			closeIdx = i
			break
		}
	}
	if closeIdx == -1 {
		return nil, "", fmt.Errorf("%w: frontmatter not terminated: missing closing '---'", ErrMalformedFrontmatter)
	}

	body := strings.Join(lines[closeIdx+1:], "")
	switch {
	case strings.HasPrefix(body, "\r\n"):
		body = body[2:]
	case strings.HasPrefix(body, "\n"):
		body = body[1:]
	}

	fm, err := parseFrontmatterBlock(lines[1:closeIdx])
	if err != nil {
		return nil, "", err
	}
	return fm, body, nil
}

// parseFrontmatterBlock parses the interior lines of a frontmatter block.
// nextLineIsIndented reports whether the line after i begins an indented block,
// skipping blank lines. It is the lookahead that separates `key:` as a null from
// `key:` as the head of a nested object.
func nextLineIsIndented(lines []string, i int) bool {
	for j := i + 1; j < len(lines); j++ {
		l := strings.TrimRight(lines[j], "\r\n")
		if strings.TrimSpace(l) == "" {
			continue
		}
		return strings.HasPrefix(l, " ") || strings.HasPrefix(l, "\t")
	}
	return false
}

// quotedScalar reports whether a raw scalar token is double-quoted, which in
// YAML makes it a string whatever it spells.
func quotedScalar(rest string) bool {
	t := strings.TrimSpace(rest)
	return len(t) >= 2 && strings.HasPrefix(t, `"`) && strings.HasSuffix(t, `"`)
}

// parseFrontmatterBlock parses the interior lines of a frontmatter block,
// deciding null-vs-string while the RAW scalar is still in hand.
//
// An earlier draft split this in two and threaded a set of quoted keys out to the
// validator. The decision is made inline here, so that set was never populated and
// both callers discarded it — dead scaffolding, and the comment justifying it was
// false as well. One function, no reserved return.
func parseFrontmatterBlock(lines []string) (map[string]any, error) {
	fm := map[string]any{}
	i := 0
	for i < len(lines) {
		raw := strings.TrimRight(lines[i], "\r\n")
		if strings.TrimSpace(raw) == "" || strings.HasPrefix(strings.TrimSpace(raw), "#") {
			i++
			continue
		}
		if strings.HasPrefix(raw, " ") || strings.HasPrefix(raw, "\t") {
			return nil, fmt.Errorf("%w: unexpected indented line %q", ErrMalformedFrontmatter, raw)
		}
		idx := strings.Index(raw, ":")
		if idx < 0 {
			return nil, fmt.Errorf("%w: line is not key: value %q", ErrMalformedFrontmatter, raw)
		}
		key := strings.TrimSpace(raw[:idx])
		// A trailing comment is not part of the value, and the rule is the shared
		// scanner's — the same call record-lint's reader makes. A strip on one
		// side alone would put the permissive verdict on the gate: this parser
		// refusing `severity: minor # todo` as out-of-enum while the gate that
		// exists to refuse exactly what this parser refuses read it as `minor`
		// (iss-2608301744268001).
		rest := strings.TrimSpace(frontmatter.StripComment(raw[idx+1:]))
		if key == "" {
			return nil, fmt.Errorf("%w: empty key in %q", ErrMalformedFrontmatter, raw)
		}
		// Reject a duplicated top-level key. Last-wins here is a silent trap: the
		// reader would keep the SECOND value while setScalarField (status
		// transitions) rewrites only the FIRST occurrence, so the file the reader
		// sees and the file a mutation edits diverge. A duplicate key is malformed.
		if _, dup := fm[key]; dup {
			return nil, fmt.Errorf("%w: duplicate key %q", ErrMalformedFrontmatter, key)
		}
		// `key:` with no value and NO indented line under it is a bare YAML null,
		// not an empty object (iss-285). Taking the nested branch there made the
		// value a map, which capture rejects as "must be a string" while
		// record-lint reads the same raw scalar as null — the split verdict this
		// normalisation exists to close, left open for the one spelling nobody
		// tested. The lookahead is what parts them: an object has a member.
		if rest == "" && !nextLineIsIndented(lines, i) {
			fm[key] = ""
			i++
			continue
		}
		if rest == "" {
			// Nested one-level object: consume following indented lines.
			sub := map[string]any{}
			i++
			for i < len(lines) {
				subRaw := strings.TrimRight(lines[i], "\r\n")
				if !strings.HasPrefix(subRaw, " ") && !strings.HasPrefix(subRaw, "\t") {
					break
				}
				subTrim := strings.TrimSpace(subRaw)
				if subTrim == "" {
					i++
					continue
				}
				sidx := strings.Index(subTrim, ":")
				if sidx < 0 {
					return nil, fmt.Errorf("%w: nested line is not key: value %q", ErrMalformedFrontmatter, subRaw)
				}
				sval, err := parseScalarOrList(strings.TrimSpace(frontmatter.StripComment(subTrim[sidx+1:])))
				if err != nil {
					return nil, err
				}
				subKey := strings.TrimSpace(subTrim[:sidx])
				if _, dup := sub[subKey]; dup {
					return nil, fmt.Errorf("%w: duplicate nested key %q", ErrMalformedFrontmatter, subKey)
				}
				sub[subKey] = sval
				i++
			}
			fm[key] = sub
			continue
		}
		val, err := parseScalarOrList(rest)
		if err != nil {
			return nil, err
		}
		// Normalise a BARE YAML null to the empty string, and leave a quoted one
		// as the string it is (iss-285).
		//
		// parseScalarOrList unquotes, which destroys the only thing separating
		// `impact: null` from `impact: "null"` — and record-lint tests the RAW
		// scalar, so the two gates reached opposite verdicts on one record: the
		// lint saw a string and refused, capture saw a null and passed. That is
		// the shape where a record passes the lint and then fails the command
		// that acts on it. Widening IsNull to the whole YAML null set (iss-287)
		// makes the split worse rather than better, which is why the two fixes
		// land together.
		//
		// Collapsing to "" here means every downstream nullness test is a test on
		// a value that already knows which it was, without threading quotedness
		// through validate and its callers.
		if str, isStr := val.(string); isStr && !quotedScalar(rest) && frontmatter.IsNull(str) {
			val = ""
		}
		fm[key] = val
		i++
	}
	return fm, nil
}

// parseScalarOrList decodes one YAML value into string, int, or []string.
func parseScalarOrList(s string) (any, error) {
	if s == "[]" {
		return []string{}, nil
	}
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		inner := strings.TrimSpace(s[1 : len(s)-1])
		if inner == "" {
			return []string{}, nil
		}
		var out []string
		for _, part := range splitInlineListItems(inner) {
			item := strings.TrimSpace(part)
			dec, err := decodeScalar(item)
			if err != nil {
				return nil, err
			}
			str, ok := dec.(string)
			if !ok {
				str = fmt.Sprint(dec)
			}
			out = append(out, str)
		}
		return out, nil
	}
	return decodeScalar(s)
}

// splitInlineListItems splits the interior of an inline list on top-level
// commas, honouring the double-quoting and backslash escaping that yamlScalar
// emits: a comma inside a quoted item (or an escaped comma/quote) is not a
// separator. Keeping the tokenizer symmetric with the serializer is what lets a
// quoted item containing a comma — e.g. synthesis_clusters: ["design review,
// session 3"] — round-trip faithfully instead of being split mid-item with
// stray quote characters left behind.
func splitInlineListItems(inner string) []string {
	var items []string
	var cur strings.Builder
	inQuote := false
	esc := false
	for _, r := range inner {
		switch {
		case esc:
			cur.WriteRune(r)
			esc = false
		case r == '\\':
			cur.WriteRune(r)
			esc = true
		case r == '"':
			cur.WriteRune(r)
			inQuote = !inQuote
		case r == ',' && !inQuote:
			items = append(items, cur.String())
			cur.Reset()
		default:
			cur.WriteRune(r)
		}
	}
	items = append(items, cur.String())
	return items
}

// decodeScalar decodes a single non-list scalar token.
func decodeScalar(s string) (any, error) {
	if strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) && len(s) >= 2 {
		return frontmatter.Unquote(s[1 : len(s)-1]), nil
	}
	if n, err := strconv.Atoi(s); err == nil {
		return n, nil
	}
	// Bare token (unquoted string, e.g. an abcd id or a legacy value).
	return s, nil
}
