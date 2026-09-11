package site

// Pictures.
//
// adr-47 decision 2: every picture is a committed asset under `docs/assets/img/`,
// referenced from a docs page like any other image, and **the build never
// draws**. So this file does two things and refuses a third. An SVG is INLINED,
// because its colours are written as `var(--token, fallback)` and inlining is
// what lets them follow the reader's theme; the width and height of its ROOT
// element are stripped so the stylesheet sizes it, and every inner element's
// size is left alone, because that size is the drawing. A raster is COPIED
// VERBATIM into the output tree and referenced — no re-encoding, because
// optimisation would mean a dependency and a decision nobody has made yet.
//
// An image the page names but the repository does not carry is a build error.
// The alternative is a broken image on a published page, which nobody notices
// until a reader does.
//
// Two refusals guard the pair of holes an image reference opens, and both are
// about the same thing: this file turns repository CONTENT into published
// OUTPUT, and repository content is written by whoever can land a pull request.
//
//  1. A reference is a PATH, and a path read out of prose will name whatever it
//     is pointed at. `![x](../../.git/config)` is a perfectly ordinary-looking
//     image reference that publishes git's config — which on a CI runner carries
//     the checkout token — into a public directory, under a name nobody reads
//     twice, in a tree git never reports as changed. So a reference must resolve
//     inside the asset root and carry a picture's extension, and anything else
//     is refused by name.
//
//  2. An SVG is INLINED, which is the whole point of committing drawings as SVG
//     — but inlining means its bytes become markup. A `<script>` element or an
//     `on*` attribute inside one is executable code on the site's own origin,
//     reached with no click, in a file that is not code and that no reviewer
//     reads as code. So an SVG is parsed and held to an allowlist of the
//     elements and attributes a drawing is made of — a CLOSED list on both
//     sides, because the attribute that matters is the one nobody thought of.
//     `style` and `class` carry no script and are still not a drawing's: inline
//     CSS on the root beats `.svgasset svg` and `.card .icon svg`, neither of
//     which is `!important`, and a child that positions itself `fixed` covers
//     the page it was drawn into. The drawings' theme colours ride on the paint
//     attributes as `var(--token, fallback)` values, so nothing legitimate is
//     lost by refusing both.

