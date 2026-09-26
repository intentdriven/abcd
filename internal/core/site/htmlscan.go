package site

// The strict tokenizer over the generator's own output.
//
// `abcd lint site` has to walk the emitted pages and say, of every visible
// word, where it came from. That is a claim about the WHOLE document, so a
// reader that skips what it does not understand cannot make it: a text node
// inside markup the reader silently dropped is a text node the provenance walk
// never asked about, and the check would pass by not looking.
//
// So this is not an HTML parser. It reads the subset THIS generator emits —
// double-quoted attributes, balanced elements, a fixed void set, inlined SVG,
// and the five entities `escapeText`/`escapeAttr` produce — and refuses
// everything else by name and position. It is the same posture
// `checkInlinableSVG` takes in assets.go, for the same reason: the bytes being
// read become the published page, and a lenient reader of a published page is a
// reader that agrees with whatever it is given.
//
// Refused on sight: comments, processing instructions, CDATA, doctypes other
// than the leading `<!doctype html>`, unquoted or single-quoted attribute
// values, an end tag that closes something else, an element left open at the
// end of the file, a bare `<` in text, and any entity outside the XML five and
// numeric character references.

import (
	"fmt"
	"strings"
)

// htmlFault is a document the tokenizer refuses.
type htmlFault struct {
	File   string
	Line   int
	Reason string
}

func (e *htmlFault) Error() string {
	return fmt.Sprintf("%s:%d: %s — the check reads the subset this generator emits and refuses the rest, "+
		"because a reader that skips what it does not understand cannot say where every word came from",
		e.File, e.Line, e.Reason)
}

// htmlKind distinguishes the two node shapes the walk cares about.
type htmlKind int

const (
	htmlElementNode htmlKind = iota
	htmlTextNode
)

// htmlNode is one element or one run of text.
type htmlNode struct {
	Kind htmlKind
	// Name is the element's tag name, lower-cased for HTML and left as written
	// for the SVG elements whose names are case-sensitive (clipPath, linearGradient).
	Name string
	// Attrs are the element's attributes, keyed as written. A valueless
	// attribute (defer, hidden) maps to the empty string.
	Attrs map[string]string
	// Text is a text node's content with its entities decoded.
	Text string
	// Line is the 1-based line the node starts on.
	Line     int
	Parent   *htmlNode
	Children []*htmlNode
}

// The elements that close themselves are `voidElements` in markdown.go — the
// renderer's own table, deliberately reused rather than copied. A void element
// added to what the generator emits is then known to the reader that checks it,
// which a second list beside the first would not be.

// htmlRawText are the elements whose content is not markup. The generator emits
// `<script src=… defer></script>` with an empty body and no `<style>` at all,
// but reading their content as markup would be wrong the moment either grows a
// body, so both are read as raw text.
var htmlRawText = map[string]bool{"script": true, "style": true}

// Attr reads an attribute, or the empty string.
func (n *htmlNode) Attr(name string) string {
	if n == nil || n.Attrs == nil {
		return ""
	}
	return n.Attrs[name]
}

// Classes splits the element's class attribute.
func (n *htmlNode) Classes() []string { return strings.Fields(n.Attr("class")) }

// HasClass reports whether the element carries a class.
func (n *htmlNode) HasClass(c string) bool {
	for _, got := range n.Classes() {
		if got == c {
			return true
		}
	}
	return false
}

// Ancestors yields the node's ancestors, nearest first.
func (n *htmlNode) Ancestors() []*htmlNode {
	var out []*htmlNode
	for p := n.Parent; p != nil; p = p.Parent {
		out = append(out, p)
	}
	return out
}

// InsideElement reports whether any ancestor is one of the named elements.
func (n *htmlNode) InsideElement(names ...string) bool {
	for _, a := range n.Ancestors() {
		for _, name := range names {
			if a.Name == name {
				return true
			}
		}
	}
	return false
}

// Walk visits every node depth-first in document order, the root included.
func (n *htmlNode) Walk(fn func(*htmlNode)) {
	if n == nil {
		return
	}
	fn(n)
	for _, c := range n.Children {
		c.Walk(fn)
	}
}

