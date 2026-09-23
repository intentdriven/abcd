package memory

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// maxPageValueBytes caps one rendered committed-page field — an index.md,
// contradictions.md or log.md value (schema.go's cleanPageField). It is generous
// — every real value (a filename, a domain, a summary, a slug, a class label)
// sits far below it — and exists only so a hostile distiller cannot make one
// rendered field unbounded. It is applied by CleanProse, which also masks the
// terminal control/bidi/zero-width runes a `cat`/`git diff`/pager over the
// committed derived page would otherwise replay (iss-2608270655495573). The
// frontmatter scalar path (dumpString/doubleQuote) masks in place instead, to
// keep its exact-round-trip escaping of \n \t \r intact.
const maxPageValueBytes = 8192

// yaml.go — a stdlib-only YAML frontmatter parser/producer scoped to the shapes
// the memory store uses (mirrors scripts/abcd/_yaml.py). It is a reader/writer
// pair: parseFrontmatter(joinFileFrontmatter(dumpFrontmatter(D), "")) round-trips
// any map D the producer admits. Supported shapes: scalars, inline flow-lists
// [a, b], inline flow-maps { k: v }, one-level nested dicts, and a block list of
// bare-dash flow-map/dict items (the multi-source source.sources shape).

type yamlError struct{ Msg string }

func (e *yamlError) Error() string { return e.Msg }

func yamlErrf(format string, a ...any) *yamlError {
	return &yamlError{Msg: fmt.Sprintf(format, a...)}
}

