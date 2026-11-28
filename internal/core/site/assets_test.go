package site

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// assetFixture writes one file under a temporary repo root and returns the root.
func assetFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// TestAssetsRefuseAnythingOutsideTheAssetRoot closes the widest hole a
// generator like this has: an image reference is a path, and a path read from
// repository text will happily name a file that is not a picture.
//
// The build publishes what it reads. `![x](../../.git/config)` would therefore
// copy git's config — which on a CI runner carries the checkout token in an
// `http.…extraheader` line — into a directory served to the public, under a
// name nobody would look at twice. The same reference reaches `.env`, and the
// gitignored `.abcd/.work.local/` tier, none of which git would ever show as
// changed, because the output directory is not tracked.
func TestAssetsRefuseAnythingOutsideTheAssetRoot(t *testing.T) {
	root := assetFixture(t, map[string]string{
		".git/config":                     "[core]\n\textraheader = AUTHORIZATION: basic SECRET\n",
		".env":                            "TOKEN=not-a-real-token\n",
		".abcd/.work.local/scratch/n.txt": "local notes\n",
		"docs/assets/img/ok.svg":          `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1 1"></svg>`,
		"docs/explanation/notes.md":       "prose\n",
	})
	pipe := newAssetPipe(root, mustOpenRoot(t, root), "/")
	at := Source{Path: "docs/explanation/page.md", Line: 3}

	for _, src := range []string{
		"../../.git/config",
		"../../.env",
		"../../.abcd/.work.local/scratch/n.txt",
		"../explanation/notes.md",
	} {
		if _, err := pipe.render("docs/explanation", src, "", at); err == nil {
			t.Errorf("%s was published; only committed pictures under the asset root may be", src)
		} else if !strings.Contains(err.Error(), at.Path) {
			t.Errorf("%s: the refusal does not name the page: %v", src, err)
		}
	}

	// The legitimate case still works.
	if _, err := pipe.render("docs/explanation", "../assets/img/ok.svg", "", at); err != nil {
		t.Errorf("a committed asset was refused: %v", err)
	}
}

// TestAssetsRefuseExecutableSVG is the other half of the same boundary. An SVG
// is INLINED — that is the whole point, so its var(--token) colours follow the
// reader's theme — which means its bytes become markup in the published page.
// A `<script>` element or an `on*` attribute inside one is therefore executable
// code on the site's own origin, reached with no click, from a file that is not
// code and that no reviewer reads as code.
func TestAssetsRefuseExecutableSVG(t *testing.T) {
	cases := []struct{ name, svg, says string }{
		{"script element",
			`<svg xmlns="http://www.w3.org/2000/svg"><script>fetch('https://evil.invalid/')</script></svg>`,
			"script"},
		{"event handler",
			`<svg xmlns="http://www.w3.org/2000/svg"><rect onload="alert(1)" width="1" height="1"/></svg>`,
			"onload"},
		{"mixed-case event handler",
			`<svg xmlns="http://www.w3.org/2000/svg"><rect OnLoad="alert(1)" width="1" height="1"/></svg>`,
			"onload"},
		{"foreign object",
			`<svg xmlns="http://www.w3.org/2000/svg"><foreignObject><body xmlns="http://www.w3.org/1999/xhtml"/></foreignObject></svg>`,
			"foreignObject"},
		{"executable href",
			`<svg xmlns="http://www.w3.org/2000/svg"><a href="javascript:alert(1)"><rect width="1" height="1"/></a></svg>`,
			""},
		{"external use",
			`<svg xmlns="http://www.w3.org/2000/svg"><use href="https://evil.invalid/x.svg#a"/></svg>`,
			"href"},
		{"style element",
			`<svg xmlns="http://www.w3.org/2000/svg"><style>@import url(https://evil.invalid/x.css);</style></svg>`,
			"style"},
		{"external image panel",
			`<svg xmlns="http://www.w3.org/2000/svg"><image href="https://evil.invalid/x.png"/></svg>`,
			"image"},
		// The <image> carve-out exists for the committed drawings' raster
		// panels; nesting a document inside one would reopen the surface.
		{"svg inside an image panel",
			`<svg xmlns="http://www.w3.org/2000/svg"><image href="data:image/svg+xml;base64,PHN2Zz48c2NyaXB0Lz48L3N2Zz4="/></svg>`,
			"image"},
		{"doctype entity",
			`<!DOCTYPE svg [<!ENTITY x SYSTEM "file:///etc/passwd">]><svg xmlns="http://www.w3.org/2000/svg"><desc>&x;</desc></svg>`,
			""},
		// A prolog, a comment or a CDATA section is inlined verbatim and the
		// emitted page's reader (htmlscan) refuses all three — so the build must
		// refuse them at the asset, not leave the page to fail the check with a
		// message that names no file. Every mainstream SVG exporter writes a prolog
		// and a generator comment.
		{"xml prolog",
			`<?xml version="1.0" encoding="UTF-8"?><svg xmlns="http://www.w3.org/2000/svg"><rect width="1" height="1"/></svg>`,
			"processing instruction"},
		{"generator comment",
			`<svg xmlns="http://www.w3.org/2000/svg"><!-- Generator: Adobe Illustrator --><rect width="1" height="1"/></svg>`,
			"comment"},
		{"cdata section",
			`<svg xmlns="http://www.w3.org/2000/svg"><title><![CDATA[x]]></title><rect width="1" height="1"/></svg>`,
			"cdata"},
	}
	at := Source{Path: "docs/explanation/page.md", Line: 5}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := assetFixture(t, map[string]string{"docs/assets/img/x.svg": c.svg})
			pipe := newAssetPipe(root, mustOpenRoot(t, root), "/")
			out, err := pipe.render("docs/explanation", "../assets/img/x.svg", "", at)
			if err == nil {
				t.Fatalf("inlined an executable SVG into the page:\n%s", out)
			}
			if !strings.Contains(err.Error(), "docs/assets/img/x.svg") {
				t.Errorf("the refusal does not name the asset: %v", err)
			}
			if c.says != "" && !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(c.says)) {
				t.Errorf("the refusal does not say %q: %v", c.says, err)
			}
		})
	}
}

