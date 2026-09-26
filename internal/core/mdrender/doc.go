// Package mdrender is the Markdown subset renderer the site publishes the
// repository's prose through, and the one answer to what that subset refuses.
//
// It is a leaf — it imports only core/mdrecord, for the fence reading its block
// walk shares with every other record reader — because two kinds of caller need
// it. The site (core/site) renders every page through it. And the writers of
// append-only record prose ask it, before they write, whether the site will
// render what they are about to write: core/site imports the record families
// that hold such prose, so they could not import it back, and a second copy of
// the renderer's rules in each writer is a copy that drifts from the renderer
// (iss-2608301646046226). RefusalIn is that question.
package mdrender