var keyRe = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_-]*)\s*:\s*(.*)`)

// isFrontmatterClose reports whether line closes a frontmatter block. The two
// ends of the same block are held to one rule: the open is matched after a
// whitespace trim, so the close trims too. A trailing space or tab on the close
// ("--- ") is a common editor artefact, and comparing it byte-exact made the
// parsers report "frontmatter not terminated" and every reader fall back to its
// empty default — silently dropping the page's source: provenance and the lint
// gates that read it. This is the rule the canonical primitive already applies
// (internal/core/frontmatter.Fields). Callers normalise \r\n -> \n first, so
// only spaces and tabs remain to trim.
func isFrontmatterClose(line string) bool {
	return strings.TrimRight(line, " \t") == "---"
}

// normaliseNewlines folds \r\n and \r to \n, the first step of every parser here
// (so a CRLF delimiter is seen as a delimiter — iss-30).
func normaliseNewlines(text string) string {
	return strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
}

// frontmatterOpenIndex returns the index of the line that opens the frontmatter
// block, skipping the tolerated preamble (leading HTML-comment lines, and blank
// lines after the first), and whether an opening delimiter was found. When it
// was not, the index is the offending line — which is len(lines) when the scan
// ran off the end. This is the single scan the parsers below share, and the one
// gate a caller must ask instead of testing the leading bytes of the document.
//
// The first line is stripped of a UTF-8 byte-order mark through the canonical
// frontmatter.TrimBOM before it is tested: U+FEFF is not Unicode White_Space, so
// strings.TrimSpace keeps it, and a BOM-led page would otherwise have frontmatter
// to record-lint and none to memory (iss-2608291814565781). Only the first line
// is stripped — a U+FEFF anywhere else is content, not a byte-order mark.
func frontmatterOpenIndex(lines []string) (int, bool) {
	start := 0
	for start < len(lines) {
		line := lines[start]
		if start == 0 {
			line = frontmatter.TrimBOM(line)
		}
		s := strings.TrimSpace(line)
		if s == "---" {
			return start, true
		}
		if strings.HasPrefix(s, "<!--") || strings.HasSuffix(s, "-->") || (start > 0 && s == "") {
			start++
			continue
		}
		return start, false
	}
	return start, false
}

// textOpensFrontmatter reports whether text opens a frontmatter block by the
// parsers' own rule. A reader that instead tests strings.HasPrefix(text, "---")
// disagrees with them on exactly the pages they tolerate — one led by an HTML
// comment, or opening on an indented delimiter — and treats a page that carries
// declared provenance as though it carried none.
func textOpensFrontmatter(text string) bool {
	_, ok := frontmatterOpenIndex(strings.Split(normaliseNewlines(text), "\n"))
	return ok
}

// ---------------------------------------------------------------------------
// Parse
// ---------------------------------------------------------------------------

// parseFrontmatter extracts and parses the YAML frontmatter at the top of text
// (between --- delimiters, tolerating leading HTML-comment / blank lines).
func parseFrontmatter(text string) (map[string]any, error) {
	lines := strings.Split(normaliseNewlines(text), "\n")
	start, ok := frontmatterOpenIndex(lines)
	if !ok {
		if start < len(lines) {
			return nil, yamlErrf("line %d: unexpected content before frontmatter '---': %q", start+1, lines[start])
		}
		return nil, yamlErrf("document must start with '---'")
	}
	// Extract inner lines up to closing ---.
	var fm []string
	i := start + 1
	found := false
	for i < len(lines) {
		if isFrontmatterClose(lines[i]) {
			found = true
			break
		}
		fm = append(fm, lines[i])
		i++
	}
	if !found {
		return nil, yamlErrf("frontmatter not terminated: missing closing '---'")
	}
	return parseYAMLLines(fm)
}

func parseYAMLLines(lines []string) (map[string]any, error) {
	result := map[string]any{}
	i := 0
	n := len(lines)
	for i < n {
		raw := lines[i]
		stripped := strings.TrimSpace(raw)
		if stripped == "" || strings.HasPrefix(stripped, "#") {
			i++
			continue
		}
		if indentOf(raw) != 0 {
			return nil, yamlErrf("top-level key must have zero indent: %q", raw)
		}
		key, rest, ok := splitKeyValue(raw)
		if !ok {
			return nil, yamlErrf("unexpected content: %q", raw)
		}
		restT := strings.TrimSpace(rest)
		if restT == "|" {
			block, ni := collectBlockScalar(lines, i+1)
			result[key] = strings.Join(block, "\n")
			i = ni
			continue
		}
		if restT != "" {
			v, err := parseScalarOrInline(restT)
			if err != nil {
				return nil, err
			}
			result[key] = v
			i++
			continue
		}
		// No inline value — look ahead.
		if i+1 >= n {
			result[key] = nil
			i++
			continue
		}
		next := lines[i+1]
		nextS := strings.TrimSpace(next)
		if nextS == "" || strings.HasPrefix(nextS, "#") {
			result[key] = nil
			i++
			continue
		}
		nextIndent := indentOf(next)
		if strings.HasPrefix(nextS, "- ") || nextS == "-" {
			items, ni, err := collectBlockList(lines, i+1, nextIndent)
			if err != nil {
				return nil, err
			}
			result[key] = items
			i = ni
			continue
		}
		if nextIndent > 0 {
			nested, ni, err := collectNestedDict(lines, i+1, nextIndent)
			if err != nil {
				return nil, err
			}
			result[key] = nested
			i = ni
			continue
		}
		result[key] = nil
		i++
	}
	return result, nil
}

func collectBlockScalar(lines []string, start int) ([]string, int) {
	blockIndent := -1
	var collected []string
	i := start
	for i < len(lines) {
		raw := lines[i]
		if strings.TrimSpace(raw) == "" {
			collected = append(collected, "")
			i++
			continue
		}
		indent := indentOf(raw)
		if blockIndent < 0 {
			// The body of a block scalar must be indented DEEPER than the "key: |"
			// line that opened it. collectBlockScalar is only ever reached from
			// parseYAMLLines, which refuses any key line that is not at zero indent,
			// so "deeper" means an indent greater than zero. A first content line at
			// column 0 therefore belongs to the document, not to the block: the block
			// is empty and the line is left unconsumed for the top-level parse.
			// Taking it as content would silently swallow every remaining line —
			// including other keys — with no error.
			if indent == 0 {
				break
			}
			blockIndent = indent
		}
		if indent < blockIndent {
			break
		}
		if blockIndent > 0 && len(raw) >= blockIndent {
			collected = append(collected, raw[blockIndent:])
		} else {
			collected = append(collected, raw)
		}
		i++
	}
	for len(collected) > 0 && collected[len(collected)-1] == "" {
		collected = collected[:len(collected)-1]
	}
	return collected, i
}

func collectBlockList(lines []string, start, expectedIndent int) ([]any, int, error) {
	var items []any
	i := start
	n := len(lines)
	for i < n {
		raw := lines[i]
		stripped := strings.TrimSpace(raw)
		if stripped == "" || strings.HasPrefix(stripped, "#") {
			i++
			continue
		}
		indent := indentOf(raw)
		if indent < expectedIndent {
			break
		}
		if indent > expectedIndent {
			return nil, 0, yamlErrf("unexpected indent in block list (expected %d, got %d)", expectedIndent, indent)
		}
		if !(strings.HasPrefix(stripped, "- ") || stripped == "-") {
			break
		}
		itemText := ""
		if strings.HasPrefix(stripped, "- ") {
			itemText = strings.TrimSpace(stripped[2:])
		}
		switch {
		case strings.HasPrefix(itemText, "{"):
			m, err := parseFlowMap(itemText)
			if err != nil {
				return nil, 0, err
			}
			items = append(items, m)
			i++
		case strings.HasPrefix(itemText, "["):
			l, err := parseFlowList(itemText)
			if err != nil {
				return nil, 0, err
			}
			items = append(items, l)
			i++
		case itemText != "":
			v, err := parseScalar(itemText)
			if err != nil {
				return nil, 0, err
			}
			items = append(items, v)
			i++
		default:
			// Bare dash — a nested dict follows at deeper indent.
			if i+1 < n {
				next := lines[i+1]
				nextS := strings.TrimSpace(next)
				nextIndent := indentOf(next)
				if nextIndent > expectedIndent && nextS != "" {
					nested, ni, err := collectNestedDict(lines, i+1, nextIndent)
					if err != nil {
						return nil, 0, err
					}
					items = append(items, nested)
					i = ni
					continue
				}
			}
			items = append(items, nil)
			i++
		}
	}
	return items, i, nil
}

func collectNestedDict(lines []string, start, expectedIndent int) (map[string]any, int, error) {
	result := map[string]any{}
	i := start
	n := len(lines)
	for i < n {
		raw := lines[i]
		stripped := strings.TrimSpace(raw)
		if stripped == "" || strings.HasPrefix(stripped, "#") {
			i++
			continue
		}
		indent := indentOf(raw)
		if indent < expectedIndent {
			break
		}
		if indent > expectedIndent {
			return nil, 0, yamlErrf("deeper nesting not supported (indent %d > %d)", indent, expectedIndent)
		}
		key, rest, ok := splitKeyValue(raw)
		if !ok {
			return nil, 0, yamlErrf("expected key: value in nested dict: %q", raw)
		}
		if strings.TrimSpace(rest) != "" {
			v, err := parseScalarOrInline(strings.TrimSpace(rest))
			if err != nil {
				return nil, 0, err
			}
			result[key] = v
			i++
			continue
		}
		// Key with no inline value — look ahead for a block list nested under it.
		if i+1 < n {
			next := lines[i+1]
			nextS := strings.TrimSpace(next)
			nextIndent := indentOf(next)
			if nextS != "" && !strings.HasPrefix(nextS, "#") &&
				(strings.HasPrefix(nextS, "- ") || nextS == "-") && nextIndent > expectedIndent {
				items, ni, err := collectBlockList(lines, i+1, nextIndent)
				if err != nil {
					return nil, 0, err
				}
				result[key] = items
				i = ni
				continue
			}
		}
		result[key] = nil
		i++
	}
	return result, i, nil
}

// ---------------------------------------------------------------------------
// Inline flow parsers
// ---------------------------------------------------------------------------

func parseFlowList(text string) ([]any, error) {
	text = strings.TrimSpace(text)
	if !(strings.HasPrefix(text, "[") && strings.HasSuffix(text, "]")) {
		return nil, yamlErrf("invalid flow-list: %q", text)
	}
	inner := strings.TrimSpace(text[1 : len(text)-1])
	if inner == "" {
		return []any{}, nil
	}
	toks, err := splitFlowSequence(inner)
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, len(toks))
	for _, tok := range toks {
		t := strings.TrimSpace(tok)
		switch {
		case strings.HasPrefix(t, "{"):
			m, err := parseFlowMap(t)
			if err != nil {
				return nil, err
			}
			items = append(items, m)
		case strings.HasPrefix(t, "["):
			return nil, yamlErrf("deeper inline nesting not supported (nested [] inside flow-list)")
		default:
			v, err := parseScalar(t)
			if err != nil {
				return nil, err
			}
			items = append(items, v)
		}
	}
	return items, nil
}

func parseFlowMap(text string) (map[string]any, error) {
	text = strings.TrimSpace(text)
	if !(strings.HasPrefix(text, "{") && strings.HasSuffix(text, "}")) {
		return nil, yamlErrf("invalid flow-map: %q", text)
	}
	inner := strings.TrimSpace(text[1 : len(text)-1])
	if inner == "" {
		return map[string]any{}, nil
	}
	result := map[string]any{}
	pairs, err := splitFlowSequence(inner)
	if err != nil {
		return nil, err
	}
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		cp := findUnquotedColon(pair)
		if cp < 0 {
			return nil, yamlErrf("missing ':' in flow-map pair: %q", pair)
		}
		k, err := parseMapKey(strings.TrimSpace(pair[:cp]))
		if err != nil {
			return nil, err
		}
		v := strings.TrimSpace(pair[cp+1:])
		if strings.HasPrefix(v, "{") || strings.HasPrefix(v, "[") {
			return nil, yamlErrf("deeper inline nesting not supported for key %q", k)
		}
		sv, err := parseScalar(v)
		if err != nil {
			return nil, err
		}
		result[k] = sv
	}
	return result, nil
}

var bracketPair = map[byte]byte{'}': '{', ']': '['}

func splitFlowSequence(inner string) ([]string, error) {
	var tokens []string
	var stack []byte
	inSingle, inDouble, escapeNext := false, false, false
	var current []byte
	for i := 0; i < len(inner); i++ {
		ch := inner[i]
		if escapeNext {
			current = append(current, ch)
			escapeNext = false
			continue
		}
		if ch == '\\' && inDouble {
			current = append(current, ch)
			escapeNext = true
			continue
		}
		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			current = append(current, ch)
		} else if ch == '"' && !inSingle {
			inDouble = !inDouble
			current = append(current, ch)
		} else if !inSingle && !inDouble {
			switch {
			case ch == '{' || ch == '[':
				stack = append(stack, ch)
				current = append(current, ch)
			case ch == '}' || ch == ']':
				if len(stack) == 0 {
					return nil, yamlErrf("unexpected closing bracket %q in flow sequence", string(ch))
				}
				if stack[len(stack)-1] != bracketPair[ch] {
					return nil, yamlErrf("mismatched brackets in flow sequence")
				}
				stack = stack[:len(stack)-1]
				current = append(current, ch)
			case ch == ',' && len(stack) == 0:
				tokens = append(tokens, string(current))
				current = nil
			default:
				current = append(current, ch)
			}
		} else {
			current = append(current, ch)
		}
	}
	if inSingle {
		return nil, yamlErrf("unterminated single-quoted string in flow sequence")
	}
	if inDouble {
		return nil, yamlErrf("unterminated double-quoted string in flow sequence")
	}
	if len(stack) > 0 {
		return nil, yamlErrf("unclosed bracket in flow sequence")
	}
	if len(current) > 0 {
		tokens = append(tokens, string(current))
	}
	return tokens, nil
}

// parseMapKey resolves a flow-map key token to its string value, unquoting a
// double- or single-quoted key so it round-trips with dumpFlowMap (which quotes
// keys carrying YAML metacharacters). A bare key is returned verbatim — keys are
// not run through parseScalar, so a bare "true"/"123" stays a string key.
func parseMapKey(text string) (string, error) {
	if text == "" {
		return "", yamlErrf("empty flow-map key")
	}
	if text[0] == '"' {
		if len(text) < 2 || text[len(text)-1] != '"' {
			return "", yamlErrf("unterminated double-quoted key: %q", text)
		}
		return unescapeDoubleQuoted(text[1 : len(text)-1])
	}
	if text[0] == '\'' {
		if len(text) < 2 || text[len(text)-1] != '\'' {
			return "", yamlErrf("unterminated single-quoted key: %q", text)
		}
		return strings.ReplaceAll(text[1:len(text)-1], "''", "'"), nil
	}
	return text, nil
}

func findUnquotedColon(text string) int {
	inSingle, inDouble, escapeNext := false, false, false
	for i := 0; i < len(text); i++ {
		ch := text[i]
		// Honour backslash escapes inside a double-quoted key exactly as
		// splitFlowSequence does — otherwise an escaped quote (\") in a quoted key
		// mis-toggles the in-string state and the real key/value colon is missed.
		if escapeNext {
			escapeNext = false
			continue
		}
		if ch == '\\' && inDouble {
			escapeNext = true
			continue
		}
		switch {
		case ch == '\'' && !inDouble:
			inSingle = !inSingle
		case ch == '"' && !inSingle:
			inDouble = !inDouble
		case ch == ':' && !inSingle && !inDouble:
			return i
		}
	}
	return -1
}

// ---------------------------------------------------------------------------
// Scalars
// ---------------------------------------------------------------------------

var (
	intRe   = regexp.MustCompile(`^-?\d+$`)
	floatRe = regexp.MustCompile(`^-?\d+\.\d*$`)
)

func parseScalarOrInline(text string) (any, error) {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "[") {
		return parseFlowList(text)
	}
	if strings.HasPrefix(text, "{") {
		return parseFlowMap(text)
	}
	return parseScalar(text)
}

func parseScalar(text string) (any, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil
	}
	switch text {
	case "null", "~", "Null", "NULL":
		return nil, nil
	case "true", "True", "TRUE":
		return true, nil
	case "false", "False", "FALSE":
		return false, nil
	}
	if text[0] == '"' {
		if len(text) < 2 || text[len(text)-1] != '"' {
			return nil, yamlErrf("unterminated double-quoted string: %q", text)
		}
		return unescapeDoubleQuoted(text[1 : len(text)-1])
	}
	if text[0] == '\'' {
		if len(text) < 2 || text[len(text)-1] != '\'' {
			return nil, yamlErrf("unterminated single-quoted string: %q", text)
		}
		return strings.ReplaceAll(text[1:len(text)-1], "''", "'"), nil
	}
	if intRe.MatchString(text) {
		if v, err := strconv.Atoi(text); err == nil {
			return v, nil
		}
	}
	if floatRe.MatchString(text) {
		if v, err := strconv.ParseFloat(text, 64); err == nil {
			return v, nil
		}
	}
	return text, nil
}

func unescapeDoubleQuoted(s string) (string, error) {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' {
			if i+1 >= len(s) {
				return "", yamlErrf("dangling backslash at end of double-quoted string")
			}
			switch s[i+1] {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			case '"':
				b.WriteByte('"')
			case '\\':
				b.WriteByte('\\')
			default:
				return "", yamlErrf("unsupported escape sequence '\\%c'", s[i+1])
			}
			i++
		} else {
			b.WriteByte(s[i])
		}
	}
	return b.String(), nil
}

func splitKeyValue(raw string) (string, string, bool) {
	m := keyRe.FindStringSubmatch(strings.TrimSpace(raw))
	if m == nil {
		return "", "", false
	}
	return m[1], m[2], true
}

func indentOf(s string) int {
	return len(s) - len(strings.TrimLeft(s, " \t"))
}

// ---------------------------------------------------------------------------
// File-level split / join
// ---------------------------------------------------------------------------

// splitFileFrontmatter splits full file text into (yaml region, body). Line
// endings are normalised to \n first (parser parity, below); the region is the
// text between the --- delimiters (each inner line newline-terminated); body is
// everything after the closing --- with normalised newlines.
func splitFileFrontmatter(text string) (string, string, error) {
	// Normalise line endings first, exactly as parseFrontmatter and
	// frontmatterKeyLine do — otherwise a CRLF closing delimiter ("---\r")
	// never equals "---" and the whole document is wrongly rejected, diverging
	// from the parser that accepts it and degrading hashes/summaries (iss-30).
	text = normaliseNewlines(text)
	lines := strings.Split(text, "\n")
	start, ok := frontmatterOpenIndex(lines)
	if !ok {
		if start < len(lines) {
			return "", "", yamlErrf("unexpected content before frontmatter '---': %q", lines[start])
		}
		return "", "", yamlErrf("document must start with '---' frontmatter")
	}
	var fm []string
	i := start + 1
	for i < len(lines) {
		if isFrontmatterClose(lines[i]) {
			var region strings.Builder
			for _, l := range fm {
				region.WriteString(l)
				region.WriteByte('\n')
			}
			body := strings.Join(lines[i+1:], "\n")
			return region.String(), body, nil
		}
		fm = append(fm, lines[i])
		i++
	}
	return "", "", yamlErrf("frontmatter not terminated: missing closing '---'")
}

// joinFileFrontmatter is the inverse of splitFileFrontmatter.
func joinFileFrontmatter(region, body string) string {
	if region != "" && !strings.HasSuffix(region, "\n") {
		region += "\n"
	}
	return "---\n" + region + "---\n" + body
}

// frontmatterKeyLine returns the 1-based file line of top-level key, or 1 when
// absent — a best-effort positional helper for lint findings (never errors).
func frontmatterKeyLine(text, key string) int {
	lines := strings.Split(normaliseNewlines(text), "\n")
	start, ok := frontmatterOpenIndex(lines)
	if !ok {
		return 1
	}
	out := 0
	for i := start + 1; i < len(lines); i++ {
		raw := lines[i]
		if isFrontmatterClose(raw) {
			break
		}
		if raw != "" && raw[0] != ' ' && raw[0] != '\t' {
			if m := keyRe.FindStringSubmatch(raw); m != nil && m[1] == key {
				out = i + 1
			}
		}
	}
	if out == 0 {
		return 1
	}
	return out
}

// ---------------------------------------------------------------------------
// Dump — deterministic (keys sorted at every level)
// ---------------------------------------------------------------------------

var needsQuoteChars = ":#[]{}&*!|>'\"%@`,"