// TestAssetsRefuseStyledSVG is the same boundary read one step further out. A
// drawing that carries no script can still carry a stylesheet, because `style`
// and `class` are attributes, and an inlined drawing's attributes are live
// markup on the page like any others.
//
// Inline CSS on the root wins against `.svgasset svg` and `.card .icon svg`,
// neither of which is `!important`, so a drawing can size itself to the
// viewport; a child can `position:fixed` out of the box the page laid out for
// it, and cover the page it was drawn into. `class` is the same reach by
// another route — it borrows whatever the site's own stylesheet says about that
// name. Neither is a thing a drawing needs: the committed corpus carries paint
// and geometry, and its colours are `var(--token, fallback)` values on the
// paint attributes themselves.
func TestAssetsRefuseStyledSVG(t *testing.T) {
	cases := []struct{ name, svg, says string }{
		{"style on the root",
			`<svg xmlns="http://www.w3.org/2000/svg" style="position:fixed;inset:0;width:100vw;height:100vh"><rect width="1" height="1"/></svg>`,
			"style"},
		{"style on a child",
			`<svg xmlns="http://www.w3.org/2000/svg"><rect style="position:fixed;inset:0" width="1" height="1"/></svg>`,
			"style"},
		{"class on the root",
			`<svg xmlns="http://www.w3.org/2000/svg" class="wrap"><rect width="1" height="1"/></svg>`,
			"class"},
		{"class on a child",
			`<svg xmlns="http://www.w3.org/2000/svg"><g class="hero"><rect width="1" height="1"/></g></svg>`,
			"class"},
		// The allowlist is closed, so an attribute nobody weighed is refused by
		// name rather than passed through for the browser to decide about.
		{"an attribute a drawing is not made of",
			`<svg xmlns="http://www.w3.org/2000/svg"><rect requiredExtensions="x" width="1" height="1"/></svg>`,
			"requiredExtensions"},
		// A namespace declaration is how a foreign vocabulary gets in; a drawing
		// declares SVG, and xlink for the `xlink:href` its <use> elements write.
		{"a foreign namespace declaration",
			`<svg xmlns="http://www.w3.org/2000/svg" xmlns:h="http://www.w3.org/1999/xhtml"><rect width="1" height="1"/></svg>`,
			"namespace"},
	}
	at := Source{Path: "docs/explanation/page.md", Line: 7}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := assetFixture(t, map[string]string{"docs/assets/img/x.svg": c.svg})
			pipe := newAssetPipe(root, mustOpenRoot(t, root), "/")
			out, err := pipe.render("docs/explanation", "../assets/img/x.svg", "", at)
			if err == nil {
				t.Fatalf("inlined a styled SVG into the page:\n%s", out)
			}
			if !strings.Contains(err.Error(), "docs/assets/img/x.svg") {
				t.Errorf("the refusal does not name the asset: %v", err)
			}
			if c.says != "" && !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(c.says)) {
				t.Errorf("the refusal does not say %q: %v", c.says, err)
			}
		})
	}
}

