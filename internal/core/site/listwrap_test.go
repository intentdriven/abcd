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

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// listTitleSelector is the selector the stylesheet must carry, and the
// declarations it must give. `overflow-wrap:anywhere` rather than `break-word`
// because only `anywhere` lowers the element's min-content width; `min-width:0`
// because a grid item's automatic minimum is that min-content width.
const listTitleSelector = ".list li>span:not(.id)"

var listTitleDecls = map[string]string{"overflow-wrap": "anywhere", "min-width": "0"}

// TestAListTitleWithAnUnbreakableTokenIsAWrappingSpan renders the dashboard's
// latest decisions with a title that is one 157-character token and holds that
// it lands where the stylesheet's wrapping rule reaches it.
func TestAListTitleWithAnUnbreakableTokenIsAWrappingSpan(t *testing.T) {
	token := abcdhome.Display("transcripts/") + strings.Repeat("0123456789abcdef", 8) + "/"
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
	for name, src := range bothStylesheets(t) {
		rule, ok := cascadeOf(src).top[listTitleSelector]
		if !ok {
			t.Errorf("%s has no top-level rule for %s: a long title in a .list row widens the page", name, listTitleSelector)
			continue
		}
		for prop, want := range listTitleDecls {
			if got := rule[prop].value; got != want {
				t.Errorf("%s gives %s %s:%q, want %q", name, listTitleSelector, prop, got, want)
			}
		}
	}
}

// bothStylesheets reads abcd's own stylesheet and the copy setup seeds a
// managed repository with, keyed by a name for the failure message.
func bothStylesheets(t *testing.T) map[string]string {
	t.Helper()
	own, err := os.ReadFile(filepath.Join("..", "..", "..", "site-src", "site.css"))
	if err != nil {
		t.Fatal(err)
	}
	seed, err := setupSources.ReadFile("setupsrc/site.css")
	if err != nil {
		t.Fatal(err)
	}
	return map[string]string{"site-src/site.css": string(own), "setupsrc/site.css": string(seed)}
}

// cssDecl is one property's value as the cascade leaves it within one rule
// set, and the source offset of the rule that gave it, so a test can hold that
// a narrow-screen override comes after the rule it overrides.
type cssDecl struct {
	value string
	at    int
}

// cssRules maps a normalised selector to its properties. Where one selector
// has several rules, the later declaration of a property wins, as it does in
// the cascade: a weaker rule further down the file is the value a browser
// uses, so it is the value a test sees.
type cssRules map[string]map[string]cssDecl

// cascade is a parsed stylesheet: the rules at the top level, which apply
// at every width, and the rules inside each `@media` block, keyed by the
// block's normalised prelude (`@media (max-width:520px)`). Other at-rules are
// skipped whole.
type cascade struct {
	top   cssRules
	media map[string]cssRules
}

// cascadeOf reads src into its top-level and media-query rules.
func cascadeOf(src string) cascade {
	src = cssCommentRe.ReplaceAllString(src, " ")
	sh := cascade{top: cssRules{}, media: map[string]cssRules{}}
	cascadeRules(src, 0, len(src), sh.top, func(prelude string, from, to int) {
		if !strings.HasPrefix(prelude, "@media") {
			return
		}
		key := normaliseSelector(prelude)
		if sh.media[key] == nil {
			sh.media[key] = cssRules{}
		}
		cascadeRules(src, from, to, sh.media[key], nil)
	})
	return sh
}