// bareBlockKeyRe matches a map key the block-level parsers (splitKeyValue's keyRe,
// collectNestedDict) can round-trip when emitted verbatim: an identifier, no YAML
// metacharacters. A distiller-supplied key that fails this cannot be represented
// as a bare block key, so the dumper rejects it (fail closed) rather than emit a
// line that re-parses to a different key or breaks the read entirely.
var bareBlockKeyRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)

// checkBlockKey rejects a block-level map key that is not round-trip-safe.
func checkBlockKey(k string) error {
	if !bareBlockKeyRe.MatchString(k) {
		return yamlErrf("map key %q is not a valid frontmatter key (identifier chars only)", k)
	}
	return nil
}

// dumpFrontmatter serialises fm to a deterministic block-style YAML region (no
// --- delimiters, trailing \n). Keys are sorted so output is stable regardless
// of Go map iteration order. Round-trips through parseFrontmatter.
func dumpFrontmatter(fm map[string]any) (string, error) {
	var lines []string
	for _, key := range sortedKeys(fm) {
		l, err := dumpValueLines(key, fm[key])
		if err != nil {
			return "", err
		}
		lines = append(lines, l...)
	}
	var b strings.Builder
	for _, l := range lines {
		b.WriteString(l)
		b.WriteByte('\n')
	}
	return b.String(), nil
}