import (
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// maxAssetBytes bounds one asset read.
const maxAssetBytes = 16 << 20

// assetOutDir is where copied rasters land in the output tree, and the prefix
// the page references them by. It is relative, so the page works from any mount
// point.
const assetOutDir = "assets/img"

// assetRootPrefix is the only directory a picture may come from. adr-47
// decision 2 says every picture is a committed asset under `docs/assets/img/`,
// referenced from a docs page like any other image; this is that sentence made
// enforceable.
const assetRootPrefix = "docs/assets/"

// pictureExts is what a picture's file name ends in. It is a closed list rather
// than a "not one of these dangerous ones" list, because the dangerous set is
// the whole rest of the repository.
var pictureExts = map[string]bool{
	".svg": true, ".png": true, ".jpg": true, ".jpeg": true,
	".gif": true, ".webp": true, ".avif": true,
}

// svgSizeRe matches the width/height attributes the stylesheet replaces. It is
// applied to the ROOT <svg> start tag alone (stripRootSize) — never to the
// document, where the same two attribute names mean something entirely
// different. `stroke-width` and `markerWidth` are not matched: the leading
// `\s` is what keeps a hyphenated or prefixed name out.
//
// The value stays `\d+` — the form every committed drawing writes. A wider
// class would match across a single-quoted attribute value holding a stray
// `"`, taking the value's closing quote from the NEXT attribute and rewriting
// the tag; digits cannot contain a quote, so this class cannot.
var svgSizeRe = regexp.MustCompile(`\s(width|height)="\d+"`)

// stripRootSize removes the ROOT <svg> element's width and height, so the
// stylesheet sizes the drawing, and leaves every other element's alone.
//
// On any other element those two attributes ARE the drawing, not its page size.
// A <rect> is a box of that size and collapses to nothing without it; an
// <image> panel is a raster framed to that size and clipped to it, and without
// it falls back to the intrinsic pixel size of the raster it embeds, bursting
// out of the clip path that framed it; a <use>, <pattern>, <mask> or <symbol>
// establishes a region of that size.
//
// The span of the root start tag comes from the same decoder that vetted the
// document, rather than from a search for the first `>` — an attribute value
// may contain one, and a size stripped out of the wrong span is exactly the
// class of bug this function exists to end. A document that does not parse
// (which checkInlinableSVG has already refused) is returned untouched.
func stripRootSize(svg string) string {
	dec := xml.NewDecoder(strings.NewReader(svg))
	dec.Strict = true
	dec.Entity = map[string]string{}
	for {
		// InputOffset is the boundary between the token just returned and the
		// next, so reading it on both sides of Token() brackets exactly the token
		// it returned — and nothing before it. Bracketing matters rather than
		// merely reading tidily: anything the decoder passed over on the way to
		// the root would otherwise be inside the replaced region, and a drawing
		// may carry text before its root element.
		start := int(dec.InputOffset())
		tok, err := dec.Token()
		if err != nil {
			return svg
		}
		if _, ok := tok.(xml.StartElement); !ok {
			continue
		}
		end := int(dec.InputOffset())
		if start < 0 || end > len(svg) || start >= end {
			return svg
		}
		return svg[:start] + svgSizeRe.ReplaceAllString(svg[start:end], "") + svg[end:]
	}
}

// svgElements is what a drawing is made of: shapes, the scaffolding that
// positions and paints them, and text. Anything else — a script, a stylesheet,
// an embedded document, a foreign namespace — is refused, because a drawing
// does not need it and an attack does.
var svgElements = map[string]bool{
	"svg": true, "g": true, "defs": true, "symbol": true, "title": true, "desc": true,
	"path": true, "rect": true, "circle": true, "ellipse": true, "line": true,
	"polyline": true, "polygon": true, "text": true, "tspan": true,
	"clipPath": true, "mask": true, "marker": true, "pattern": true, "use": true,
	"linearGradient": true, "radialGradient": true, "stop": true,
	// <image> is allowed only because the committed drawings place raster
	// panels inside them; its href is held to an embedded raster (below), never
	// to a fetch and never to another SVG.
	"image": true,
}

// embeddedRasterPrefixes are the only `data:` payloads an <image> may carry: a
// self-contained raster. `data:image/svg+xml` is deliberately absent — that is
// a document, not a picture, and nesting one inside a drawing reopens exactly
// the surface this check exists to close.
var embeddedRasterPrefixes = []string{
	"data:image/png;base64,", "data:image/jpeg;base64,", "data:image/gif;base64,",
	"data:image/webp;base64,", "data:image/avif;base64,",
}

// svgHrefAttrs are the attributes that point somewhere. Every one of them is
// held to a same-document fragment: a drawing refers to its own defs and to
// nothing else, so an absolute or relative URL here is either an external fetch
// from a page that makes none, or a scheme that executes.
var svgHrefAttrs = map[string]bool{"href": true, "xlink:href": true, "src": true}

// svgNamespace and xlinkNamespace are the two vocabularies a drawing declares:
// SVG itself, and XLink for the `xlink:href` an older exporter writes on <use>.
// A declaration of anything else is how a foreign vocabulary gets into a file
// that is inlined into the page, so it is refused rather than carried through.
const (
	svgNamespace   = "http://www.w3.org/2000/svg"
	xlinkNamespace = "http://www.w3.org/1999/xlink"
)

// svgAttrs is what a drawing is made of, attribute-side: the geometry that
// places a shape, the paint that fills and strokes it, the type that sets its
// text, the units and transforms that gradients, patterns, clips, masks and
// markers are configured by, and the identity a same-document reference points
// at. `aria-*` is allowed by prefix below, and the pointing attributes
// (href, xlink:href, src) are settled before this map is consulted.
//
// It is CLOSED, and that is the whole point: the old check refused `on*` and
// scheme-checked the pointing attributes, then returned nil for everything
// else, which made an allowlist in name a denylist in behaviour — `style` and
// `class` walked through it into the published page. An attribute nobody has
// weighed is now refused by name, so the failure is a build error naming the
// asset rather than live markup on the site.
//
// Every name the committed corpus carries is here (TestCommittedSVGAssets-
// AreDrawings inlines all 32 drawings through the real pipe, so a name dropped
// from this list fails that test rather than the site build). The rest is the
// standard presentation and geometry set, with three deliberate absences:
// `style` and `class`, for the reason at the top of this file, and `overflow`,
// which is the one presentation attribute that lets a drawing paint outside the
// viewport its root establishes.
var svgAttrs = map[string]bool{
	// Identity and the document's own frame.
	"id": true, "xmlns": true, "role": true, "version": true,
	"viewBox": true, "preserveAspectRatio": true,

	// Geometry: where a shape is and how big it is.
	"x": true, "y": true, "dx": true, "dy": true,
	"x1": true, "y1": true, "x2": true, "y2": true,
	"cx": true, "cy": true, "r": true, "rx": true, "ry": true,
	"width": true, "height": true, "d": true, "points": true,
	"transform": true, "pathLength": true,

	// Paint: fill, stroke and the opacity of either.
	"fill": true, "fill-opacity": true, "fill-rule": true,
	"stroke": true, "stroke-width": true, "stroke-opacity": true,
	"stroke-linecap": true, "stroke-linejoin": true, "stroke-miterlimit": true,
	"stroke-dasharray": true, "stroke-dashoffset": true,
	"opacity": true, "color": true, "paint-order": true, "vector-effect": true,
	"shape-rendering": true, "text-rendering": true, "image-rendering": true,
	"color-interpolation": true, "display": true, "visibility": true,

	// Type.
	"font-family": true, "font-size": true, "font-weight": true,
	"font-style": true, "font-variant": true, "font-stretch": true,
	"letter-spacing": true, "word-spacing": true, "text-decoration": true,
	"text-anchor": true, "dominant-baseline": true, "alignment-baseline": true,
	"baseline-shift": true, "writing-mode": true, "direction": true,
	"textLength": true, "lengthAdjust": true,

	// Clipping and masking — the referenced element is itself allowlisted.
	"clip-path": true, "clip-rule": true, "clipPathUnits": true,
	"mask": true, "maskUnits": true, "maskContentUnits": true,

	// Markers: the arrowheads the loop drawing puts on its paths.
	"marker-start": true, "marker-mid": true, "marker-end": true,
	"markerWidth": true, "markerHeight": true, "markerUnits": true,
	"refX": true, "refY": true, "orient": true,

	// Gradients and patterns.
	"gradientUnits": true, "gradientTransform": true, "spreadMethod": true,
	"fx": true, "fy": true, "fr": true, "offset": true,
	"stop-color": true, "stop-opacity": true,
	"patternUnits": true, "patternContentUnits": true, "patternTransform": true,
}

// checkInlinableSVG parses an SVG and refuses anything a drawing does not need.
// It parses rather than greps: entities, CDATA, comments and attribute quoting
// are exactly what a substring check gets wrong, and an XML decoder gets right.
func checkInlinableSVG(svg, rel string) error {
	bad := func(format string, args ...any) error {
		return fmt.Errorf("%s: %s — an inlined drawing becomes markup in the published page, so it is held to the elements and attributes a drawing is made of",
			rel, fmt.Sprintf(format, args...))
	}
	// A CDATA section is CharData to the decoder, indistinguishable from ordinary
	// text, so it is caught on the raw bytes before the token walk. Like a comment
	// and a processing instruction, it is inlined verbatim and the emitted page's
	// reader (htmlscan) refuses it — the build must refuse it here, at the asset,
	// rather than let the page fail the check with a message that names no file.
	if strings.Contains(svg, "<![CDATA[") {
		return bad("carries a CDATA section")
	}
	dec := xml.NewDecoder(strings.NewReader(svg))
	// Entity expansion is how an XML parser is made to read files it was never
	// given. A drawing needs no custom entity, so the map stays empty and any
	// reference to one is an error from the decoder itself.
	dec.Strict = true
	dec.Entity = map[string]string{}
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return bad("is not well-formed XML (%v)", err)
		}
		switch t := tok.(type) {
		case xml.ProcInst:
			// Including the `<?xml … ?>` declaration every SVG exporter writes: it
			// is inlined verbatim, and the emitted page carries no processing
			// instruction (htmlscan refuses one). Strip the prolog from the drawing.
			return bad("carries a processing instruction (<?%s?>); strip the XML prolog, as the inlined drawing becomes part of the page", t.Target)
		case xml.Comment:
			// Inlined verbatim, and the page carries no comment (htmlscan refuses
			// one, where unreviewed text would hide). Strip it from the drawing.
			return bad("carries a comment; strip it, as the inlined drawing becomes part of the page")
		case xml.Directive:
			// `<!DOCTYPE … [<!ENTITY x SYSTEM "file:///etc/passwd">]>` is the
			// classic external-entity read, and no drawing has a doctype.
			return bad("carries a document-type declaration")
		case xml.StartElement:
			name := t.Name.Local
			if !svgElements[name] {
				return bad("uses <%s>, which is not one of the drawing elements", name)
			}
			for _, a := range t.Attr {
				if err := checkSVGAttr(a, name, bad); err != nil {
					return err
				}
			}
		}
	}
}