// Text returns the concatenated text of a subtree, entities decoded.
func (n *htmlNode) TextContent() string {
	var b strings.Builder
	n.Walk(func(x *htmlNode) {
		if x.Kind == htmlTextNode {
			b.WriteString(x.Text)
		}
	})
	return b.String()
}

// parseHTML reads one emitted page into a tree. The returned root is a synthetic
// document node whose children are the doctype's element and nothing else.
func parseHTML(file, src string) (*htmlNode, error) {
	p := &htmlParser{file: file, src: src, line: 1}
	return p.parse()
}

type htmlParser struct {
	file string
	src  string
	i    int
	line int
}

func (p *htmlParser) fault(reason string) error {
	return &htmlFault{File: p.file, Line: p.line, Reason: reason}
}

// advance moves the cursor to j, counting the lines it crossed.
func (p *htmlParser) advance(j int) {
	p.line += strings.Count(p.src[p.i:j], "\n")
	p.i = j
}

func (p *htmlParser) parse() (*htmlNode, error) {
	root := &htmlNode{Kind: htmlElementNode, Name: "#document", Line: 1}
	stack := []*htmlNode{root}
	sawDoctype := false

	for p.i < len(p.src) {
		if p.src[p.i] != '<' {
			j := strings.IndexByte(p.src[p.i:], '<')
			end := len(p.src)
			if j >= 0 {
				end = p.i + j
			}
			raw := p.src[p.i:end]
			text, err := p.decodeEntities(raw, true)
			if err != nil {
				return nil, err
			}
			node := &htmlNode{Kind: htmlTextNode, Text: text, Line: p.line}
			parent := stack[len(stack)-1]
			node.Parent = parent
			parent.Children = append(parent.Children, node)
			p.advance(end)
			continue
		}

		switch {
		case strings.HasPrefix(p.src[p.i:], "<!--"):
			return nil, p.fault("an HTML comment; the generator emits none, and a comment is where unreviewed text hides")
		case strings.HasPrefix(p.src[p.i:], "<![CDATA["):
			return nil, p.fault("a CDATA section")
		case strings.HasPrefix(p.src[p.i:], "<?"):
			return nil, p.fault("a processing instruction")
		case strings.HasPrefix(p.src[p.i:], "<!"):
			if sawDoctype || len(stack) != 1 {
				return nil, p.fault("a document-type declaration outside the top of the file")
			}
			end := strings.IndexByte(p.src[p.i:], '>')
			if end < 0 {
				return nil, p.fault("an unterminated declaration")
			}
			decl := strings.ToLower(strings.TrimSpace(p.src[p.i : p.i+end+1]))
			if decl != "<!doctype html>" {
				return nil, p.fault("the doctype is " + quote(decl) + ", not <!doctype html>")
			}
			sawDoctype = true
			p.advance(p.i + end + 1)
			continue
		case strings.HasPrefix(p.src[p.i:], "</"):
			name, next, err := p.endTag()
			if err != nil {
				return nil, err
			}
			if len(stack) == 1 {
				return nil, p.fault("</" + name + "> closes an element that is not open")
			}
			open := stack[len(stack)-1]
			if open.Name != name {
				return nil, p.fault("</" + name + "> closes <" + open.Name + ">, opened on line " + itoaLine(open.Line))
			}
			stack = stack[:len(stack)-1]
			p.advance(next)
			continue
		}

		node, selfClosing, next, err := p.startTag()
		if err != nil {
			return nil, err
		}
		parent := stack[len(stack)-1]
		node.Parent = parent
		parent.Children = append(parent.Children, node)
		p.advance(next)

		if selfClosing || voidElements[node.Name] {
			continue
		}
		if htmlRawText[node.Name] {
			close := "</" + node.Name + ">"
			at := indexFoldASCII(p.src[p.i:], close)
			if at < 0 {
				return nil, p.fault("<" + node.Name + "> is never closed")
			}
			if body := p.src[p.i : p.i+at]; strings.TrimSpace(body) != "" {
				node.Children = append(node.Children, &htmlNode{
					Kind: htmlTextNode, Text: body, Line: p.line, Parent: node,
				})
			}
			p.advance(p.i + at + len(close))
			continue
		}
		stack = append(stack, node)
	}

	if len(stack) != 1 {
		open := stack[len(stack)-1]
		return nil, &htmlFault{File: p.file, Line: open.Line, Reason: "<" + open.Name + "> is never closed"}
	}
	return root, nil
}