func dumpValueLines(key string, value any) ([]string, error) {
	if err := checkBlockKey(key); err != nil {
		return nil, err
	}
	switch v := value.(type) {
	case []any:
		if len(v) == 0 {
			return []string{key + ": []"}, nil
		}
		lines := []string{key + ":"}
		for _, item := range v {
			switch it := item.(type) {
			case map[string]any:
				fm, err := dumpFlowMap(it)
				if err != nil {
					return nil, err
				}
				lines = append(lines, "- "+fm)
			case []any:
				return nil, yamlErrf("nested lists not supported in block list for key %q", key)
			default:
				s, err := dumpScalar(item)
				if err != nil {
					return nil, err
				}
				lines = append(lines, "- "+s)
			}
		}
		return lines, nil
	case []string:
		return dumpValueLines(key, toAnySlice(v))
	case map[string]any:
		if len(v) == 0 {
			return []string{key + ": {}"}, nil
		}
		lines := []string{key + ":"}
		for _, k := range sortedKeys(v) {
			l, err := dumpNestedPairLines(key, k, v[k])
			if err != nil {
				return nil, err
			}
			lines = append(lines, l...)
		}
		return lines, nil
	default:
		s, err := dumpScalar(value)
		if err != nil {
			return nil, err
		}
		return []string{key + ": " + s}, nil
	}
}