// checkSVGAttr holds one attribute to the allowlist rules.
func checkSVGAttr(a xml.Attr, element string, bad func(string, ...any) error) error {
	local := a.Name.Local
	full := local
	if a.Name.Space != "" {
		// The decoder resolves a prefix to its namespace URL; for the reader's
		// sake report the conventional spelling.
		if strings.Contains(a.Name.Space, "xlink") {
			full = "xlink:" + local
		}
	}
	if len(local) >= 2 && strings.EqualFold(local[:2], "on") {
		return bad("sets %s on <%s>, which is an event handler", strings.ToLower(full), element)
	}
	if svgHrefAttrs[full] || svgHrefAttrs[local] {
		v := strings.TrimSpace(a.Value)
		if strings.HasPrefix(v, "#") {
			return nil
		}
		if element == "image" {
			for _, p := range embeddedRasterPrefixes {
				if strings.HasPrefix(v, p) {
					return nil
				}
			}
			return bad("points %s on <image> at %q; an embedded panel is a self-contained raster (data:image/png|jpeg|gif|webp|avif;base64,…)",
				full, clip(v))
		}
		return bad("points %s on <%s> at %q; a drawing refers only to its own definitions (#id)",
			full, element, clip(v))
	}
	// A namespace declaration — `xmlns` (which the decoder reports with an empty
	// space) or `xmlns:prefix` (reported with the space "xmlns"). Declaring a
	// second vocabulary is how markup from somewhere else gets into a file that
	// is inlined into the page, so the two a drawing needs are named and the
	// rest is refused.
	if a.Name.Space == "xmlns" || (a.Name.Space == "" && local == "xmlns") {
		if v := strings.TrimSpace(a.Value); v == svgNamespace || v == xlinkNamespace {
			return nil
		}
		return bad("declares the namespace %q on <%s>; a drawing is written in SVG, and in XLink for its <use> references, and in nothing else",
			clip(a.Value), element)
	}
	// Any other prefixed attribute. `xlink:href` is settled above and is the
	// only namespaced attribute a drawing carries; the rest of XLink says
	// nothing a picture needs, and a prefix the document never declared is the
	// shape of something trying to look like an attribute this check knows.
	if a.Name.Space != "" {
		return bad("sets the prefixed attribute %s on <%s>; the only namespaced attribute a drawing carries is xlink:href",
			full, element)
	}
	// `style` and `class` carry no script and still do not belong to a drawing:
	// inline CSS on the root outranks the stylesheet's own `.svgasset svg` and
	// `.card .icon svg` rules, and a child can position itself out of the box
	// the page laid out and over the page itself.
	if local == "style" || local == "class" {
		return bad("sets %s on <%s>, which styles the page rather than draws; a drawing's colours are var(--token, fallback) values on its paint attributes",
			local, element)
	}
	// ARIA is how a drawing names itself to a screen reader, and its names are
	// open-ended, so it is allowed by prefix rather than one at a time.
	if strings.HasPrefix(local, "aria-") {
		return nil
	}
	if svgAttrs[local] {
		return nil
	}
	return bad("sets %s on <%s>, which is not one of the attributes a drawing is made of",
		full, element)
}