// TestAssetsAcceptDrawings keeps the refusal from swallowing the real corpus:
// the shapes, gradients, clip paths and text the committed drawings are made of
// stay renderable, and a same-document reference is not an external one.
func TestAssetsAcceptDrawings(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 10 10" width="10" height="10">
<title>A drawing</title><desc>Described</desc>
<defs><clipPath id="c"><rect width="10" height="10"/></clipPath>
<marker id="m" markerWidth="4" markerHeight="4" refX="2" refY="2" orient="auto"><path d="M0 0 L4 2 L0 4 z"/></marker>
<linearGradient id="g"><stop offset="0" stop-color="var(--accent, #06c)"/></linearGradient></defs>
<g clip-path="url(#c)"><circle cx="5" cy="5" r="4" fill="var(--ink, #000)"/>
<path d="M0 0 L10 10" stroke="url(#g)" marker-end="url(#m)"/>
<text x="1" y="9" font-family="sans-serif" font-size="2">hi<tspan dy="1">there</tspan></text>
<use xlink:href="#c"/></g></svg>`
	root := assetFixture(t, map[string]string{"docs/assets/img/x.svg": svg})
	pipe := newAssetPipe(root, mustOpenRoot(t, root), "/")
	out, err := pipe.render("docs/explanation", "../assets/img/x.svg", "", Source{Path: "p.md", Line: 1})
	if err != nil {
		t.Fatalf("a drawing was refused: %v", err)
	}
	if !strings.Contains(out, "var(--ink, #000)") {
		t.Error("the token colours did not survive inlining")
	}
	// The root is sized by the stylesheet; the clip path's rect is not, and
	// TestAssetsStripOnlyTheRootDrawingSize is where that split is pinned.
	if strings.Contains(out, `viewBox="0 0 10 10" width="10"`) {
		t.Error("the stylesheet must size the drawing; the root's own width survived")
	}
}

// TestAssetsStripOnlyTheRootDrawingSize pins the one attribute pair the build
// is allowed to take away, and the elements it must not take it away from.
//
// The ROOT <svg> is sized by the stylesheet, so its width and height go. Every
// other element's width and height ARE the drawing: a <rect> is a box of that
// size, an <image> panel is a raster framed to that size and clipped to it, and
// a <use>, <pattern>, <mask> or <symbol> establishes a region of that size.
// Dropping those does not resize a drawing, it destroys it — an <image> falls
// back to its raster's intrinsic pixel size and bursts out of the clip path it
// was framed by, and a <rect> collapses to nothing at all.
func TestAssetsStripOnlyTheRootDrawingSize(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100" width="100" height="100">
<defs><clipPath id="cp"><circle cx="21" cy="21" r="21"/></clipPath>
<pattern id="pt" width="8" height="8" patternUnits="userSpaceOnUse"><line x1="0" y1="0" x2="8" y2="8"/></pattern>
<mask id="mk" width="30" height="30"><rect width="30" height="30" fill="#fff"/></mask>
<symbol id="sy" width="12" height="12"><rect width="12" height="12"/></symbol></defs>
<rect x="4" y="4" width="92" height="34" rx="7" fill="var(--surface, #fff)"/>
<image href="data:image/webp;base64,AAAA" x="10" y="10" width="42" height="42" clip-path="url(#cp)" preserveAspectRatio="xMidYMid slice"/>
<use href="#sy" x="60" y="60" width="12" height="12"/></svg>`
	root := assetFixture(t, map[string]string{"docs/assets/img/x.svg": svg})
	pipe := newAssetPipe(root, mustOpenRoot(t, root), "/")
	out, err := pipe.render("docs/explanation", "../assets/img/x.svg", "", Source{Path: "p.md", Line: 1})
	if err != nil {
		t.Fatalf("a drawing was refused: %v", err)
	}

	// The root, and only the root, loses its size.
	rootTag := out[strings.Index(out, "<svg"):]
	rootTag = rootTag[:strings.Index(rootTag, ">")+1]
	if strings.Contains(rootTag, "width=") || strings.Contains(rootTag, "height=") {
		t.Errorf("the stylesheet must size the drawing; the root kept its own size: %s", rootTag)
	}
	if !strings.Contains(rootTag, `viewBox="0 0 100 100"`) {
		t.Errorf("the root lost its viewBox, which is the coordinate system: %s", rootTag)
	}

	// Every other element keeps the size that IS the drawing.
	for _, want := range []string{
		`<pattern id="pt" width="8" height="8"`,
		`<mask id="mk" width="30" height="30"`,
		`<symbol id="sy" width="12" height="12"`,
		`<rect width="30" height="30" fill="#fff"/>`,
		`<rect width="12" height="12"/>`,
		`<rect x="4" y="4" width="92" height="34"`,
		`x="10" y="10" width="42" height="42" clip-path="url(#cp)"`,
		`<use href="#sy" x="60" y="60" width="12" height="12"/>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("the inlined drawing lost a size that is part of it: want %s", want)
		}
	}
}

// TestAssetsStripTouchesTheRootTagAlone pins the boundary of the replaced
// region itself. The size strip is a text substitution, so the bytes it is
// allowed to see are the whole of its correctness: given the document rather
// than the root start tag, it edits whatever else happens to read like a size —
// which is the shape of the bug it was written to end, one level up.
func TestAssetsStripTouchesTheRootTagAlone(t *testing.T) {
	svg := `keep width="1" me<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 4 4" width="4" height="4"><rect width="2" height="2"/></svg>`
	root := assetFixture(t, map[string]string{"docs/assets/img/x.svg": svg})
	pipe := newAssetPipe(root, mustOpenRoot(t, root), "/")
	out, err := pipe.render("docs/explanation", "../assets/img/x.svg", "", Source{Path: "p.md", Line: 1})
	if err != nil {
		t.Fatalf("a drawing was refused: %v", err)
	}
	if !strings.Contains(out, `keep width="1" me`) {
		t.Errorf("the strip reached past the root start tag, into what precedes it:\n%s", out)
	}
	if !strings.Contains(out, `<rect width="2" height="2"/>`) {
		t.Errorf("the strip reached past the root start tag, into the drawing:\n%s", out)
	}
	if strings.Contains(out, `viewBox="0 0 4 4" width="4"`) {
		t.Errorf("the root kept its own size:\n%s", out)
	}
}

// TestCommittedSVGAssetsAreDrawings runs the same refusal over every SVG this
// repository actually carries, so an asset that could never be published is
// caught when it lands rather than when a page first references it.
//
// It is also the anti-vacuity guard on the attribute allowlist. A closed list
// is only as good as the evidence it was drawn from, and the cheapest way to
// write one that refuses nothing real is to write one that refuses everything
// real: every committed drawing goes through the whole pipeline here, not just
// the parse, so an attribute the corpus carries and the list forgot fails this
// test rather than the site build.
func TestCommittedSVGAssetsAreDrawings(t *testing.T) {
	repoRoot := filepath.Join("..", "..", "..")
	dir := filepath.Join(repoRoot, "docs", "assets", "img")
	pipe := newAssetPipe(repoRoot, mustOpenRoot(t, repoRoot), "/")
	at := Source{Path: "docs/explanation/page.md", Line: 1}
	var found int
	err := filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(p, ".svg") {
			return err
		}
		found++
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if err := checkInlinableSVG(string(data), p); err != nil {
			t.Errorf("%v", err)
			return nil
		}
		// The reference as a page would write it: a docs page naming an asset
		// under the asset root, resolved and inlined by the real pipe.
		rel, err := filepath.Rel(filepath.Join(repoRoot, "docs"), p)
		if err != nil {
			return err
		}
		out, err := pipe.render("docs", filepath.ToSlash(rel), "", at)
		if err != nil {
			t.Errorf("a committed drawing no longer inlines: %v", err)
			return nil
		}
		if !strings.Contains(out, "<svg") {
			t.Errorf("%s: the inlined drawing carries no <svg> element", p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if found == 0 {
		t.Fatal("no SVG assets found; the scan proved nothing")
	}
}

// inlinedName reads the accessible name of an inlined picture the way an
// assistive reader meets it: the wrapper is the picture, so its own role and
// label are what it says, and a hidden wrapper says nothing. It also reports
// whether the drawing inside is hidden, since a drawing left exposed would add
// a name of its own beside the page's.
func inlinedName(t *testing.T, out string) (name string, exposed, drawingHidden bool) {
	t.Helper()
	dec := xml.NewDecoder(strings.NewReader(out))
	var starts []xml.StartElement
	for len(starts) < 2 {
		tok, err := dec.Token()
		if err != nil {
			t.Fatalf("the inlined picture does not parse: %v\n%s", err, out)
		}
		if se, ok := tok.(xml.StartElement); ok {
			starts = append(starts, se.Copy())
		}
	}
	attr := func(se xml.StartElement, key string) (string, bool) {
		for _, a := range se.Attr {
			if a.Name.Space == "" && a.Name.Local == key {
				return a.Value, true
			}
		}
		return "", false
	}
	wrap, drawing := starts[0], starts[1]
	if wrap.Name.Local != "span" || drawing.Name.Local != "svg" {
		t.Fatalf("want a span wrapping the drawing, got <%s> then <%s>:\n%s", wrap.Name.Local, drawing.Name.Local, out)
	}
	hidden, _ := attr(drawing, "aria-hidden")
	if h, _ := attr(wrap, "aria-hidden"); h == "true" {
		return "", false, hidden == "true"
	}
	role, _ := attr(wrap, "role")
	label, _ := attr(wrap, "aria-label")
	if role != "img" {
		label = ""
	}
	return label, true, hidden == "true"
}

// TestAssetsNameAnInlinedDrawingFromThePage pins iss-2610241550257430: the alt
// text a page gives an SVG picture is the picture's accessible name on the
// site, as it is on the forge, and not whatever the drawing file declares. The
// drawing here declares a name of its own, as the committed ones do, so a
// wrapper that ignored alt would read "brief" and not the page's words. An
// empty alt is a decorative picture, which says nothing at all.
func TestAssetsNameAnInlinedDrawingFromThePage(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 4 4" width="4" height="4" role="img" aria-label="brief"><title>brief</title><rect width="2" height="2"/></svg>`
	root := assetFixture(t, map[string]string{"docs/assets/img/x.svg": svg})
	pipe := newAssetPipe(root, mustOpenRoot(t, root), "/")
	at := Source{Path: "p.md", Line: 1}

	for _, alt := range []string{"The brief", `A "quoted" <brief> & more`} {
		out, err := pipe.render("docs/explanation", "../assets/img/x.svg", alt, at)
		if err != nil {
			t.Fatalf("a drawing was refused: %v", err)
		}
		name, exposed, drawingHidden := inlinedName(t, out)
		if !exposed || name != alt {
			t.Errorf("alt %q: the picture's accessible name is %q (exposed %v), want the page's alt:\n%s", alt, name, exposed, out)
		}
		if !drawingHidden {
			t.Errorf("alt %q: the drawing inside is not hidden, so its own name reads beside the page's:\n%s", alt, out)
		}
	}

	out, err := pipe.render("docs/explanation", "../assets/img/x.svg", "", at)
	if err != nil {
		t.Fatalf("a drawing was refused: %v", err)
	}
	if name, exposed, _ := inlinedName(t, out); exposed {
		t.Errorf("an empty alt is a decorative picture, yet it reads as %q:\n%s", name, out)
	}

	// Text before the root element stays where it was, and the hiding lands on
	// the root start tag alone.
	root = assetFixture(t, map[string]string{"docs/assets/img/x.svg": `aria-hidden="false" ` + svg})
	pipe = newAssetPipe(root, mustOpenRoot(t, root), "/")
	out, err = pipe.render("docs/explanation", "../assets/img/x.svg", "The brief", at)
	if err != nil {
		t.Fatalf("a drawing was refused: %v", err)
	}
	if !strings.Contains(out, `aria-hidden="false" <svg`) || strings.Count(out, `aria-hidden="false"`) != 1 || strings.Count(out, `aria-hidden="true"`) != 1 || !strings.Contains(out, `<rect width="2" height="2"/>`) {
		t.Errorf("the hiding reached past the root start tag:\n%s", out)
	}
}

// TestAssetsRootThatDeclaresItsOwnHiding pins the drawing whose root already
// says aria-hidden. Said "true", the root is left as it is, so the emitted page
// carries the attribute once and the page's own reader (parseHTML, which
// refuses a repeated attribute) reads it; the picture is still named by the
// page's alt. Said anything else, the drawing asks to be read beside the
// page's name for it, so the build refuses it and names the asset rather than
// letting the page fail later with a message that names no drawing.
func TestAssetsRootThatDeclaresItsOwnHiding(t *testing.T) {
	at := Source{Path: "p.md", Line: 1}
	page := func(body string) string {
		return "<!doctype html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n<title>t</title>\n</head>\n<body>\n" +
			body + "\n</body>\n</html>\n"
	}

	for _, alt := range []string{"The brief", ""} {
		svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 4 4" aria-hidden="true"><rect width="2" height="2"/></svg>`
		root := assetFixture(t, map[string]string{"docs/assets/img/x.svg": svg})
		pipe := newAssetPipe(root, mustOpenRoot(t, root), "/")
		out, err := pipe.render("docs/explanation", "../assets/img/x.svg", alt, at)
		if err != nil {
			t.Fatalf("alt %q: a drawing that hides itself was refused: %v", alt, err)
		}
		doc, err := parseHTML("index.html", page(out))
		if err != nil {
			t.Errorf("alt %q: the page's reader refused the inlined drawing: %v\n%s", alt, err, out)
			continue
		}
		drawing := findElement(doc, func(n *htmlNode) bool { return n.Name == "svg" })
		if drawing == nil || drawing.Attr("aria-hidden") != "true" {
			t.Errorf("alt %q: the drawing is not hidden:\n%s", alt, out)
		}
		name, exposed, _ := inlinedName(t, out)
		if alt != "" && (!exposed || name != alt) {
			t.Errorf("alt %q: the picture's accessible name is %q (exposed %v):\n%s", alt, name, exposed, out)
		}
		if alt == "" && exposed {
			t.Errorf("an empty alt is a decorative picture, yet it reads as %q:\n%s", name, out)
		}
	}

	for _, value := range []string{"false", "", "TRUE"} {
		svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 4 4" aria-hidden="` + value + `"><rect width="2" height="2"/></svg>`
		root := assetFixture(t, map[string]string{"docs/assets/img/x.svg": svg})
		pipe := newAssetPipe(root, mustOpenRoot(t, root), "/")
		out, err := pipe.render("docs/explanation", "../assets/img/x.svg", "The brief", at)
		if err == nil {
			t.Errorf("aria-hidden=%q on the root was inlined, though the page names the picture itself:\n%s", value, out)
			continue
		}
		if !strings.Contains(err.Error(), "docs/assets/img/x.svg") || !strings.Contains(err.Error(), "aria-hidden") {
			t.Errorf("aria-hidden=%q: the refusal names neither the asset nor the attribute: %v", value, err)
		}
	}

	// Below the root the attribute means nothing the page reads: the whole
	// drawing is hidden, so a part of it saying "false" is left as it is.
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 4 4"><g aria-hidden="false"><rect width="2" height="2"/></g></svg>`
	root := assetFixture(t, map[string]string{"docs/assets/img/x.svg": svg})
	pipe := newAssetPipe(root, mustOpenRoot(t, root), "/")
	out, err := pipe.render("docs/explanation", "../assets/img/x.svg", "The brief", at)
	if err != nil {
		t.Fatalf("a drawing with aria-hidden below its root was refused: %v", err)
	}
	if _, err := parseHTML("index.html", page(out)); err != nil {
		t.Errorf("the page's reader refused the inlined drawing: %v\n%s", err, out)
	}
}

// TestAssetsFoldTheCaseOfARootsHiding pins that the build reads an attribute
// name the way a browser does, folding its case. A root that says aria-HIDDEN
// says aria-hidden: "true" is left as it is, so the page carries one such
// attribute, and "false" is refused at the asset like its lower-case twin.
func TestAssetsFoldTheCaseOfARootsHiding(t *testing.T) {
	at := Source{Path: "p.md", Line: 1}
	page := func(body string) string {
		return "<!doctype html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n<title>t</title>\n</head>\n<body>\n" +
			body + "\n</body>\n</html>\n"
	}
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 4 4" aria-HIDDEN="true"><rect width="2" height="2"/></svg>`
	root := assetFixture(t, map[string]string{"docs/assets/img/x.svg": svg})
	pipe := newAssetPipe(root, mustOpenRoot(t, root), "/")
	out, err := pipe.render("docs/explanation", "../assets/img/x.svg", "The brief", at)
	if err != nil {
		t.Fatalf("a drawing that hides itself was refused: %v", err)
	}
	doc, err := parseHTML("index.html", page(out))
	if err != nil {
		t.Fatalf("the page's reader refused the inlined drawing: %v\n%s", err, out)
	}
	drawing := findElement(doc, func(n *htmlNode) bool { return n.Name == "svg" })
	var hiding int
	for name := range drawing.Attrs {
		if strings.EqualFold(name, "aria-hidden") {
			hiding++
		}
	}
	if hiding != 1 {
		t.Errorf("the root carries %d attributes a browser reads as aria-hidden, want 1:\n%s", hiding, out)
	}

	svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 4 4" aria-HIDDEN="false"><rect width="2" height="2"/></svg>`
	root = assetFixture(t, map[string]string{"docs/assets/img/x.svg": svg})
	pipe = newAssetPipe(root, mustOpenRoot(t, root), "/")
	if out, err := pipe.render("docs/explanation", "../assets/img/x.svg", "The brief", at); err == nil {
		t.Errorf("aria-HIDDEN=\"false\" on the root was inlined:\n%s", out)
	} else if !strings.Contains(err.Error(), "docs/assets/img/x.svg") {
		t.Errorf("the refusal does not name the asset: %v", err)
	}
}

// TestAssetsRefuseARepeatedAttribute pins that a start element naming one
// attribute twice is refused at the asset. The XML decoder admits the repeat,
// the page's reader does not, and a browser keeps the first and drops the
// rest, so the drawing would say one thing to the check and another to the
// page. Names are compared as a browser compares them, case folded, within
// their namespace, on any element.
func TestAssetsRefuseARepeatedAttribute(t *testing.T) {
	at := Source{Path: "p.md", Line: 1}
	long := strings.Repeat("v", 300)
	for _, tc := range []struct{ svg, attr string }{
		{`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 4 4" aria-hidden="true" aria-hidden="true"><rect width="2" height="2"/></svg>`, "aria-hidden"},
		{`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 4 4" aria-HIDDEN="false" aria-hidden="true"><rect width="2" height="2"/></svg>`, "aria-hidden"},
		{`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 4 4"><rect width="2" width="3" height="2"/></svg>`, "width"},
		{`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 4 4"><g aria-label="a" aria-LABEL="` + long + `"><rect width="2" height="2"/></g></svg>`, "aria-label"},
	} {
		root := assetFixture(t, map[string]string{"docs/assets/img/x.svg": tc.svg})
		pipe := newAssetPipe(root, mustOpenRoot(t, root), "/")
		out, err := pipe.render("docs/explanation", "../assets/img/x.svg", "The brief", at)
		if err == nil {
			t.Errorf("a repeated %s was inlined:\n%s", tc.attr, out)
			continue
		}
		msg := err.Error()
		if !strings.Contains(msg, "docs/assets/img/x.svg") || !strings.Contains(strings.ToLower(msg), tc.attr) {
			t.Errorf("the refusal names neither the asset nor %s: %v", tc.attr, err)
		}
		if strings.Contains(msg, long) {
			t.Errorf("the refusal echoes a value unclipped: %v", err)
		}
	}

	// A long repeated name is echoed clipped as well: the repeat message names
	// the attribute, never its value, so the name is what must stay bounded.
	longName := "aria-" + long
	nameSVG := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 4 4"><g ` + longName + `="a" ` + longName + `="b"><rect width="2" height="2"/></g></svg>`
	nameRoot := assetFixture(t, map[string]string{"docs/assets/img/x.svg": nameSVG})
	namePipe := newAssetPipe(nameRoot, mustOpenRoot(t, nameRoot), "/")
	if _, err := namePipe.render("docs/explanation", "../assets/img/x.svg", "The brief", at); err == nil || strings.Contains(err.Error(), longName) {
		t.Errorf("a long repeated attribute name was not refused with its name clipped: %v", err)
	}

	// A long aria-hidden value on the root is echoed clipped too.
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 4 4" aria-hidden="` + long + `"><rect width="2" height="2"/></svg>`
	root := assetFixture(t, map[string]string{"docs/assets/img/x.svg": svg})
	pipe := newAssetPipe(root, mustOpenRoot(t, root), "/")
	if _, err := pipe.render("docs/explanation", "../assets/img/x.svg", "The brief", at); err == nil || strings.Contains(err.Error(), long) {
		t.Errorf("a long aria-hidden value was not refused with its value clipped: %v", err)
	}
}

// TestAssetsStripKeepsAttributesSeparate pins that the build's rewrites of a
// drawing's root start tag keep every attribute separate from the element name
// and from its neighbours. The XML decoder reads an attribute written straight
// after the closing quote of the one before it, so the size strip, which takes
// the whitespace before a size attribute with it, puts one back where the next
// byte would otherwise touch what came before. Read back through the page's
// reader, the root is an svg element carrying the attributes it was given,
// less its size, plus the build's aria-hidden. The asset check also asks a
// drawing to separate its own attributes, so the page reads them as written.
func TestAssetsStripKeepsAttributesSeparate(t *testing.T) {
	page := func(body string) string {
		return "<!doctype html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n<title>t</title>\n</head>\n<body>\n" +
			body + "\n</body>\n</html>\n"
	}
	attrsOf := func(t *testing.T, out string) (string, []string) {
		t.Helper()
		doc, err := parseHTML("index.html", page(out))
		if err != nil {
			t.Errorf("the page's reader refused the rewritten drawing: %v\n%s", err, out)
			return "", nil
		}
		body := findElement(doc, func(n *htmlNode) bool { return n.Name == "body" })
		var first *htmlNode
		for _, c := range body.Children {
			if c.Kind == htmlElementNode {
				first = c
				break
			}
		}
		if first == nil {
			t.Errorf("no element in the rewritten drawing:\n%s", out)
			return "", nil
		}
		var names []string
		for name := range first.Attrs {
			names = append(names, name)
		}
		sort.Strings(names)
		return first.Name, names
	}

	for _, tc := range []struct{ svg, want string }{
		{`<svg xmlns="http://www.w3.org/2000/svg" width="4"viewBox="0 0 4 4" height="4"aria-label="d"><rect width="2" height="2"/></svg>`,
			"aria-hidden aria-label viewBox xmlns"},
		{`<svg width="4"xmlns="http://www.w3.org/2000/svg"><rect width="2" height="2"/></svg>`,
			"aria-hidden xmlns"},
		{`<svg xmlns="http://www.w3.org/2000/svg" width="4" height="4"><rect width="2" height="2"/></svg>`,
			"aria-hidden xmlns"},
		{`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 4 4" width="4"/>`,
			"aria-hidden viewBox xmlns"},
		{`<svg>
<rect width="2" height="2"/></svg>`, "aria-hidden"},
	} {
		out := hideRootDrawing(stripRootSize(tc.svg))
		name, attrs := attrsOf(t, out)
		if attrs == nil && name == "" {
			continue
		}
		if name != "svg" || strings.Join(attrs, " ") != tc.want {
			t.Errorf("the rewritten root is <%s> carrying %v, want <svg> carrying %s:\n%s", name, attrs, tc.want, out)
		}
	}

	// At the asset, a drawing that does not separate its own attributes is
	// refused, naming the asset.
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 4 4"><rect width="2"height="2"/></svg>`
	root := assetFixture(t, map[string]string{"docs/assets/img/x.svg": svg})
	pipe := newAssetPipe(root, mustOpenRoot(t, root), "/")
	if out, err := pipe.render("docs/explanation", "../assets/img/x.svg", "The brief", Source{Path: "p.md", Line: 1}); err == nil {
		t.Errorf("a drawing whose attributes touch was inlined:\n%s", out)
	} else if !strings.Contains(err.Error(), "docs/assets/img/x.svg") {
		t.Errorf("the refusal does not name the asset: %v", err)
	}
}
