package capture

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
)

// kv is one ordered frontmatter entry. val is a string, int, or []string.
type kv struct {
	key string
	val any
}

// rawScalar is a frontmatter value written verbatim — no surrounding quotes —
// for a constrained, machine-read enum like `impact`. The frontmatter scanner is
// a line reader that compares the value byte-for-byte, so a quoted `"fix"` reads
// as a different string than the enum member `fix`; the enum fields must be bare.
// Only values already validated against their enum reach here, and yamlScalar
// still rejects any control char, so the verbatim path cannot inject a newline or
// a second frontmatter key.
type rawScalar string

// yamlScalar encodes a scalar value as a safe YAML literal, mirroring
// _issue_lib._yaml_scalar. Strings are double-quoted with backslash/dquote
// escaping and reject any ASCII control char (< 0x20); ints render bare.
func yamlScalar(value any) (string, error) {
	switch v := value.(type) {
	case bool:
		if v {
			return "true", nil
		}
		return "false", nil
	case int:
		return strconv.Itoa(v), nil
	case string:
		for _, r := range v {
			if r < 0x20 {
				return "", fmt.Errorf("%w: control char U+%04X in scalar string %q",
					ErrMalformedFrontmatter, r, v)
			}
		}
		esc := strings.ReplaceAll(v, `\`, `\\`)
		esc = strings.ReplaceAll(esc, `"`, `\"`)
		return `"` + esc + `"`, nil
	case rawScalar:
		// Verbatim, but never a value that could break the frontmatter shape: reject
		// any control char (a newline would inject a second key) and any character
		// outside a bare enum token. The caller has already validated the value
		// against its enum; this is defence in depth at the serialise boundary.
		//
		// The dot is admitted for shipped_in, whose value is a release tag
		// (v0.6.2) and which reShippedIn has already constrained to ^v[0-9.]+$
		// before it reaches here. It stays a bare token: nothing in this class can
		// open a quote, start a comment, or introduce a key.
		for _, r := range v {
			if r < 0x20 || !(r == '-' || r == '_' || r == '.' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
				return "", fmt.Errorf("%w: unsafe raw scalar %q", ErrMalformedFrontmatter, string(v))
			}
		}
		return string(v), nil
	default:
		return "", fmt.Errorf("%w: unsupported scalar type %T", ErrMalformedFrontmatter, value)
	}
}

// buildIssueText serialises an ordered field list + body into on-disk text,
// mirroring _issue_lib.build_issue_text: opening ---, one key: value line per
// entry, closing ---, exactly one blank separator, then body. No `...`
// end-marker. Empty list -> `key: []`; a list whose items are all abcd ids
// (itd-N/fn-N/iss-N) -> unquoted inline `[itd-4, fn-12]`; any other list ->
// per-item quoted inline. List items must all be strings (they are, by type).
func buildIssueText(fields []kv, body string) (string, error) {
	lines := []string{"---"}
	for _, f := range fields {
		switch v := f.val.(type) {
		case []string:
			enc, err := yamlList(v)
			if err != nil {
				return "", err
			}
			lines = append(lines, f.key+": "+enc)
		default:
			enc, err := yamlScalar(v)
			if err != nil {
				return "", err
			}
			lines = append(lines, f.key+": "+enc)
		}
	}
	lines = append(lines, "---")
	// The record ends with EXACTLY one trailing newline. The body arrives with an
	// inconsistent tail — some callers newline-terminate it, some do not — which is
	// how 170 of 174 ledger files came to lack an EOF newline (iss-175): the writer
	// appended the body verbatim. Normalise here so a freshly written record and one
	// the rewrite serialiser (setScalarField/setMapField, which preserve the body
	// byte-for-byte) later touches agree on the trailing newline. An empty body
	// keeps only the blank-line separator after the closing delimiter.
	body = strings.TrimRight(body, "\n")
	out := strings.Join(lines, "\n") + "\n\n" + body
	if body != "" {
		out += "\n"
	}
	return out, nil
}

// yamlList encodes a string list as the inline flow form buildIssueText has
// always written and parseScalarOrList reads back: `[]` when empty, bare ids
// when every item is an abcd id (`[itd-4, iss-12]`), per-item quoted otherwise.
// It is the ONE list encoder, shared by the create path and the in-place list
// rewrite (setListField), so a list a verb edits after capture is spelled
// exactly as a list capture wrote.
func yamlList(items []string) (string, error) {
	switch {
	case len(items) == 0:
		return "[]", nil
	case allAbcdIDs(items):
		return "[" + strings.Join(items, ", ") + "]", nil
	}
	parts := make([]string, len(items))
	for i, item := range items {
		enc, err := yamlScalar(item)
		if err != nil {
			return "", err
		}
		parts[i] = enc
	}
	return "[" + strings.Join(parts, ", ") + "]", nil
}

func allAbcdIDs(items []string) bool {
	for _, it := range items {
		if !reAbcdListID.MatchString(it) {
			return false
		}
	}
	return len(items) > 0
}

// setScalarField sets `key: <yamlScalar(value)>` inside content's frontmatter
// block, mirroring _issue_lib.set_scalar_field. If the key already exists at
// the top level it is replaced in place (order preserved); otherwise the new
// line is inserted before the closing ---. Only string/int values are accepted.
func setScalarField(content, key string, value any) (string, error) {
	if !reScalarKey.MatchString(key) {
		return "", fmt.Errorf("%w: invalid key %q", ErrMalformedFrontmatter, key)
	}
	encoded, err := yamlScalar(value)
	if err != nil {
		return "", err
	}

	lines := splitKeepEnds(content)
	openIdx, closeIdx, err := frontmatterBounds(lines)
	if err != nil {
		return "", err
	}

	eol := "\n"
	if strings.HasSuffix(lines[openIdx], "\r\n") {
		eol = "\r\n"
	}
	newLine := key + ": " + encoded + eol

	replaced := false
	for k := openIdx + 1; k < closeIdx; k++ {
		ln := lines[k]
		if strings.HasPrefix(ln, " ") || strings.HasPrefix(ln, "\t") {
			continue
		}
		bodyLn := strings.TrimRight(ln, "\r\n")
		if bodyLn == "" || strings.HasPrefix(bodyLn, "#") {
			continue
		}
		idx := strings.Index(bodyLn, ":")
		if idx < 0 {
			continue
		}
		if strings.TrimSpace(bodyLn[:idx]) == key {
			lines[k] = newLine
			replaced = true
			break
		}
	}
	if !replaced {
		lines = append(lines[:closeIdx], append([]string{newLine}, lines[closeIdx:]...)...)
	}
	return strings.Join(lines, ""), nil
}

// setListField sets `key: [<items>]` inside content's frontmatter block, the
// list sibling of setScalarField: an existing top-level key is replaced in place
// (order preserved), an absent one is inserted before the closing ---. An EMPTY
// list removes the key instead of writing `key: []`: the one caller (link, on an
// --unblock that empties blocked_by) is returning the record to the shape a
// capture that never named a blocker writes, and a leftover empty list would read
// as a field somebody set to nothing. The body is preserved byte-for-byte.
func setListField(content, key string, items []string) (string, error) {
	if !reScalarKey.MatchString(key) {
		return "", fmt.Errorf("%w: invalid key %q", ErrMalformedFrontmatter, key)
	}
	encoded, err := yamlList(items)
	if err != nil {
		return "", err
	}

	lines := splitKeepEnds(content)
	openIdx, closeIdx, err := frontmatterBounds(lines)
	if err != nil {
		return "", err
	}

	eol := "\n"
	if strings.HasSuffix(lines[openIdx], "\r\n") {
		eol = "\r\n"
	}
	newLine := key + ": " + encoded + eol

	for k := openIdx + 1; k < closeIdx; k++ {
		ln := lines[k]
		if strings.HasPrefix(ln, " ") || strings.HasPrefix(ln, "\t") {
			continue
		}
		bodyLn := strings.TrimRight(ln, "\r\n")
		if bodyLn == "" || strings.HasPrefix(bodyLn, "#") {
			continue
		}
		idx := strings.Index(bodyLn, ":")
		if idx < 0 || strings.TrimSpace(bodyLn[:idx]) != key {
			continue
		}
		if len(items) == 0 {
			lines = append(lines[:k], lines[k+1:]...)
		} else {
			lines[k] = newLine
		}
		return strings.Join(lines, ""), nil
	}
	if len(items) == 0 {
		return content, nil
	}
	lines = append(lines[:closeIdx], append([]string{newLine}, lines[closeIdx:]...)...)
	return strings.Join(lines, ""), nil
}

// nested marks a kv whose value is an ordered one-level object: transition
// routes it through setMapField instead of setScalarField.
type nested []kv

// setMapField inserts `key:` followed by an ordered, two-space-indented member
// block inside content's frontmatter, just before the closing --- (the
// deterministic nested-map writer beside setScalarField, spc-25). Members are
// encoded via yamlScalar — quoted strings, control chars refused — so the
// writer emits exactly the one-level object shape parseFrontmatterAndBody
// reads back. A key already present at the top level is refused (fail closed:
// replacing a nested block in place is not supported), as is an empty member
// list.
func setMapField(content, key string, members []kv) (string, error) {
	if !reScalarKey.MatchString(key) {
		return "", fmt.Errorf("%w: invalid key %q", ErrMalformedFrontmatter, key)
	}
	if len(members) == 0 {
		return "", fmt.Errorf("%w: setMapField needs at least one member for %q", ErrMalformedFrontmatter, key)
	}

	lines := splitKeepEnds(content)
	openIdx, closeIdx, err := frontmatterBounds(lines)
	if err != nil {
		return "", err
	}

	// Refuse a duplicate top-level key rather than replace: the one caller
	// (resolve) transitions from open/, where the key is never pre-set.
	for k := openIdx + 1; k < closeIdx; k++ {
		ln := lines[k]
		if strings.HasPrefix(ln, " ") || strings.HasPrefix(ln, "\t") {
			continue
		}
		bodyLn := strings.TrimRight(ln, "\r\n")
		if idx := strings.Index(bodyLn, ":"); idx >= 0 && strings.TrimSpace(bodyLn[:idx]) == key {
			return "", fmt.Errorf("%w: key %q already present", ErrMalformedFrontmatter, key)
		}
	}

	eol := "\n"
	if strings.HasSuffix(lines[openIdx], "\r\n") {
		eol = "\r\n"
	}
	block := []string{key + ":" + eol}
	for _, m := range members {
		if !reScalarKey.MatchString(m.key) {
			return "", fmt.Errorf("%w: invalid nested key %q", ErrMalformedFrontmatter, m.key)
		}
		s, ok := m.val.(string)
		if !ok {
			return "", fmt.Errorf("%w: nested member %q must be a string", ErrMalformedFrontmatter, m.key)
		}
		enc, err := yamlScalar(s)
		if err != nil {
			return "", err
		}
		block = append(block, "  "+m.key+": "+enc+eol)
	}
	out := make([]string, 0, len(lines)+len(block))
	out = append(out, lines[:closeIdx]...)
	out = append(out, block...)
	out = append(out, lines[closeIdx:]...)
	return strings.Join(out, ""), nil
}

// frontmatterBounds locates the leading frontmatter block's opening and closing
// delimiter lines in lines (as produced by splitKeepEnds, so ends are kept). The
// open must be the first non-empty line; the close is the next un-indented
// delimiter after it.
//
// The two rewrite paths (setScalarField, setMapField) held byte-identical copies
// of this scan, and both matched `---` byte-exact — the same divergence from the
// canonical reader that parseFrontmatterAndBody carried (GitHub #338). One copy,
// one delimiter rule (frontmatter.IsDelimiter), so a record capture can read is a
// record capture can also rewrite: a reader that accepts bytes its writer refuses
// is the same split verdict pointing the other way.
func frontmatterBounds(lines []string) (openIdx, closeIdx int, err error) {
	openIdx, closeIdx = -1, -1
	for i, ln := range lines {
		// A BOM is only a BOM at the file's first line; past it, U+FEFF is an
		// ordinary character and must not make a body line a delimiter
		// (iss-2608270926036966). Trimming it at i == 0 keeps this writer able to
		// rewrite exactly the BOM'd records parseFrontmatterAndBody accepts.
		if i == 0 {
			ln = frontmatter.TrimBOM(ln)
		}
		if strings.TrimRight(ln, "\r\n") == "" {
			continue
		}
		if frontmatter.IsDelimiter(ln) {
			openIdx = i
			break
		}
		return -1, -1, fmt.Errorf("%w: content has no frontmatter block", ErrMalformedFrontmatter)
	}
	if openIdx == -1 {
		return -1, -1, fmt.Errorf("%w: content has no frontmatter block", ErrMalformedFrontmatter)
	}
	for j := openIdx + 1; j < len(lines); j++ {
		ln := lines[j]
		if strings.HasPrefix(ln, " ") || strings.HasPrefix(ln, "\t") {
			continue
		}
		if frontmatter.IsDelimiter(ln) {
			closeIdx = j
			break
		}
	}
	if closeIdx == -1 {
		return -1, -1, fmt.Errorf("%w: frontmatter not terminated", ErrMalformedFrontmatter)
	}
	return openIdx, closeIdx, nil
}

// splitKeepEnds splits s into lines preserving their trailing newline(s),
// mirroring Python's str.splitlines(keepends=True) for \n and \r\n.
func splitKeepEnds(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i+1])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}