func dumpNestedPairLines(parentKey, k string, v any) ([]string, error) {
	if err := checkBlockKey(k); err != nil {
		return nil, err
	}
	switch val := v.(type) {
	case map[string]any:
		if len(val) == 0 {
			return []string{"  " + k + ": {}"}, nil
		}
		fm, err := dumpFlowMap(val)
		if err != nil {
			return nil, err
		}
		return []string{"  " + k + ": " + fm}, nil
	case []string:
		return dumpNestedPairLines(parentKey, k, toAnySlice(val))
	case []any:
		if len(val) == 0 {
			return []string{"  " + k + ": []"}, nil
		}
		allScalar := true
		allDict := true
		for _, item := range val {
			if _, ok := item.(map[string]any); ok {
				allScalar = false
			} else {
				allDict = false
			}
		}
		if allScalar {
			fl, err := dumpFlowList(val)
			if err != nil {
				return nil, err
			}
			return []string{"  " + k + ": " + fl}, nil
		}
		if !allDict {
			return nil, yamlErrf("block list under %q.%q must be all-scalar or all-dict items", parentKey, k)
		}
		lines := []string{"  " + k + ":"}
		for _, item := range val {
			m := item.(map[string]any)
			if len(m) == 0 {
				return nil, yamlErrf("empty dict item not serialisable in block list under %q.%q", parentKey, k)
			}
			lines = append(lines, "    -")
			for _, ik := range sortedKeys(m) {
				if err := checkBlockKey(ik); err != nil {
					return nil, err
				}
				iv := m[ik]
				switch inner := iv.(type) {
				case map[string]any:
					fm, err := dumpFlowMap(inner)
					if err != nil {
						return nil, err
					}
					lines = append(lines, "      "+ik+": "+fm)
				case []string:
					fl, err := dumpFlowList(toAnySlice(inner))
					if err != nil {
						return nil, err
					}
					lines = append(lines, "      "+ik+": "+fl)
				case []any:
					fl, err := dumpFlowList(inner)
					if err != nil {
						return nil, err
					}
					lines = append(lines, "      "+ik+": "+fl)
				default:
					s, err := dumpScalar(iv)
					if err != nil {
						return nil, err
					}
					lines = append(lines, "      "+ik+": "+s)
				}
			}
		}
		return lines, nil
	default:
		s, err := dumpScalar(v)
		if err != nil {
			return nil, err
		}
		return []string{"  " + k + ": " + s}, nil
	}
}