// endTag reads `</name>` and returns the name and the index just past it.
func (p *htmlParser) endTag() (string, int, error) {
	j := p.i + 2
	start := j
	for j < len(p.src) && isTagNameByte(p.src[j], j == start) {
		j++
	}
	name := p.src[start:j]
	if name == "" {
		return "", 0, p.fault("an end tag with no name")
	}
	for j < len(p.src) && isSpace(p.src[j]) {
		j++
	}
	if j >= len(p.src) || p.src[j] != '>' {
		return "", 0, p.fault("</" + name + " is not closed by '>'")
	}
	return normalizeTagName(name), j + 1, nil
}

// startTag reads one start tag.
func (p *htmlParser) startTag() (*htmlNode, bool, int, error) {
	j := p.i + 1
	start := j
	for j < len(p.src) && isTagNameByte(p.src[j], j == start) {
		j++
	}
	raw := p.src[start:j]
	if raw == "" {
		return nil, false, 0, p.fault("a bare '<' in text; write it as &lt;")
	}
	node := &htmlNode{Kind: htmlElementNode, Name: normalizeTagName(raw), Attrs: map[string]string{}, Line: p.line}

	for {
		for j < len(p.src) && isSpace(p.src[j]) {
			j++
		}
		if j >= len(p.src) {
			return nil, false, 0, p.fault("<" + node.Name + " is not closed by '>'")
		}
		if p.src[j] == '>' {
			return node, false, j + 1, nil
		}
		if p.src[j] == '/' {
			if j+1 >= len(p.src) || p.src[j+1] != '>' {
				return nil, false, 0, p.fault("<" + node.Name + " has a stray '/'")
			}
			return node, true, j + 2, nil
		}
		nameStart := j
		for j < len(p.src) && isAttrNameByte(p.src[j]) {
			j++
		}
		name := p.src[nameStart:j]
		if name == "" {
			return nil, false, 0, p.fault("<" + node.Name + "> has an attribute name the check cannot read at " +
				quote(clip(p.src[j:])))
		}
		if _, dup := node.Attrs[name]; dup {
			return nil, false, 0, p.fault("<" + node.Name + "> repeats the attribute " + quote(name))
		}
		if j < len(p.src) && p.src[j] == '=' {
			j++
			if j >= len(p.src) || p.src[j] != '"' {
				return nil, false, 0, p.fault("<" + node.Name + "> gives " + quote(name) +
					" a value that is not double-quoted; the generator quotes every one")
			}
			j++
			close := strings.IndexByte(p.src[j:], '"')
			if close < 0 {
				return nil, false, 0, p.fault("<" + node.Name + "> leaves " + quote(name) + " unterminated")
			}
			value, err := p.decodeEntities(p.src[j:j+close], false)
			if err != nil {
				return nil, false, 0, err
			}
			node.Attrs[name] = value
			j += close + 1
			continue
		}
		node.Attrs[name] = ""
	}
}

// decodeEntities reverses the escaping the generator applies.
//
// In TEXT it is strict: an ampersand that is not one of the entities
// `escapeText` produces is either a character the generator failed to escape or
// an entity this reader does not understand, and either way the provenance walk
// would be reading words nobody can vouch for.
//
// In an ATTRIBUTE VALUE it is not. An attribute holds no visible text — the
// walk never reads one for provenance — and a URL legitimately carries a bare
// `&` between its query parameters, which is what the font stylesheet's href
// is made of. An unresolvable reference there is left as the literal text it
// is, rather than failing a page over punctuation nobody reads.
func (p *htmlParser) decodeEntities(s string, strict bool) (string, error) {
	if !strings.ContainsRune(s, '&') {
		return s, nil
	}
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] != '&' {
			b.WriteByte(s[i])
			i++
			continue
		}
		end := strings.IndexByte(s[i:], ';')
		ref := ""
		if end >= 0 && end <= 12 {
			ref = s[i+1 : i+end]
		}
		out, ok := decodeEntityRef(ref)
		if !ok {
			if strict {
				if ref == "" {
					return "", p.fault("an unterminated entity at " + quote(clip(s[i:])))
				}
				return "", p.fault("the entity " + quote("&"+ref+";") + " is not one the generator emits")
			}
			b.WriteByte('&')
			i++
			continue
		}
		b.WriteString(out)
		i += end + 1
	}
	return b.String(), nil
}

