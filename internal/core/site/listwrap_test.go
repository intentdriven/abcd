package site

// A `.list` row never widens the page, whatever title it carries.
//
// A row is a two-column grid, the id and then the title, and a grid item's
// automatic minimum width is its min-content width: a title holding one long
// unspaced token — a home-relative path, a URL — sets the width of the row, and
// the row spills past its panel and makes the whole page scroll sideways on a
// phone (iss-2610040729344770, the /record/ dashboard at 360 px). The browser
// half of that rule is the overflow audit in site-src/audit/; this is the half a
// test can hold without one: the title reaches the page as a non-id span
// directly under the row, and the stylesheet gives every such span, at every
// width, the two properties that let it give.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// listTitleSelector is the selector the stylesheet must carry, and the
// declarations it must give. `overflow-wrap:anywhere` rather than `break-word`
// because only `anywhere` lowers the element's min-content width; `min-width:0`
// because a grid item's automatic minimum is that min-content width.
const listTitleSelector = ".list li>span:not(.id)"

var listTitleDecls = []string{"overflow-wrap:anywhere", "min-width:0"}

// TestAListTitleWithAnUnbreakableTokenIsAWrappingSpan renders the dashboard's
// latest decisions with a title that is one 157-character token and holds that
// it lands where the stylesheet's wrapping rule reaches it.
func TestAListTitleWithAnUnbreakableTokenIsAWrappingSpan(t *testing.T) {
	token := "~/.abcd.noindex/transcripts/" + strings.Repeat("0123456789abcdef", 8) + "/"
	e := &explorer{c: &composer{}, export: RecordExport{Nodes: []ExportNode{
		{ID: "adr-7", Type: "adr", Date: "2026-10-03", Title: token},
	}}}
	got := e.latestDecisions()
	want := `<li><span class="id"><a href="/record/adr/adr-7/">adr-7</a><span class="d">2026-10-03</span></span>` +
		`<span>` + escapeText(token) + `</span></li>`
	if !strings.Contains(got, `<ul class="list">`) || !strings.Contains(got, want) {
		t.Fatalf("the title is not a bare span directly under a .list row:\nwant %s\nin   %s", want, got)
	}
}

// TestTheStylesheetLetsAListTitleGive holds the rule in abcd's own stylesheet
// and in the copy setup seeds a managed repository with, at the top level so it
// applies at every width rather than inside one media query.
func TestTheStylesheetLetsAListTitleGive(t *testing.T) {
	own, err := os.ReadFile(filepath.Join("..", "..", "..", "site-src", "site.css"))
	if err != nil {
		t.Fatal(err)
	}
	seed, err := setupSources.ReadFile("setupsrc/site.css")
	if err != nil {
		t.Fatal(err)
	}
	for name, src := range map[string]string{"site-src/site.css": string(own), "setupsrc/site.css": string(seed)} {
		body, ok := topLevelRules(src)[listTitleSelector]
		if !ok {
			t.Errorf("%s has no top-level rule for %s: a long title in a .list row widens the page", name, listTitleSelector)
			continue
		}
		decls := strings.ReplaceAll(body, " ", "")
		for _, d := range listTitleDecls {
			if !strings.Contains(decls, d) {
				t.Errorf("%s gives %s no %s (body %q)", name, listTitleSelector, d, body)
			}
		}
	}
}

// topLevelRules maps each selector of every top-level rule to its declaration
// block, concatenated where one selector has several rules. At-rules are
// skipped whole: a rule inside a media query holds at one width, not at all of
// them.
func topLevelRules(src string) map[string]string {
	src = cssCommentRe.ReplaceAllString(src, " ")
	out := map[string]string{}
	for i := 0; i < len(src); {
		open := strings.IndexByte(src[i:], '{')
		if open < 0 {
			break
		}
		open += i
		selectors := strings.TrimSpace(src[i:open])
		end := matchBrace(src, open)
		if end < 0 {
			break
		}
		if !strings.HasPrefix(selectors, "@") {
			for _, sel := range strings.Split(selectors, ",") {
				sel = strings.TrimSpace(sel)
				out[sel] += src[open+1:end] + ";"
			}
		}
		i = end + 1
	}
	return out
}