func dumpFlowMap(d map[string]any) (string, error) {
	if len(d) == 0 {
		return "{}", nil
	}
	var parts []string
	for _, k := range sortedKeys(d) {
		v := d[k]
		if _, ok := v.(map[string]any); ok {
			return "", yamlErrf("deeper nesting not supported in flow-map value for key %q", k)
		}
		if _, ok := v.([]any); ok {
			return "", yamlErrf("deeper nesting not supported in flow-map value for key %q", k)
		}
		s, err := dumpScalar(v)
		if err != nil {
			return "", err
		}
		// Quote the KEY through the same path as the value: a citation key that
		// carries a YAML metacharacter (":", ",", "}", newline) would otherwise
		// re-parse to a different key or break the read. parseFlowMap unquotes it
		// on the way back, so the round-trip holds.
		parts = append(parts, dumpString(k)+": "+s)
	}
	return "{ " + strings.Join(parts, ", ") + " }", nil
}

func dumpFlowList(items []any) (string, error) {
	var parts []string
	for _, item := range items {
		if _, ok := item.(map[string]any); ok {
			return "", yamlErrf("deeper nesting not supported in flow-list item")
		}
		if _, ok := item.([]any); ok {
			return "", yamlErrf("deeper nesting not supported in flow-list item")
		}
		s, err := dumpScalar(item)
		if err != nil {
			return "", err
		}
		parts = append(parts, s)
	}
	return "[" + strings.Join(parts, ", ") + "]", nil
}