// decodeEntityRef resolves the named and numeric references the generator and
// the committed drawings use.
func decodeEntityRef(ref string) (string, bool) {
	switch ref {
	case "amp":
		return "&", true
	case "lt":
		return "<", true
	case "gt":
		return ">", true
	case "quot":
		return `"`, true
	case "apos":
		return "'", true
	}
	if !strings.HasPrefix(ref, "#") || len(ref) < 2 {
		return "", false
	}
	digits, base := ref[1:], 10
	if digits[0] == 'x' || digits[0] == 'X' {
		digits, base = digits[1:], 16
	}
	if digits == "" {
		return "", false
	}
	n := 0
	for i := 0; i < len(digits); i++ {
		d := hexDigit(digits[i])
		if d < 0 || d >= base {
			return "", false
		}
		n = n*base + d
		if n > 0x10FFFF {
			return "", false
		}
	}
	return string(rune(n)), true
}

func hexDigit(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return -1
}

// normalizeTagName lower-cases HTML tag names while keeping the mixed-case SVG
// ones the drawings use, so `<clipPath>` stays `clipPath` and `<DIV>` — which
// the generator never writes — is still recognised as a div.
func normalizeTagName(raw string) string {
	if svgElements[raw] {
		return raw
	}
	return strings.ToLower(raw)
}

// indexFoldASCII finds needle in hay, ASCII-case-insensitively, returning an
// index into HAY.
//
// The obvious spelling — `strings.Index(strings.ToLower(hay), needle)` — returns
// an index into the LOWER-CASED COPY, and case folding is not length-preserving:
// `İ` and `K` shrink, and a byte that is not valid UTF-8 grows into a
// three-byte replacement character. The index then lands in the wrong place in
// the original, and the parser resumes part-way through an element — which does
// not merely garble a text node, it can skip the markup that follows the raw-text
// element entirely, so a page with unsourced words in it parses "successfully"
// and the provenance walk never sees them. A gate that passes by not looking is
// the one outcome this file exists to prevent, so the search never leaves the
// original bytes.
//
// Folding is ASCII-only on purpose. `strings.EqualFold` would match `K` (U+212A)
// to `k`, which is a tag name no browser closes.
func indexFoldASCII(hay, needle string) int {
	if needle == "" {
		return 0
	}
	for i := 0; i+len(needle) <= len(hay); i++ {
		if equalFoldASCII(hay[i:i+len(needle)], needle) {
			return i
		}
	}
	return -1
}

// equalFoldASCII compares two equal-length strings, ASCII case folded.
func equalFoldASCII(a, b string) bool {
	for i := 0; i < len(a); i++ {
		if lowerASCII(a[i]) != lowerASCII(b[i]) {
			return false
		}
	}
	return true
}

// isASCIILetter reports whether a byte opens or continues a tag or attribute
// name. The markdown renderer had a predicate of this shape and no longer does;
// the reader keeps its own rather than reaching for one that is not there.
func isASCIILetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func lowerASCII(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + ('a' - 'A')
	}
	return c
}

func isTagNameByte(c byte, first bool) bool {
	if isASCIILetter(c) {
		return true
	}
	if first {
		return false
	}
	return (c >= '0' && c <= '9') || c == '-' || c == ':' || c == '_' || c == '.'
}

func isAttrNameByte(c byte) bool {
	return isASCIILetter(c) || (c >= '0' && c <= '9') || c == '-' || c == ':' || c == '_' || c == '.'
}

func itoaLine(n int) string { return fmt.Sprintf("%d", n) }