// assetPipe resolves image references and records what the build must copy.
type assetPipe struct {
	repoRoot string
	// root is repoRoot as an os.Root containment scope: an asset read resolves
	// every path component inside it, so a symlinked ancestor cannot pull a
	// picture from outside the repository into the published tree (gh #487).
	root *os.Root
	// copies maps a repo-relative source to its output-relative destination.
	copies map[string]string
}

func newAssetPipe(repoRoot string, root *os.Root) *assetPipe {
	return &assetPipe{repoRoot: repoRoot, root: root, copies: map[string]string{}}
}

// Copies lists the rasters to copy, sorted, as (source, destination) pairs.
func (a *assetPipe) Copies() [][2]string {
	out := make([][2]string, 0, len(a.copies))
	for src, dst := range a.copies {
		out = append(out, [2]string{src, dst})
	}
	sort.Slice(out, func(i, j int) bool { return out[i][0] < out[j][0] })
	return out
}

// render turns one image reference into HTML. pageDir is the repo-relative
// directory of the page the reference was written in, so `../assets/img/x.png`
// resolves the way a reader on GitHub sees it.
func (a *assetPipe) render(pageDir, src, alt string, at Source) (string, error) {
	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") || strings.HasPrefix(src, "data:") {
		return "", &UnsupportedError{at.Path, at.Line, "remote image",
			"every picture is a committed asset under docs/assets/img/; " + quote(src) + " is fetched at render time"}
	}
	rel := path.Clean(path.Join(pageDir, src))
	if !fsutil.ValidRelPath(rel) {
		return "", fmt.Errorf("%s:%d: image %q resolves outside the repository", at.Path, at.Line, src)
	}
	// The build PUBLISHES what it reads here, so what may be read is a closed
	// set: a picture, under the asset root, and nothing else. Without this a
	// reference in prose reaches `.git/config`, `.env`, or the gitignored local
	// tier, and copies it into a directory served to the public.
	if !strings.HasPrefix(rel, assetRootPrefix) {
		return "", fmt.Errorf("%s:%d: image %q resolves to %q, outside %s — every picture is a committed asset under the asset root",
			at.Path, at.Line, src, rel, assetRootPrefix)
	}
	ext := strings.ToLower(path.Ext(rel))
	if !pictureExts[ext] {
		return "", fmt.Errorf("%s:%d: image %q is a %q file, which is not a picture the build publishes",
			at.Path, at.Line, src, ext)
	}
	data, err := fsutil.ReadGuardedInRoot(a.root, rel, maxAssetBytes)
	if err != nil {
		return "", fmt.Errorf("%s:%d: image %q (%s): %w", at.Path, at.Line, src, rel, err)
	}

	name := path.Base(rel)
	if ext == ".svg" {
		if err := checkInlinableSVG(string(data), rel); err != nil {
			return "", fmt.Errorf("%s:%d: %w", at.Path, at.Line, err)
		}
		svg := stripRootSize(string(data))
		stem := strings.TrimSuffix(name, ".svg")
		return `<span class="svgasset ` + escapeAttr(stem) + `" data-asset="` + escapeAttr(rel) + `">` + svg + `</span>`, nil
	}

	dst := assetOutDir + "/" + name
	if prev, ok := a.copies[rel]; ok {
		dst = prev
	} else {
		// Two different files with the same basename would silently overwrite one
		// another in the flat output directory. The clash is reported against the
		// LOWEST-sorting other source rather than whichever the map happened to
		// yield first, so two identical builds name the same pair.
		clash := ""
		for other, taken := range a.copies {
			if taken == dst && other != rel && (clash == "" || other < clash) {
				clash = other
			}
		}
		if clash != "" {
			return "", fmt.Errorf("%s:%d: assets %q and %q share a file name; the output tree is flat", at.Path, at.Line, clash, rel)
		}
		a.copies[rel] = dst
	}
	size := ""
	if w, h, ok := pngSize(data); ok {
		size = fmt.Sprintf(` width="%d" height="%d"`, w, h)
	}
	// The href is root-absolute, because the same asset is referenced from every
	// depth the site serves: the landing page at `/`, and a record's page at
	// `/record/adr/adr-1/`. An output-relative src resolves only at the root and
	// 404s everywhere else — a picture missing on hundreds of pages and present
	// on the one page anybody previews.
	return `<img src="/` + escapeAttr(dst) + `" alt="` + escapeAttr(alt) + `"` + size + ` loading="lazy">`, nil
}

// pngSize reads a PNG's pixel dimensions out of its IHDR chunk, so the page can
// reserve the right box and not reflow when the image arrives. A format this
// does not understand simply gets no attributes.
func pngSize(data []byte) (int, int, bool) {
	const header = "\x89PNG\r\n\x1a\n"
	if len(data) < 24 || string(data[:8]) != header || string(data[12:16]) != "IHDR" {
		return 0, 0, false
	}
	w := binary.BigEndian.Uint32(data[16:20])
	h := binary.BigEndian.Uint32(data[20:24])
	if w == 0 || h == 0 {
		return 0, 0, false
	}
	return int(w), int(h), true
}