func dumpScalar(value any) (string, error) {
	switch v := value.(type) {
	case nil:
		return "null", nil
	case bool:
		if v {
			return "true", nil
		}
		return "false", nil
	case int:
		return strconv.Itoa(v), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case float64:
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10), nil
		}
		return strconv.FormatFloat(v, 'g', -1, 64), nil
	case string:
		return dumpString(v), nil
	default:
		return "", yamlErrf("unsupported scalar type for YAML dump: %T", value)
	}
}

func dumpString(s string) string {
	if s == "" {
		return `""`
	}
	switch s {
	case "null", "~", "Null", "NULL", "true", "True", "TRUE", "false", "False", "FALSE":
		return doubleQuote(s)
	}
	if intRe.MatchString(s) || floatRe.MatchString(s) {
		return doubleQuote(s)
	}
	if s != strings.TrimSpace(s) {
		return doubleQuote(s)
	}
	if strings.ContainsAny(s, needsQuoteChars) {
		return doubleQuote(s)
	}
	if strings.HasPrefix(s, "-") || strings.HasPrefix(s, "?") {
		return doubleQuote(s)
	}
	// Line-structure characters. \r belongs here with \n: parseFrontmatter
	// normalises \r and \r\n to \n before splitting, so a bare \r emitted raw
	// becomes a real line break on read — enough to end the frontmatter early
	// (a "---" after it) and drop every later key into the page body. doubleQuote
	// escapes all three.
	if strings.ContainsAny(s, "\n\r\t") {
		return doubleQuote(s)
	}
	// A value carrying a terminal-display attack rune that is NOT one of the
	// line-structure characters above — an ESC, a C1 control, a bidi override, a
	// zero-width or DEL — reaches this point unquoted. dumpString would emit it raw
	// and a `cat`/`git diff`/pager over the committed page would replay it (the
	// write residue of #250/#262's read-path fix, iss-2608270655495573). Route it
	// through doubleQuote, whose default arm masks exactly those runes via the
	// canonical termsafe.Sanitize while the \n \t \r ESCAPING above is left intact —
	// so the provenance/exact-round-trip invariant for the line-structure characters
	// is untouched. termsafe.Sanitize(s) differs from s here only for an attack rune,
	// since s already contains no \n \r \t.
	if termsafe.Sanitize(s) != s {
		return doubleQuote(s)
	}
	return s
}

func doubleQuote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, ch := range s {
		switch ch {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		case '\r':
			b.WriteString(`\r`)
		default:
			// Every other rune is emitted as-is EXCEPT the terminal-display attack
			// runes (C0/DEL, C1, bidi overrides, zero-width), which are masked to a
			// visible '?' via the canonical sanitiser rather than written raw into
			// the committed page (iss-2608270655495573). \n \t \r never reach here —
			// their explicit arms above escape them losslessly — so the exact
			// round-trip of the line-structure characters is preserved.
			b.WriteString(termsafe.Sanitize(string(ch)))
		}
	}
	b.WriteByte('"')
	return b.String()
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func toAnySlice(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}
