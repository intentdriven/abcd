package site

// The site renders repository prose through the Markdown subset renderer in
// core/mdrender. The renderer lives there, outside this package, so the writers
// of append-only record prose can ask it what it refuses before they write:
// this package imports the record families that hold such prose, and they could
// not import it back (iss-2608301646046226).
//
// What stays here is the site's side of the renderer: its interface strings,
// which come from the site's closed allowlist, and the names this package's
// other files already use for the renderer's types and helpers.

import "github.com/intentdriven/abcd/internal/core/mdrender"

// UnsupportedError is a markdown construct outside the rendered subset.
type UnsupportedError = mdrender.UnsupportedError

// Source is a position in a repository file.
type Source = mdrender.Source

// Block is one top-level markdown block.
type Block = mdrender.Block

// Renderer turns the markdown subset into the site's HTML. Its two hooks are
// the places the site differs from a generic renderer: an image becomes a
// committed asset (inlined SVG or copied raster) rather than a bare <img>, and
// a repo-relative link becomes a site route.
type Renderer struct {
	// UI is the closed allowlist of interface strings; the renderer adds the
	// copy-button label and nothing else.
	UI UI
	// Image renders one image reference. src is as written in the markdown,
	// relative to the page; alt is its alt text.
	Image func(src, alt string, at Source) (string, error)
	// Link rewrites one href. It never fails: an href it does not recognise is
	// left exactly as the record wrote it.
	Link func(href string, at Source) string
	// Refs are the document's link reference definitions, from LinkDefinitions.
	// A nil map means the document defines none, and every reference link in it
	// is a refusal.
	Refs map[string]string
}

// RenderBlocks renders a block sequence in order.
func (r *Renderer) RenderBlocks(path string, blocks []Block) (string, error) {
	return r.subset().RenderBlocks(path, blocks)
}

// RenderBlock renders one top-level block.
func (r *Renderer) RenderBlock(path string, blk Block) (string, error) {
	return r.subset().RenderBlock(path, blk)
}

// inline renders one span of prose — a heading's title — on its own.
func (r *Renderer) inline(at Source, s string) (string, error) {
	return r.subset().Inline(at, s)
}

// subset is the renderer this one configures.
func (r *Renderer) subset() *mdrender.Renderer {
	return &mdrender.Renderer{
		Labels: mdrender.Labels{Copy: r.UI.Copy, Copied: r.UI.Copied},
		Image:  r.Image,
		Link:   r.Link,
		Refs:   r.Refs,
	}
}

// LinkDefinitions collects a document's link reference definitions.
func LinkDefinitions(md string) map[string]string { return mdrender.LinkDefinitions(md) }

// Blocks splits a section body into its top-level blocks.
func Blocks(md string, start int) []Block { return mdrender.Blocks(md, start) }

// The renderer's patterns and helpers, under the names the site's own pages use.
var (
	headingRe     = mdrender.HeadingRe
	orderedItemRe = mdrender.OrderedItemRe
	voidElements  = mdrender.VoidElements
)

func escapeText(s string) string                  { return mdrender.EscapeText(s) }
func escapeAttr(s string) string                  { return mdrender.EscapeAttr(s) }
func executableScheme(href string) (string, bool) { return mdrender.ExecutableScheme(href) }
func clip(s string) string                        { return mdrender.Clip(s) }
func quote(s string) string                       { return mdrender.Quote(s) }
func isUnorderedItem(ln string) bool              { return mdrender.IsUnorderedItem(ln) }
func indentOf(s string) int                       { return mdrender.IndentOf(s) }
func isSpace(c byte) bool                         { return mdrender.IsSpace(c) }