// cascadeRules walks the rules in src[from:to] into into. An at-rule's block is
// handed to atRule, or skipped when atRule is nil.
func cascadeRules(src string, from, to int, into cssRules, atRule func(prelude string, from, to int)) {
	for i := from; i < to; {
		open := strings.IndexByte(src[i:to], '{')
		if open < 0 {
			return
		}
		open += i
		end := matchBrace(src, open)
		if end < 0 || end > to {
			return
		}
		prelude := strings.TrimSpace(src[i:open])
		if strings.HasPrefix(prelude, "@") {
			if atRule != nil {
				atRule(prelude, open+1, end)
			}
			i = end + 1
			continue
		}
		decls := cascadeDecls(src[open+1:end], open)
		for _, sel := range strings.Split(prelude, ",") {
			sel = normaliseSelector(sel)
			if into[sel] == nil {
				into[sel] = map[string]cssDecl{}
			}
			for prop, d := range decls {
				into[sel][prop] = d
			}
		}
		i = end + 1
	}
}

// cascadeDecls reads one declaration block; within it, too, the last
// declaration of a property wins.
func cascadeDecls(body string, at int) map[string]cssDecl {
	out := map[string]cssDecl{}
	for _, d := range strings.Split(body, ";") {
		prop, value, ok := strings.Cut(d, ":")
		if !ok {
			continue
		}
		out[strings.ToLower(strings.TrimSpace(prop))] = cssDecl{value: normaliseValue(value), at: at}
	}
	return out
}

// normaliseSelector collapses whitespace and drops it around a combinator, so
// `.list li > span:not( .id )` and `.list li>span:not(.id)` are one key. A
// space before `:` or `(` stays: there it is a descendant combinator. The same
// pass keys a media prelude (`@media (max-width: 520px)`).
func normaliseSelector(sel string) string {
	sel = strings.Join(strings.Fields(sel), " ")
	for _, c := range []string{">", "+", "~"} {
		sel = strings.ReplaceAll(sel, " "+c, c)
		sel = strings.ReplaceAll(sel, c+" ", c)
	}
	for _, r := range []struct{ from, to string }{{"( ", "("}, {" )", ")"}, {": ", ":"}} {
		sel = strings.ReplaceAll(sel, r.from, r.to)
	}
	return sel
}

// normaliseValue collapses whitespace and drops it inside parentheses and
// after a comma, so `minmax(0, 1fr)` reads as `minmax(0,1fr)`.
func normaliseValue(v string) string {
	v = strings.Join(strings.Fields(v), " ")
	for _, r := range []struct{ from, to string }{{"( ", "("}, {" )", ")"}, {" ,", ","}, {", ", ","}} {
		v = strings.ReplaceAll(v, r.from, r.to)
	}
	return v
}

// TestTheCascadeHelperSeesWhatABrowserUses holds the helper the stylesheet
// tests read through: a selector written with spaces around its combinator is
// the same selector, a weaker rule further down the file is the value that
// holds, and a rule inside a media query is not a top-level rule.
func TestTheCascadeHelperSeesWhatABrowserUses(t *testing.T) {
	src := `.list li>span:not(.id){overflow-wrap:anywhere;min-width:0}
/* a later, weaker rule written with spaces */
.list  li > span:not( .id ) { overflow-wrap : break-word }
.list :hover{color:red}
@media (max-width: 520px){.list li{grid-template-columns:minmax(0, 1fr)}}`
	c := cascadeOf(src)
	rule := c.top[listTitleSelector]
	if got := rule["overflow-wrap"].value; got != "break-word" {
		t.Errorf("overflow-wrap = %q, want the later rule's break-word", got)
	}
	if got := rule["min-width"].value; got != "0" {
		t.Errorf("min-width = %q, want the earlier rule's 0 kept", got)
	}
	if _, ok := c.top[".list :hover"]; !ok {
		t.Error("a descendant combinator before a pseudo-class was dropped")
	}
	if _, ok := c.top[".list li"]; ok {
		t.Error("a rule inside a media query is read as a top-level rule")
	}
	if got := c.media["@media (max-width:520px)"][".list li"]["grid-template-columns"].value; got != "minmax(0,1fr)" {
		t.Errorf("media rule grid-template-columns = %q (media keys %v)", got, mediaKeys(c))
	}
}

func mediaKeys(c cascade) []string {
	var keys []string
	for k := range c.media {
		keys = append(keys, k)
	}
	return keys
}
