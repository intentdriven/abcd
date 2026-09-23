package memory

import "testing"

// yaml_bom_test.go — iss-2608291814565781: a UTF-8 byte-order mark ahead of the
// opening `---` is not Unicode White_Space, so strings.TrimSpace keeps it and the
// memory parsers read a BOM-led page as one without frontmatter, while record-lint
// (frontmatter.TrimBOM) reads the same bytes as a page with it. Every memory
// frontmatter reader goes through frontmatterOpenIndex, so each is driven here.
func TestBOMLedPageIsParsedWithItsFrontmatter(t *testing.T) {
	const bom = "\ufeff"
	docs := map[string]string{
		"delimiter_first":  bom + "---\ntitle: Alpha\n---\nbody\n",
		"comment_preamble": bom + "<!-- attribution -->\n---\ntitle: Alpha\n---\nbody\n",
	}
	for name, doc := range docs {
		t.Run(name, func(t *testing.T) {
			if !textOpensFrontmatter(doc) {
				t.Errorf("textOpensFrontmatter = false, want true")
			}
			fm, err := parseFrontmatter(doc)
			if err != nil {
				t.Fatalf("parseFrontmatter: %v", err)
			}
			if fm["title"] != "Alpha" {
				t.Errorf("parseFrontmatter title = %v, want Alpha", fm["title"])
			}
			region, body, err := splitFileFrontmatter(doc)
			if err != nil {
				t.Fatalf("splitFileFrontmatter: %v", err)
			}
			if region != "title: Alpha\n" || body != "body\n" {
				t.Errorf("splitFileFrontmatter = (%q, %q)", region, body)
			}
			if got, want := frontmatterKeyLine(doc, "title"), map[string]int{"delimiter_first": 2, "comment_preamble": 3}[name]; got != want {
				t.Errorf("frontmatterKeyLine = %d, want %d", got, want)
			}
		})
	}
}

// A BOM is stripped from the document's first line only: U+FEFF anywhere else is
// content (a zero-width no-break space), not a byte-order mark, so a later line
// carrying it does not become a delimiter.
func TestBOMIsStrippedFromTheFirstLineOnly(t *testing.T) {
	doc := "<!-- c -->\n\ufeff---\ntitle: Alpha\n---\n"
	if textOpensFrontmatter(doc) {
		t.Errorf("a U+FEFF on a non-first line was treated as a byte-order mark")
	}
}
