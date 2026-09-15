package site

// `abcd site check` — the gates adr-47 decision 3 arms.
//
// The build renders; this says whether what it rendered may be published. Seven
// independent checks run over an already-built output directory and the
// repository it came from, each reporting EVERY failure it finds rather than
// the first, because a gate that stops at one finding turns a single review into
// as many rounds as there are mistakes.
//
//  1. Provenance — every visible word on a composed surface is either inside an
//     element naming the repository span it came from, or is one of the four
//     things the single-source rule lets the generator add.
//  2. Hero-vs-Identity — the rendered hero is the Identity block, read through
//     the canonical parser rather than compared as bytes.
//  3. Banned tokens over composed text — the docs-lint bans, run over the words
//     the site publishes, whichever tree they came from.
//  4. Snippet pinning — every `abcd …` command the site shows names a command
//     the generated CLI reference documents.
//  5. Baseline ratchet — unresolved references outside the committed baseline
//     fail; a baseline entry whose reference has been fixed is INVITED to be
//     removed, and says so without failing.
//  6. Static mobile checks — viewport, overflow containers, image constraint,
//     and inline widths, over every emitted page.
//  7. Loop-figure labels — every label in a page's lifted diagram is a phrase on
//     that page, so a drawing cannot drift from the prose beside it.
//
// # Scope
//
// adr-47 decision 3 scopes the banned-token gate — and, with it, the provenance
// walk — to the COMPOSED SURFACES: `/` and every span selected through
// `.abcd/site.json`, whichever tree the span came from. The verbatim record
// rendering under `/record/**` is EXEMPT: the record legitimately contains
// change-narration and tool names, and rewriting it to read better on the site
// is the thing that ADR forbids. The static mobile checks are the exception
// that covers everything THIS BUILD writes, record pages included: a table that
// scrolls off a phone does so whatever tree its words came from.
//
// `/docs/` is MkDocs' own tree and is not this build's output at all, so it is
// excluded from the page walk before any gate sees it — including the mobile
// checks, which therefore say nothing about it. Its words are gated at the
// source by docs-lint, and its layout is the SSG's own responsibility. A green
// report here is a statement about the pages abcd renders, and about no others.
//
// The explorer's two non-record pages — `/contributors/` and `/references/` —
// are held to the composed rules. `/contributors/` plainly is one: it quotes the
// attribution policy span the manifest selects. `/references/` selects no span
// through the manifest, and the reading that would therefore exempt it is
// rejected on purpose: it would let any future page escape the single-source
// rule by the simple expedient of not being named in the manifest, which is the
// opposite of what that rule is for. Both pages render repository prose outside
// the record's verbatim rendering, and brief 04-surfaces/22-site.md says every
// rendered block names its source.
//
// # The attribution escape
//
// adr-47 decision 3 grants one carve-out inside that scope: the contributors
// page prints model names under the declared attribution escape. The convention
// behind it (AGENTS.md) is that naming a tool is confined to CREDIT, so the
// escape is scoped to credit rather than to a page — a span selected from
// `ACKNOWLEDGEMENTS.md`, whose whole purpose is to say which tools and sources
// an idea came from, and the attribution page's own authorship data, which is
// git's record of what assisted rather than anything anybody wrote.
//
// It is an escape from the tool-NAMING bans, never from provenance: every name
// on that page is still matched against the trailers and contributors the
// history actually carries (`attributionWords`), so a name no commit declared
// fails like any other untraceable text.
//
// # The allow-escape, across the lift
//
// docs-lint's escape is a line-level HTML comment. A composed page cannot carry
// one usefully — the comment is not in the rendered markdown subset, and a
// reader would never see it — so the escape is read SOURCE-SIDE: text selected
// from a span is linted with that span's own lines in hand, and a match is
// exempt when some line of the source span both contains the matched text and
// carries the token's allow context (`lint.TokenChecker.LintComposed`).
//
// That is LOOSER than the file walk, and the looseness is worth stating rather
// than glossing. The walk decides per LINE; this decides per SPAN, because
// composed text no longer knows which of its span's lines it came from. A span
// selected by heading is one section, so an escape's reach is that section; a
// span with no anchor is the whole file, and one escape anywhere in it excuses
// that token in everything composed from that file. What the mechanism does
// guarantee is the direction: composing a sentence never HARDENS a ban its own
// file declares legitimate, which is the failure that would otherwise arrive as
// a build breaking over text nobody changed.
//
// # What the provenance walk proves, and what it does not
//
// It proves that every visible word is INSIDE a block that names a repository
// span, and that the span RESOLVES — the file is there and the heading anchor
// is a heading in it. It does not prove the words are the span's own words. It
// cannot: the composer renders markdown, and a card's text is a transformation
// of its source (emphasis markers gone, entities decoded, a lead-in split from
// its paragraph, a table cell lifted out of its row), so byte containment is
// false for text that is perfectly well sourced.
//
// What closes that gap is the other end: the composer is the only thing that
// writes `data-src`, and it writes it from the span it is rendering. A block
// carrying a resolvable attribute and someone else's words would have to be
// composed by code that does that on purpose, which is a change to the
// generator and not a change to a page. Held together, the walk refuses text
// with no source and the composer refuses to invent one.
//
// # What the generator may add
//
// The single-source rule (adr-47 decision 2) lets four kinds of word onto the
// page beside the selected spans: an interface string from `site-src/ui.json`,
// a number, a date, a file name and an asset name. Two documented additions sit
// beside them, and each is VERIFIED against the repository file it comes from
// rather than waved through:
//
//   - `<title>` and `<meta name="description">` carry Identity text with no
//     `data-src`, because neither element can hold visible text a provenance
//     walk would reach (brief 04-surfaces/22-site.md). They are checked against
//     the Identity block instead of skipped.
//   - The package's own metadata — its name, its author, its licence and its
//     forge handle — reaches the brand and the footer from
//     `.claude-plugin/plugin.json`, and a chapter's letter reaches its heading
//     from `.abcd/site.json`. Neither is prose written for the site; both are
//     repository DATA, and both are matched against the file that declares them.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/core/positioning"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// The check names, as they appear in a finding and in the JSON envelope.
const (
	CheckProvenance   = "provenance"
	CheckHero         = "hero"
	CheckBannedTokens = "banned-tokens"
	CheckSnippets     = "snippets"
	CheckBaseline     = "baseline"
	CheckMobile       = "mobile"
	CheckFigureLabels = "figure-labels"
)

// CheckNames is every check this verb runs, in the order it runs them.
var CheckNames = []string{
	CheckProvenance, CheckHero, CheckBannedTokens, CheckSnippets,
	CheckBaseline, CheckMobile, CheckFigureLabels,
}

// maxOutputPageBytes bounds one emitted page read.
const maxOutputPageBytes = 32 << 20

// maxStylesheetBytes bounds one linked stylesheet read.
const maxStylesheetBytes = 4 << 20

// recordRoutePrefix is the verbatim record rendering, exempt from the
// composed-surface checks by adr-47 decision 3.
const recordRoutePrefix = "record/"

// homePageRel is the landing page the manifest's `home` block composes.
const homePageRel = "index.html"

// attributionPageRel is the attribution surface: the page that credits who
// wrote the history and what assisted.
const attributionPageRel = routeContributors + "index.html"

// docsRoutePrefix is MkDocs' own tree. This build does not write it; the prefix
// is named so a future co-located render is excluded rather than silently
// checked against rules that were never written for it.
const docsRoutePrefix = "docs/"

// CheckRequest is one run of the gates.
type CheckRequest struct {
	// RepoRoot is the repository the output was rendered from.
	RepoRoot string
	// OutDir is the built output directory, absolute or relative to the caller.
	OutDir string
}

// CheckFinding is one failure. Every finding names the check that raised it and
// the place a reader's next edit goes.
type CheckFinding struct {
	Check string `json:"check"`
	// Where is the emitted file, or the repository file, the finding is about.
	Where string `json:"where"`
	// Source is the `data-src` span the offending content came from, where the
	// finding is about composed content.
	Source string `json:"source,omitempty"`
	// Detail says what is wrong, in the reader's terms.
	Detail string `json:"detail"`
}

// CheckResult is what one run found.
type CheckResult struct {
	OutDir string `json:"out_dir"`
	// Pages are the emitted HTML files read, output-relative and sorted.
	Pages []string `json:"pages"`
	// Composed are the subset held to the composed-surface rules.
	Composed []string `json:"composed"`
	// Checks are the gates that ran.
	Checks []string `json:"checks"`
	// Findings fail the run.
	Findings []CheckFinding `json:"findings"`
	// Notes do not. The ratchet's shrink invitation is the only one: a baseline
	// entry whose reference has been fixed is news, not a failure.
	Notes []CheckFinding `json:"notes"`
	// Built reports whether this run rendered the output itself.
	Built bool `json:"built"`
}

// OK reports whether the run found nothing that fails it.
func (r CheckResult) OK() bool { return len(r.Findings) == 0 }

// checker carries everything the seven checks read, gathered once.
type checker struct {
	repoRoot string
	// root is repoRoot as an os.Root containment scope, shared by every read of a
	// repo-relative source below, so a symlinked ancestor cannot walk a check read
	// outside the repository — the same containment the build's reads carry (gh
	// #487).
	root     *os.Root
	outDir   string
	manifest Manifest
	ui       UI
	repo     RepoMeta
	identity positioning.Block
	hasBlock bool
	tokens   *lint.TokenChecker

	// pages are the parsed emitted documents, keyed by output-relative path.
	pages map[string]*htmlNode
	// order is the sorted page list, so every run reports in one order.
	order []string
	// sources caches one read of each repository file a data-src names.
	sources map[string]*sourceFile
	// styles are the parsed stylesheets the pages link, keyed by output path.
	styles map[string]*styleSheet

	res *CheckResult
}

// sourceFile is one repository file a composed span was selected from.
type sourceFile struct {
	Lines    []string
	Sections []Section
	Missing  bool
}

// Check runs every gate against an already-built output directory.
//
// An output directory with no `index.html` is BUILT first rather than refused:
// the check is about what the repository publishes, and a caller who has not
// rendered yet is asking the same question as one who has.
func Check(req CheckRequest) (CheckResult, error) {
	repoRoot := req.RepoRoot
	// The same gate the build's writes pass: the check reads the directory it
	// is handed, and builds into it when it holds no index.html.
	outDir, err := resolveOutDir(repoRoot, req.OutDir)
	if err != nil {
		return CheckResult{}, err
	}

	// Every collection is non-nil from the start: the JSON envelope is a machine
	// surface, and a reader that has to tell `null` from `[]` is a reader that
	// will one day get it wrong.
	res := CheckResult{
		OutDir: outDir, Checks: CheckNames,
		Pages: []string{}, Composed: []string{},
		Findings: []CheckFinding{}, Notes: []CheckFinding{},
	}
	if ok, _ := fsutil.Exists(filepath.Join(outDir, "index.html")); !ok {
		if _, err := Build(Request{RepoRoot: repoRoot, OutDir: outDir}); err != nil {
			return CheckResult{}, err
		}
		res.Built = true
	}

	manifest, err := LoadManifest(repoRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return CheckResult{}, ErrNoManifest
		}
		return CheckResult{}, err
	}
	ui, err := LoadUI(repoRoot, manifest.UIStrings)
	if err != nil {
		return CheckResult{}, err
	}
	repo, err := LoadRepoMeta(repoRoot)
	if err != nil {
		return CheckResult{}, err
	}
	docsRoot, err := os.OpenRoot(repoRoot)
	if err != nil {
		return CheckResult{}, err
	}
	docsCfg, err := lint.LoadConfigInRoot(docsRoot, docsLintConfigRelPath)
	docsRoot.Close()
	if err != nil && !os.IsNotExist(err) {
		return CheckResult{}, err
	}
	tokens, err := lint.NewTokenChecker(docsCfg.BannedTokens)
	if err != nil {
		return CheckResult{}, err
	}

	srcRoot, err := os.OpenRoot(repoRoot)
	if err != nil {
		return CheckResult{}, err
	}
	defer srcRoot.Close()

	c := &checker{
		repoRoot: repoRoot, root: srcRoot, outDir: outDir, manifest: manifest, ui: ui, repo: repo,
		tokens: tokens, pages: map[string]*htmlNode{}, sources: map[string]*sourceFile{},
		styles: map[string]*styleSheet{}, res: &res,
	}
	c.identity, c.hasBlock = readIdentity(repoRoot, manifest)

	if err := c.readPages(); err != nil {
		return CheckResult{}, err
	}
	c.checkProvenance()
	c.checkHero()
	c.checkBannedTokens()
	c.checkSnippets()
	if err := c.checkBaseline(); err != nil {
		return CheckResult{}, err
	}
	c.checkMobile()
	c.checkFigureLabels()

	sortFindings(res.Findings)
	sortFindings(res.Notes)
	return res, nil
}

// docsLintConfigRelPath is the documentation lint whose bans the composed text
// is held to. The site publishes text from several trees; this is the config
// that says which words `docs/` may not carry, and adr-47 decision 3 extends it
// to every word the site composes.
const docsLintConfigRelPath = ".abcd/docs-lint.json"

// readIdentity resolves the canonical Identity block, or reports its absence.
func readIdentity(repoRoot string, m Manifest) (positioning.Block, bool) {
	blk, err := positioning.ParseBlock(repoRoot, positioning.BlockLocation{
		File: m.Identity.File, Heading: m.Identity.Heading,
	})
	if err != nil {
		return positioning.Block{}, false
	}
	return blk, true
}

// fail records a finding.
func (c *checker) fail(check, where, source, format string, args ...any) {
	c.res.Findings = append(c.res.Findings, CheckFinding{
		Check: check, Where: where, Source: source, Detail: fmt.Sprintf(format, args...),
	})
}

// note records something worth saying that does not fail the run.
func (c *checker) note(check, where, format string, args ...any) {
	c.res.Notes = append(c.res.Notes, CheckFinding{
		Check: check, Where: where, Detail: fmt.Sprintf(format, args...),
	})
}

// sortFindings puts a run's output in one order, whatever order the walk found
// it in: two identical runs must print the same report.
func sortFindings(f []CheckFinding) {
	sort.SliceStable(f, func(i, j int) bool {
		if f[i].Check != f[j].Check {
			return f[i].Check < f[j].Check
		}
		if f[i].Where != f[j].Where {
			return f[i].Where < f[j].Where
		}
		if f[i].Source != f[j].Source {
			return f[i].Source < f[j].Source
		}
		return f[i].Detail < f[j].Detail
	})
}

// readPages parses every emitted HTML file. A page the tokenizer refuses is a
// FINDING rather than a fault: the run carries on and reports everything else,
// which is what a gate reporting every failure has to do.
func (c *checker) readPages() error {
	var rels []string
	err := filepath.WalkDir(c.outDir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".html") {
			return nil
		}
		rel, rerr := filepath.Rel(c.outDir, p)
		if rerr != nil {
			return rerr
		}
		// `docs/` is MkDocs' own tree, not this build's output: its pages carry
		// HTML comments the generator's grammar refuses, and every rule here
		// was written for generator output. It is excluded from the walk — the
		// declared scope above — rather than parsed into findings; its words
		// are gated at the source by docs-lint.
		if strings.HasPrefix(filepath.ToSlash(rel), docsRoutePrefix) {
			return nil
		}
		rels = append(rels, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return err
	}
	sort.Strings(rels)
	for _, rel := range rels {
		data, err := fsutil.ReadGuarded(filepath.Join(c.outDir, filepath.FromSlash(rel)), maxOutputPageBytes)
		if err != nil {
			return err
		}
		doc, err := parseHTML(rel, string(data))
		if err != nil {
			var f *htmlFault
			if errors.As(err, &f) {
				c.fail(CheckProvenance, rel, "", "%s", f.Reason)
				// A page that did not parse never enters c.pages, so every gate
				// that walks the page list would silently skip it and, finding
				// nothing, print `ok` — the report telling a reader a gate held
				// when it never examined the page. Deny each page-walking gate that
				// pass, so a genuine finding on an unparsed page cannot disappear.
				for _, name := range []string{CheckHero, CheckBannedTokens, CheckSnippets, CheckMobile, CheckFigureLabels} {
					c.fail(name, rel, "", "this page did not parse, so this gate could not examine it")
				}
				continue
			}
			return err
		}
		c.pages[rel] = doc
		c.res.Pages = append(c.res.Pages, rel)
		if isComposedSurface(rel) {
			c.res.Composed = append(c.res.Composed, rel)
		}
	}
	return nil
}

// isComposedSurface reports whether a page is held to the composed-surface
// rules: everything the build writes except the verbatim record rendering and
// the documentation tree the SSG owns (adr-47 decision 3).
func isComposedSurface(rel string) bool {
	return !strings.HasPrefix(rel, recordRoutePrefix) && !strings.HasPrefix(rel, docsRoutePrefix)
}

// composedPages yields the composed surfaces in a fixed order.
func (c *checker) composedPages() []string { return c.res.Composed }

// source reads a repository file a data-src names, once.
//
// The path arrives from the rendered page, which is generated from the manifest
// — and a check that reads whatever a rendered attribute names is a check that
// can be pointed at the rest of the disk by editing one committed file. So the
// path is held to a repo-relative shape before anything opens it, the same way
// the build holds a manifest path.
func (c *checker) source(rel string) *sourceFile {
	if s, ok := c.sources[rel]; ok {
		return s
	}
	s := &sourceFile{}
	if !fsutil.ValidRelPath(rel) {
		s.Missing = true
		c.sources[rel] = s
		return s
	}
	data, err := fsutil.ReadGuardedInRoot(c.root, rel, maxPageBytes)
	if err != nil {
		s.Missing = true
	} else {
		s.Lines = strings.Split(string(data), "\n")
		body, consumed := StripFrontmatter(string(data))
		if secs, serr := Sections(rel, body, consumed); serr == nil {
			s.Sections = secs
		}
	}
	c.sources[rel] = s
	return s
}

// spanLines returns the source lines a `path#anchor` span was selected from —
// the section's heading and body where an anchor is named, the whole file
// otherwise. They are what carries a ban's line-level escape across the lift
// into composed text.
func (c *checker) spanLines(src string) []string {
	rel, anchor, _ := strings.Cut(src, "#")
	s := c.source(rel)
	if s.Missing {
		return nil
	}
	if anchor == "" {
		return s.Lines
	}
	for _, sec := range s.Sections {
		if sec.Anchor != anchor {
			continue
		}
		lines := strings.Split(sec.Body, "\n")
		if sec.Line > 0 && sec.Line <= len(s.Lines) {
			lines = append([]string{s.Lines[sec.Line-1]}, lines...)
		}
		return lines
	}
	return s.Lines
}

// resolveSource reports why a data-src cannot be resolved, or the empty string.
func (c *checker) resolveSource(src string) string {
	rel, anchor, _ := strings.Cut(src, "#")
	if rel == "" || !fsutil.ValidRelPath(rel) {
		return "names " + quote(src) + ", which is not a repo-relative path"
	}
	s := c.source(rel)
	if s.Missing {
		return "names " + quote(rel) + ", which the repository does not carry"
	}
	if anchor == "" {
		return ""
	}
	for _, sec := range s.Sections {
		if sec.Anchor == anchor {
			return ""
		}
	}
	return "names the heading anchor " + quote("#"+anchor) + " in " + rel + ", which that file has no heading for"
}

// ---------------------------------------------------------------------------
// 1. Provenance
// ---------------------------------------------------------------------------

// textUse is one visible text node and the span it sits in.
type textUse struct {
	Page string
	Src  string
	Text string
	Node *htmlNode
}

// visibleText collects the page's visible text nodes with the nearest data-src
// span each sits in, and reports the head's Identity-carrying elements
// separately: `<title>` and `<meta name="description">` hold Identity text with
// no attribute to name it, and are checked against the block instead.
func visibleText(page string, doc *htmlNode) (uses []textUse, title string, hasTitle bool, desc string, hasDesc bool) {
	doc.Walk(func(n *htmlNode) {
		if n.Kind == htmlElementNode {
			if n.Name == "meta" && strings.EqualFold(n.Attr("name"), "description") {
				desc, hasDesc = n.Attr("content"), true
			}
			return
		}
		if strings.TrimSpace(n.Text) == "" {
			return
		}
		if n.InsideElement("script", "style") {
			return
		}
		// The document title, not an SVG's accessible name: the drawing's own
		// <title> sits inside a figure that names its source like any other
		// composed block.
		if n.Parent != nil && n.Parent.Name == "title" && !n.Parent.InsideElement("svg") {
			title, hasTitle = n.Text, true
			return
		}
		src := ""
		for _, a := range n.Ancestors() {
			if v := a.Attr("data-src"); v != "" {
				src = v
				break
			}
		}
		uses = append(uses, textUse{Page: page, Src: src, Text: n.Text, Node: n})
	})
	return uses, title, hasTitle, desc, hasDesc
}

// checkProvenance walks every composed surface and refuses a visible word it
// cannot trace.
func (c *checker) checkProvenance() {
	base := c.generatorWords()
	for _, page := range c.composedPages() {
		allow := base
		if page == attributionPageRel {
			allow.attribution = c.attributionWords()
		}
		uses, title, hasTitle, desc, hasDesc := visibleText(page, c.pages[page])
		seenSrc := map[string]bool{}
		for _, u := range uses {
			if u.Src != "" {
				if !seenSrc[u.Src] {
					seenSrc[u.Src] = true
					if why := c.resolveSource(u.Src); why != "" {
						c.fail(CheckProvenance, page, u.Src, "a data-src %s", why)
					}
				}
				continue
			}
			if allow.covers(u.Text) {
				continue
			}
			c.fail(CheckProvenance, page, "", "%s carries the text %s, which sits in no data-src element "+
				"and is not an interface string, a number, a date, a file name or an asset name",
				elementPath(u.Node), quote(clip(strings.TrimSpace(u.Text))))
		}
		// The two documented exceptions are VERIFIED, not skipped. Where the
		// repository records no Identity block there is nothing to verify
		// against, and they fall back to the generator's own vocabulary rather
		// than going unread — an exception that stops checking when its
		// reference is absent is an exception with a hole in it.
		switch {
		case c.hasBlock:
			if hasTitle {
				if why := c.titleWhy(title, allow, sourcedTexts(uses)); why != "" {
					c.fail(CheckProvenance, page, c.identitySpan(), "<title> reads %s: %s", quote(clip(title)), why)
				}
			}
			if hasDesc && desc != c.identity.Tagline {
				c.fail(CheckProvenance, page, c.identitySpan(),
					`<meta name="description"> reads %s; the Identity block's tagline is %s`, quote(desc), quote(c.identity.Tagline))
			}
		default:
			if hasTitle && !allow.covers(title) {
				c.fail(CheckProvenance, page, "",
					"<title> reads %s, and this repository records no Identity block to check it against", quote(clip(title)))
			}
			if hasDesc && !allow.covers(desc) {
				c.fail(CheckProvenance, page, "",
					`<meta name="description"> reads %s, and this repository records no Identity block to check it against`, quote(clip(desc)))
			}
		}
	}
}

// titleSeparator joins a page's own name to the project's in a `<title>`. It is
// the composer's own spelling (compose.go's `headWith`), not a guess.
const titleSeparator = " · "

// titleWhy says why a page's `<title>` is not accounted for, or nothing.
//
// The landing page's title IS the Identity title. Every page that names itself
// reads `<page> · <project>`, and both halves have to hold up: the tail is the
// Identity block, verified; the head is either a word the generator may add (a
// navigation label from ui.json) or text the page itself renders from a source
// it names. The second arm is what lets a record page be titled by its own
// heading without that heading becoming a sentence written for the website.
func (c *checker) titleWhy(title string, allow generatorWords, sourced map[string]bool) string {
	if title == c.identity.Title {
		return ""
	}
	suffix := titleSeparator + c.identity.Title
	if !strings.HasSuffix(title, suffix) {
		return "the Identity block's title is " + quote(c.identity.Title) +
			", and this is neither that nor a page name followed by it"
	}
	head := strings.TrimSpace(strings.TrimSuffix(title, suffix))
	switch {
	case head == "":
		return "it names no page before " + quote(titleSeparator)
	case allow.covers(head), sourced[collapseSpaces(head)]:
		return ""
	}
	return "the page name " + quote(clip(head)) +
		" is not an interface string, and nothing this page renders from a named source says it"
}

// sourcedTexts is every string this page renders from a span it names — each
// text node on its own, and each named block's text entire, because a heading
// carrying inline markup arrives as several nodes and one element.
func sourcedTexts(uses []textUse) map[string]bool {
	out := map[string]bool{}
	seen := map[*htmlNode]bool{}
	for _, u := range uses {
		if u.Src == "" {
			continue
		}
		out[collapseSpaces(u.Text)] = true
		for _, a := range u.Node.Ancestors() {
			if a.Attr("data-src") == "" || seen[a] {
				continue
			}
			seen[a] = true
			out[collapseSpaces(a.TextContent())] = true
		}
	}
	delete(out, "")
	return out
}

// identitySpan is the data-src the Identity block's text carries elsewhere on
// the page, so a title finding points at the same place a hero finding does.
func (c *checker) identitySpan() string {
	return c.manifest.Identity.File + "#" + Slug(c.manifest.Identity.Heading)
}

// elementPath renders a text node's position as its ancestor chain, so a
// finding says WHERE on the page the untraceable words are.
func elementPath(n *htmlNode) string {
	var parts []string
	for _, a := range n.Ancestors() {
		if a.Name == "#document" {
			break
		}
		label := a.Name
		if cls := a.Attr("class"); cls != "" {
			label += "." + strings.Join(strings.Fields(cls), ".")
		}
		parts = append([]string{label}, parts...)
	}
	if len(parts) > 3 {
		parts = parts[len(parts)-3:]
	}
	return strings.Join(parts, " > ")
}

// ---------------------------------------------------------------------------
// The generator's own vocabulary
// ---------------------------------------------------------------------------

// generatorWords is the closed set of words the single-source rule lets the
// generator add, together with the repository facts the documented exceptions
// are matched against.
type generatorWords struct {
	// phrases are matched whole, case-folded: interface strings, their
	// separator-delimited segments, package metadata and chapter letters.
	phrases map[string]bool
	// names are the single tokens a field may be: file names, asset names and
	// the package's own metadata.
	names map[string]bool
	// attribution is the authorship vocabulary the attribution surface prints —
	// contributor names and declared `Assisted-by:` values, read out of git.
	// It is empty everywhere else, which is what keeps the escape scoped to the
	// one page adr-47 decision 3 grants it to.
	attribution map[string]bool
	// exists reports whether a token names a file the repository carries.
	exists func(string) bool
}

// decorationRunes are the characters the layouts use to join two facts. They
// carry no words, so a text node made only of them is nothing to trace, and a
// node containing them is read as the facts they separate.
// A slash is deliberately absent: it joins the two halves of a forge handle
// (`owner/name`), which is one name rather than two facts.
const decorationRunes = "·©↗↓←→—–•|"

// generatorWords gathers the vocabulary once per run.
func (c *checker) generatorWords() generatorWords {
	g := generatorWords{phrases: map[string]bool{}, names: map[string]bool{}}
	addPhrase := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		g.phrases[strings.ToLower(s)] = true
	}
	addName := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		g.names[strings.ToLower(s)] = true
		addPhrase(s)
	}

	// Every interface string, and every segment of one: a layout that shows the
	// half of "macOS · Intel" that differs from the row above it is still
	// showing a word ui.json declares.
	for _, s := range uiStrings(c.ui) {
		addPhrase(s)
		for _, seg := range strings.Split(s, "·") {
			addPhrase(seg)
		}
	}
	// The package's own metadata, from `.claude-plugin/plugin.json`.
	addName(c.repo.Name)
	addName(c.repo.AuthorName)
	addName(c.repo.License)
	if c.repo.Repository != "" {
		addName(repoHandle(c.repo.Repository))
	}
	// The brand is the first word of the canonical title.
	if c.hasBlock {
		if f := strings.Fields(c.identity.Title); len(f) > 0 {
			addName(f[0])
		}
	}
	// A chapter's letter, from `.abcd/site.json`.
	for _, ch := range c.manifest.Home.Chapters {
		addName(ch.Letter)
	}
	// The released asset names the page links.
	for _, a := range LinkedBinaryAssets {
		addName(a)
	}
	addName(AssetChecksums)

	g.exists = func(token string) bool {
		if !fsutil.ValidRelPath(token) {
			return false
		}
		_, err := c.root.Stat(token)
		return err == nil
	}
	return g
}

// attributionWords is the vocabulary the attribution surface prints: every
// contributor's name and every declared `Assisted-by:` value, read out of git
// by the same pass the page renders from.
//
// This is adr-47 decision 3's carve-out — "the contributors page prints model
// names under the declared attribution escape" — implemented as a VERIFICATION
// rather than an exemption. A model name on that page is not waved through for
// being on that page; it is matched against the trailers the repository's own
// history actually carries, so a name no commit declared still fails. The
// convention that makes this the sanctioned place to name a tool is the same one
// `ACKNOWLEDGEMENTS.md` rests on: naming is confined to credit.
func (c *checker) attributionWords() map[string]bool {
	out := map[string]bool{}
	auth, err := LoadAuthorship(c.repoRoot)
	if err != nil {
		return out
	}
	add := func(s string) {
		if s = strings.TrimSpace(s); s != "" {
			out[strings.ToLower(s)] = true
		}
	}
	for _, a := range auth.Humans {
		add(a.Name)
	}
	for _, a := range auth.Bots {
		add(a.Name)
	}
	for _, m := range auth.ByModel {
		add(m.Model)
		// A trailer reads `Vendor:model`; the page also prints the vendor alone
		// as the row that groups them.
		if vendor, _, ok := strings.Cut(m.Model, ":"); ok {
			add(vendor)
		}
	}
	add(noneDeclaration)
	return out
}

// underAttributionEscape reports whether a composed span is credit rather than
// prose, and so is where the conventions confine naming a tool.
//
// Two things are: text selected from the repository's `ACKNOWLEDGEMENTS.md`,
// whose entire purpose is to say which tools and sources an idea came from; and
// the attribution page's own authorship data, which is git's record of what
// assisted rather than anything anybody wrote. Everywhere else the bans apply,
// which is what keeps this an escape rather than a hole — and the provenance
// walk still holds every word on that page to the authorship it came from, so
// the escape covers what may be NAMED, never what may be said.
func underAttributionEscape(page, src string) bool {
	file, _, _ := strings.Cut(src, "#")
	if file == AcknowledgementsRelPath {
		return true
	}
	return page == attributionPageRel && src == ""
}

// uiStrings lists every interface string the allowlist declares.
//
// It walks the struct rather than naming its fields, for the same reason
// `UI.missing` does: a hand-written list beside a growing struct goes stale
// silently, and the way it goes stale is the worst one available here — a word
// the generator is entitled to add stops being recognised, and the provenance
// walk starts refusing text that is perfectly well allowed. Reflection cannot
// forget a field.
func uiStrings(ui UI) []string {
	var out []string
	var walk func(v reflect.Value, t reflect.Type)
	walk = func(v reflect.Value, t reflect.Type) {
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			switch f.Type.Kind() {
			case reflect.Struct:
				walk(v.Field(i), f.Type)
			case reflect.Map:
				// A keyed group of interface strings — the forge names. Only the
				// VALUES are rendered; a key is a lookup term (a host), never a
				// word on a page. Reflection has to walk maps as well as fields,
				// or a whole family of declared words goes unrecognised and the
				// walk refuses text the allowlist plainly permits.
				iter := v.Field(i).MapRange()
				for iter.Next() {
					if iter.Value().Kind() == reflect.String {
						out = append(out, iter.Value().String())
					}
				}
			case reflect.String:
				// `_purpose` documents the file for its readers and is never
				// rendered, so it is not a word the generator may add.
				if strings.Split(f.Tag.Get("json"), ",")[0] != "_purpose" {
					out = append(out, v.Field(i).String())
				}
			}
		}
	}
	walk(reflect.ValueOf(ui), reflect.TypeOf(ui))
	return out
}

var (
	// A number, optionally a version, optionally a proportion. The dashboards
	// print `70%` beside `976 / 1377`, and both are counts rather than words.
	numberRe = regexp.MustCompile(`^v?\d+(\.\d+)*%?$`)
	dateRe   = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	commitRe = regexp.MustCompile(`^[0-9a-f]{7,40}$`)
)

// covers reports whether a text node is entirely accounted for by what the
// generator is allowed to add.
//
// The node is tried whole first, so a multi-word interface string matches as
// the string it is. Failing that it is split on the decorations the layouts
// join facts with, and every piece must be an interface string or resolve, word
// by word, to a number, a date, a commit, a file name, an asset name or the
// package's own metadata. Splitting stops at decorations deliberately: a
// sentence is not rescued by having each of its words appear somewhere in
// ui.json.
func (g generatorWords) covers(text string) bool {
	t := strings.TrimSpace(text)
	if t == "" || !hasWordCharacter(t) {
		return true
	}
	if g.phrases[strings.ToLower(collapseSpaces(t))] || g.attribution[strings.ToLower(collapseSpaces(t))] {
		return true
	}
	for _, piece := range strings.FieldsFunc(t, func(r rune) bool {
		return strings.ContainsRune(decorationRunes, r)
	}) {
		piece = strings.TrimSpace(piece)
		if piece == "" || !hasWordCharacter(piece) {
			continue
		}
		if g.phrases[strings.ToLower(collapseSpaces(piece))] || g.attribution[strings.ToLower(collapseSpaces(piece))] {
			continue
		}
		for _, field := range strings.Fields(piece) {
			if !g.token(field) {
				return false
			}
		}
	}
	return true
}

// token reports whether one whitespace-delimited word is something the
// generator may add on its own.
func (g generatorWords) token(field string) bool {
	// Trailing punctuation is the sentence's; a LEADING dot is the name's.
	// Trimming both ends turns `.abcd/site.json` into a path that resolves
	// nowhere, and the file name the page legitimately printed is then reported
	// as untraceable text.
	f := strings.TrimRight(strings.TrimLeft(field, "([{"), ".,;:!?)]}")
	if f == "" || !hasWordCharacter(f) {
		// A separator that survived the split — the `/` between two counts — is
		// punctuation, and punctuation is not a word anybody has to source.
		return true
	}
	switch {
	case numberRe.MatchString(f), dateRe.MatchString(f), commitRe.MatchString(f):
		return true
	case g.names[strings.ToLower(f)], g.attribution[strings.ToLower(f)]:
		return true
	case strings.Contains(f, ".") && g.exists(f):
		return true
	}
	return false
}

// hasWordCharacter reports whether a text node carries any word at all, so a
// run of pure decoration is nothing to trace.
//
// It asks Unicode rather than ASCII. An ASCII test would declare every node of
// a site written in Cyrillic, Greek or Han "no words here" and wave it through
// unread — which turns the walk's claim, that every visible word is sourced,
// into a claim about Latin script only.
func hasWordCharacter(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

func collapseSpaces(s string) string { return strings.Join(strings.Fields(s), " ") }

// ---------------------------------------------------------------------------
// 2. Hero-vs-Identity
// ---------------------------------------------------------------------------

// heroSpans are the three hero paragraphs and the Identity field each renders.
var heroSpans = []struct {
	class string
	field func(positioning.Block) string
	name  string
}{
	{"eyebrow", func(b positioning.Block) string { return b.Title }, "title"},
	{"tagline", func(b positioning.Block) string { return b.Tagline }, "tagline"},
	{"pitch", func(b positioning.Block) string { return b.Pitch }, "pitch"},
}

// checkHero holds the rendered hero to the Identity block, read through
// `positioning.ParseBlock` — the canonical parser, so the check cannot disagree
// with the surfaces the same block already re-renders (itd-135 AC 1).
func (c *checker) checkHero() {
	if !c.hasBlock {
		// A repository that records no Identity block renders a hero without one
		// (graceful absence, build.go); there is nothing to compare against.
		c.note(CheckHero, c.manifest.Identity.File,
			"no Identity block at %q; the hero renders without an eyebrow, tagline and pitch", c.manifest.Identity.Heading)
		return
	}
	for _, page := range c.composedPages() {
		hero := findElement(c.pages[page], func(n *htmlNode) bool {
			return n.Name == "section" && n.HasClass("hero")
		})
		if hero == nil {
			// An absent hero is the LARGEST drift this gate exists to catch, so
			// it is a finding on the page the manifest composes a hero onto, and
			// a skip everywhere else. Reading "no hero here" as "nothing to
			// compare" is how a gate comes to pass a page it should refuse.
			if page == homePageRel && c.manifest.Home.Hero.Page != "" {
				c.fail(CheckHero, page, c.identitySpan(),
					"the page renders no .hero section, and the manifest composes one from %s", c.manifest.Home.Hero.Page)
			}
			continue
		}
		for _, want := range heroSpans {
			text := want.field(c.identity)
			el := findElement(hero, func(n *htmlNode) bool { return n.Name == "p" && n.HasClass(want.class) })
			switch {
			case el == nil && text == "":
				// The block declares none; the hero renders none.
			case el == nil:
				c.fail(CheckHero, page, c.identitySpan(),
					"the hero renders no .%s, but the Identity block declares a %s: %s", want.class, want.name, quote(clip(text)))
			case collapseSpaces(el.TextContent()) != collapseSpaces(text):
				c.fail(CheckHero, page, c.identitySpan(),
					"the hero's .%s reads %s; the Identity block's %s is %s",
					want.class, quote(clip(el.TextContent())), want.name, quote(clip(text)))
			}
		}
	}
}

// findElement returns the first element in document order matching pred.
func findElement(root *htmlNode, pred func(*htmlNode) bool) *htmlNode {
	var found *htmlNode
	root.Walk(func(n *htmlNode) {
		if found != nil || n.Kind != htmlElementNode {
			return
		}
		if pred(n) {
			found = n
		}
	})
	return found
}

// findElements returns every element matching pred, in document order.
func findElements(root *htmlNode, pred func(*htmlNode) bool) []*htmlNode {
	var out []*htmlNode
	root.Walk(func(n *htmlNode) {
		if n.Kind == htmlElementNode && pred(n) {
			out = append(out, n)
		}
	})
	return out
}

// ---------------------------------------------------------------------------
// 3. Banned tokens over composed text
// ---------------------------------------------------------------------------

// checkBannedTokens runs the docs-lint bans over the words every composed
// surface publishes, span by span, with each span's own source lines in hand so
// the line-level escape survives the lift.
func (c *checker) checkBannedTokens() {
	if c.tokens.Len() == 0 {
		return
	}
	for _, page := range c.composedPages() {
		uses, title, hasTitle, desc, hasDesc := visibleText(page, c.pages[page])
		// One span, one body of text: the bans are line-level, and a node is the
		// smallest thing the page renders as a line of its own. A span is split
		// again by whether the text was rendered as CODE, because that is what a
		// ban's `skip_code_fences` asks about — and the markup is the only place
		// that fact still exists once the fence markers are gone.
		type spanKey struct {
			src    string
			fenced bool
		}
		type span struct {
			key   spanKey
			lines []string
		}
		var spans []span
		index := map[spanKey]int{}
		add := func(key spanKey, text string) {
			t := collapseSpaces(text)
			if t == "" {
				return
			}
			i, ok := index[key]
			if !ok {
				i = len(spans)
				index[key] = i
				spans = append(spans, span{key: key})
			}
			spans[i].lines = append(spans[i].lines, t)
		}
		for _, u := range uses {
			// A code BLOCK, not an inline code span. The file walk masks fenced
			// blocks and nothing else, so treating `<code>` on its own as a fence
			// would exempt a banned token merely for being set in monospace —
			// which is a way to smuggle one past a gate that the source file
			// itself does not offer.
			add(spanKey{u.Src, u.Node.InsideElement("pre")}, u.Text)
		}
		// The head's Identity text is composed text too: it is published, and
		// adr-47 decision 3 scopes the ban to what the site publishes.
		if hasTitle {
			add(spanKey{src: c.identitySpan()}, title)
		}
		if hasDesc {
			add(spanKey{src: c.identitySpan()}, desc)
		}
		for _, s := range spans {
			if underAttributionEscape(page, s.key.src) {
				continue
			}
			name := page
			if s.key.src != "" {
				name = page + " (" + s.key.src + ")"
			}
			text := strings.Join(s.lines, "\n")
			for _, f := range c.tokens.LintComposed(name, text, s.key.fenced, c.spanLines(s.key.src)) {
				line := ""
				if f.Line >= 1 && f.Line <= len(s.lines) {
					line = s.lines[f.Line-1]
				}
				c.fail(CheckBannedTokens, page, s.key.src, "%s: %s — in %s", f.RuleID, f.Message, quote(clip(line)))
			}
		}
	}
}

// ---------------------------------------------------------------------------
// 4. Snippet pinning
// ---------------------------------------------------------------------------

// checkSnippets holds every `abcd …` command the site shows to the generated
// CLI reference (itd-135 AC 4).
//
// The reference is GENERATED from the command tree, and it spells a command as
// its usage line — `abcd intent [text] [flags]` — while a page shows the command
// a reader would type: `abcd intent "<one-line idea>"`. Comparing those as
// bytes is a check nothing can ever satisfy, and a check nothing can satisfy is
// a check somebody turns off. So what is pinned is the part the reference
// actually states: the command PATH the snippet invokes must be a command the
// reference documents, and every flag it passes must be a flag documented for
// that command or for one of its parents. A renamed verb, a removed sub-verb
// and a dropped flag all fail; a placeholder argument does not.
func (c *checker) checkSnippets() {
	ref := c.manifest.Docs.CLI
	if ref == "" {
		// Nothing to pin against is a state, not a pass. Printing `ok snippets`
		// for a run that never compared anything is the report telling a reader
		// a gate held when it did not run.
		c.note(CheckSnippets, ManifestRelPath,
			"docs.cli names no CLI reference, so no `abcd …` snippet on the site is pinned to anything")
		return
	}
	data, err := fsutil.ReadGuardedInRoot(c.root, ref, maxPageBytes)
	if err != nil {
		c.fail(CheckSnippets, ref, "", "the manifest names %s as the CLI reference, and the repository does not carry it", quote(ref))
		return
	}
	doc := parseCLIReference(string(data))
	if len(doc.commands) == 0 {
		c.fail(CheckSnippets, ref, "", "no `abcd …` commands are documented there; every snippet on the site would be unpinnable")
		return
	}
	for _, page := range c.composedPages() {
		// Fenced blocks only. An inline code span is prose punctuation — "the
		// `abcd capture` verb" — and reading its words as a command line would
		// fail a sentence for being a sentence.
		for _, code := range findElements(c.pages[page], func(n *htmlNode) bool {
			return n.Name == "code" && n.Parent != nil && n.Parent.Name == "pre"
		}) {
			src := ""
			for _, a := range code.Ancestors() {
				if v := a.Attr("data-src"); v != "" {
					src = v
					break
				}
			}
			for _, line := range strings.Split(code.TextContent(), "\n") {
				cmd := strings.TrimSpace(line)
				if cmd == "" || !strings.HasPrefix(cmd, "abcd ") {
					continue
				}
				if why := doc.pin(cmd); why != "" {
					c.fail(CheckSnippets, page, src, "the snippet %s %s (%s)", quote(cmd), why, ref)
				}
			}
		}
	}
}

// cliReference is the generated CLI reference, read as the set of commands it
// documents and the flags each declares.
type cliReference struct {
	commands map[string]bool
	flags    map[string]map[string]bool
}

var (
	cliHeadingRe = regexp.MustCompile("^#{2,6}\\s+`(abcd[^`]*)`\\s*$")
	cliUsageRe   = regexp.MustCompile("\\*\\*Usage:\\*\\*\\s+`(abcd[^`]*)`")
	cliFlagRe    = regexp.MustCompile(`--[A-Za-z][A-Za-z0-9-]*`)
)

// parseCLIReference reads the generated page. Each heading names a command; the
// fenced block under its `**Flags:**` line declares that command's flags.
//
// Only FENCED lines contribute flags. Prose in a section can name a flag while
// saying it is gone, and crediting that would let the very drift this check
// exists to catch pass.
func parseCLIReference(md string) cliReference {
	ref := cliReference{commands: map[string]bool{}, flags: map[string]map[string]bool{}}
	cur := ""
	fenced := false
	for _, line := range strings.Split(md, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			fenced = !fenced
			continue
		}
		if !fenced {
			if m := cliHeadingRe.FindStringSubmatch(strings.TrimRight(line, " \t")); m != nil {
				cur = commandPathOf(m[1])
				ref.commands[cur] = true
				if ref.flags[cur] == nil {
					ref.flags[cur] = map[string]bool{}
				}
				continue
			}
			if m := cliUsageRe.FindStringSubmatch(line); m != nil {
				ref.commands[commandPathOf(m[1])] = true
			}
			continue
		}
		if cur == "" {
			continue
		}
		for _, f := range cliFlagRe.FindAllString(line, -1) {
			ref.flags[cur][f] = true
		}
	}
	return ref
}

// commandWords reduces a command line to the leading words that could be a
// command path: everything up to the first flag, placeholder or quoted
// argument.
func commandWords(usage string) []string {
	var parts []string
	for _, w := range strings.Fields(usage) {
		if strings.HasPrefix(w, "-") || strings.HasPrefix(w, "<") || strings.HasPrefix(w, "[") || strings.HasPrefix(w, `"`) {
			break
		}
		parts = append(parts, w)
	}
	return parts
}

// commandPathOf reduces a usage line to the command path it invokes.
func commandPathOf(usage string) string { return strings.Join(commandWords(usage), " ") }

// pin says why a snippet does not match the reference, or nothing.
//
// The path is walked DOWN, not guessed: each word extends the path while the
// longer form is a documented command, and the first word that does not is an
// argument. The one word that may not be an argument is the sub-verb itself —
// `abcd bogus` is a snippet naming a command that does not exist, and reading
// `bogus` as an argument to bare `abcd` is exactly how that would slip through.
func (r cliReference) pin(cmd string) string {
	words := commandWords(cmd)
	if len(words) == 0 {
		return ""
	}
	path := words[0]
	if !r.commands[path] {
		return "names " + quote(path) + ", which the CLI reference does not document"
	}
	for _, w := range words[1:] {
		next := path + " " + w
		if r.commands[next] {
			path = next
			continue
		}
		if path == words[0] {
			return "names " + quote(next) + ", which the CLI reference does not document"
		}
		break
	}
	for _, f := range cliFlagRe.FindAllString(cmd, -1) {
		if r.flagKnown(path, f) {
			continue
		}
		return "passes " + quote(f) + ", which the CLI reference does not document for " + quote(path)
	}
	return ""
}

// flagKnown reports whether a flag is documented for a command or inherited
// from one of its parents, which is how a persistent flag reaches a sub-verb.
func (r cliReference) flagKnown(path, flag string) bool {
	for p := path; p != ""; {
		if r.flags[p][flag] {
			return true
		}
		i := strings.LastIndex(p, " ")
		if i < 0 {
			break
		}
		p = p[:i]
	}
	return false
}

// ---------------------------------------------------------------------------
// 5. The baseline ratchet
// ---------------------------------------------------------------------------

// checkBaseline holds the record's unresolved references to the committed
// ratchet. The direction is the whole point: a reference outside the baseline
// FAILS, and a baseline entry whose reference has been fixed is reported as
// shrinkable without failing — growing is refused, shrinking is invited.
func (c *checker) checkBaseline() error {
	data, err := fsutil.ReadGuarded(filepath.Join(c.outDir, "record.json"), maxOutputPageBytes)
	if err != nil {
		if os.IsNotExist(err) {
			c.fail(CheckBaseline, "record.json", "", "the output directory carries no record export; the ratchet has nothing to measure")
			return nil
		}
		return err
	}
	var export RecordExport
	if err := json.Unmarshal(data, &export); err != nil {
		return fmt.Errorf("site: record.json: %w", err)
	}
	baseline, ok, err := LoadBaseline(c.repoRoot, c.manifest.Checks.UnresolvedReferenceBaseline)
	if err != nil {
		return err
	}
	where := BaselineRelPath
	if b := c.manifest.Checks.UnresolvedReferenceBaseline; b != "" {
		where = b
	}
	admitted := map[[2]string]bool{}
	for _, e := range baseline.UnresolvedReferences {
		admitted[[2]string{e.From, e.To}] = true
	}
	live := map[[2]string]bool{}
	for _, e := range export.Health.Unresolved {
		key := [2]string{e.From, e.To}
		live[key] = true
		if admitted[key] {
			continue
		}
		if !ok {
			c.fail(CheckBaseline, where, "",
				"%s → %s (%s) is unresolved and this repository admits no baseline", e.From, e.To, e.Rel)
			continue
		}
		c.fail(CheckBaseline, where, "",
			"%s → %s (%s) is unresolved and outside the committed baseline; fix the reference, or admit it deliberately",
			e.From, e.To, e.Rel)
	}
	if !ok {
		return nil
	}
	// The other direction is an INVITATION, not a failure: a baseline entry
	// whose reference now resolves is the ratchet's own good news, and failing a
	// build over it would teach people to stop fixing references.
	for _, e := range baseline.UnresolvedReferences {
		if live[[2]string{e.From, e.To}] {
			continue
		}
		c.note(CheckBaseline, where,
			"%s → %s resolves now; the baseline can shrink by one entry", e.From, e.To)
	}
	return nil
}

// ---------------------------------------------------------------------------
// 6. Static mobile checks
// ---------------------------------------------------------------------------

// maxInlineWidthPx is the narrow viewport itd-135 AC 7 names. An element whose
// own inline style pins it wider than this cannot shrink to fit a phone,
// whatever the stylesheet says.
const maxInlineWidthPx = 390

// checkMobile runs the static half of AC 7 over every page THIS BUILD writes,
// the record rendering included: a table that scrolls off a phone does so
// whatever tree its words came from. `docs/` is not among them — it is the
// SSG's own output and is excluded from the walk entirely, so no gate here
// examines it. The rendered-overflow audit needs a browser and is CI's; it
// draws its routes from this build's own pages and excludes `docs/` too.
func (c *checker) checkMobile() {
	for _, page := range c.res.Pages {
		doc := c.pages[page]
		c.checkViewport(page, doc)
		sheets := c.stylesheetsFor(page, doc)
		c.checkOverflowContainers(page, doc, sheets)
		c.checkImageConstraint(page, doc, sheets)
		c.checkInlineWidths(page, doc)
	}
}

// checkViewport asserts the one meta tag without which nothing else about a
// narrow viewport is true: a page without it is rendered at desktop width and
// scaled down.
func (c *checker) checkViewport(page string, doc *htmlNode) {
	meta := findElement(doc, func(n *htmlNode) bool {
		return n.Name == "meta" && strings.EqualFold(n.Attr("name"), "viewport")
	})
	switch {
	case meta == nil:
		c.fail(CheckMobile, page, "", `no <meta name="viewport">; the page is laid out at desktop width and scaled down on a phone`)
	case !strings.Contains(meta.Attr("content"), "width=device-width"):
		c.fail(CheckMobile, page, "", `<meta name="viewport"> content is %s; it must set width=device-width`, quote(meta.Attr("content")))
	}
}

// stylesheetsFor reads the stylesheets a page links from inside the output
// tree. A stylesheet the page links and the output does not carry is a finding:
// every rule the mobile checks rely on would be missing from the served page.
func (c *checker) stylesheetsFor(page string, doc *htmlNode) []*styleSheet {
	var out []*styleSheet
	for _, link := range findElements(doc, func(n *htmlNode) bool {
		return n.Name == "link" && strings.EqualFold(n.Attr("rel"), "stylesheet")
	}) {
		href := link.Attr("href")
		if href == "" || strings.Contains(href, "://") || strings.HasPrefix(href, "//") {
			continue // a remote sheet; this check reads what the build wrote
		}
		// A root-absolute href resolves against the SERVED ROOT, which is the
		// output directory — not against the page's own directory. The shared
		// stylesheet is linked that way precisely because the pages that link it
		// sit at every depth the site has, so resolving it per-page would look
		// for one stylesheet in as many places as there are routes and find it
		// in none of them.
		rel := ""
		if strings.HasPrefix(href, "/") {
			rel = path.Clean(strings.TrimPrefix(href, "/"))
		} else {
			rel = path.Clean(path.Join(path.Dir(page), href))
		}
		if !fsutil.ValidRelPath(rel) {
			c.fail(CheckMobile, page, "", "links the stylesheet %s, which resolves outside the output directory", quote(href))
			continue
		}
		if s, ok := c.styles[rel]; ok {
			out = append(out, s)
			continue
		}
		data, err := fsutil.ReadGuarded(filepath.Join(c.outDir, filepath.FromSlash(rel)), maxStylesheetBytes)
		if err != nil {
			c.fail(CheckMobile, page, "", "links the stylesheet %s, which the output directory does not carry", quote(rel))
			continue
		}
		s := parseStylesheet(string(data))
		c.styles[rel] = s
		out = append(out, s)
	}
	return out
}

// overflowElements and overflowClasses report which selectors the linked
// stylesheets give an overflow rule to.
func overflowFrom(sheets []*styleSheet) (elements, classes map[string]bool) {
	elements, classes = map[string]bool{}, map[string]bool{}
	for _, s := range sheets {
		for e := range s.OverflowElements {
			elements[e] = true
		}
		for cl := range s.OverflowClasses {
			classes[cl] = true
		}
	}
	return elements, classes
}

// wideElements are the two things the corpus renders that are wider than a
// phone: a pipe table and a fenced command block.
var wideElements = []string{"table", "pre"}

// checkOverflowContainers asserts every wide element can scroll inside itself
// or inside a container that scrolls. Either satisfies AC 7 — what must not
// happen is a wide element with no scroll anywhere above it, which pushes the
// whole page sideways.
func (c *checker) checkOverflowContainers(page string, doc *htmlNode, sheets []*styleSheet) {
	elements, classes := overflowFrom(sheets)
	for _, name := range wideElements {
		if elements[name] {
			continue
		}
		for _, el := range findElements(doc, func(n *htmlNode) bool { return n.Name == name }) {
			wrapped := false
			for _, a := range el.Ancestors() {
				for _, cl := range a.Classes() {
					if classes[cl] {
						wrapped = true
					}
				}
			}
			if wrapped {
				continue
			}
			c.fail(CheckMobile, page, el.Attr("data-src"),
				"a <%s> at %s sits in no element the stylesheet gives an overflow rule; on a narrow viewport it widens the page instead of scrolling",
				name, elementPath(el))
		}
	}
}

// checkImageConstraint asserts the stylesheet ships the rule that lets a
// picture shrink, and that no picture pins itself wider than the widest column
// the design offers — the width it would take if that rule were ever dropped.
func (c *checker) checkImageConstraint(page string, doc *htmlNode, sheets []*styleSheet) {
	imgs := findElements(doc, func(n *htmlNode) bool { return n.Name == "img" })
	if len(imgs) == 0 {
		return
	}
	constrained := false
	column := 0
	for _, s := range sheets {
		if s.ImageMaxWidth {
			constrained = true
		}
		if s.ContentColumnPx > column {
			column = s.ContentColumnPx
		}
	}
	if !constrained {
		c.fail(CheckMobile, page, "",
			"the page renders %d image(s) and no linked stylesheet declares a max-width for `img`; a picture wider than the viewport widens the page", len(imgs))
	}
	if column == 0 {
		return
	}
	for _, img := range imgs {
		w, err := strconv.Atoi(strings.TrimSpace(img.Attr("width")))
		if err != nil || w <= column {
			continue
		}
		c.fail(CheckMobile, page, img.Attr("data-src"),
			"the image %s declares width=%d, wider than the %dpx content column", quote(img.Attr("src")), w, column)
	}
}

// checkInlineWidths refuses an element that pins its own width wider than the
// narrow viewport. A stylesheet rule can be overridden by a media query; an
// inline style cannot, so this one is absolute.
func (c *checker) checkInlineWidths(page string, doc *htmlNode) {
	for _, el := range findElements(doc, func(n *htmlNode) bool { return n.Attr("style") != "" }) {
		for prop, px := range fixedWidthsIn(el.Attr("style")) {
			if px <= maxInlineWidthPx {
				continue
			}
			c.fail(CheckMobile, page, el.Attr("data-src"),
				"<%s> sets an inline %s of %dpx, wider than the %dpx viewport the mobile audit uses",
				el.Name, prop, px, maxInlineWidthPx)
		}
	}
}

var declRe = regexp.MustCompile(`(?i)(^|[;{\s])(min-width|width)\s*:\s*(\d+)px`)

// fixedWidthsIn reads the pixel widths a declaration block pins. A custom
// property (`--stack:22px`) is not a width, and the leading-boundary group is
// what keeps it from being read as one.
func fixedWidthsIn(decls string) map[string]int {
	out := map[string]int{}
	for _, m := range declRe.FindAllStringSubmatch(decls, -1) {
		n, err := strconv.Atoi(m[3])
		if err != nil {
			continue
		}
		prop := strings.ToLower(m[2])
		if n > out[prop] {
			out[prop] = n
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// 7. Loop-figure labels
// ---------------------------------------------------------------------------

// checkFigureLabels consumes the manifest's `figure.labels-from-page`: every
// text label in the figure a chapter lifts must be a phrase on the page it
// illustrates, so a diagram cannot drift from the prose beside it. Ported from
// the closing assertion in `compose.py`, inverted: the script checked a list of
// labels it knew about, and this checks every label the drawing actually
// carries.
func (c *checker) checkFigureLabels() {
	for _, ch := range c.manifest.Home.Chapters {
		if ch.Figure == nil || !ch.Figure.LabelsFromPage {
			continue
		}
		s := c.source(ch.Page)
		if s.Missing {
			c.fail(CheckFigureLabels, ch.Page, "", "the manifest composes chapter %q from a page the repository does not carry", ch.Letter)
			continue
		}
		page := strings.ToLower(collapseSpaces(strings.Join(s.Lines, " ")))
		found := 0
		for _, out := range c.composedPages() {
			for _, fig := range findElements(c.pages[out], func(n *htmlNode) bool {
				return n.Name == "figure" && strings.HasPrefix(n.Attr("data-src"), ch.Page)
			}) {
				if findElement(fig, func(n *htmlNode) bool { return n.Name == "svg" }) == nil {
					continue
				}
				found++
				for _, label := range figureLabels(fig) {
					if strings.Contains(page, label) {
						continue
					}
					c.fail(CheckFigureLabels, out, fig.Attr("data-src"),
						"the figure's label %s is not a phrase on %s; the manifest asks that every label be one", quote(label), ch.Page)
				}
			}
		}
		if found == 0 {
			c.fail(CheckFigureLabels, ch.Page, "",
				"the manifest asks that chapter %q's figure take its labels from the page, and the render carries no figure for it", ch.Letter)
		}
	}
}

// figureLabels reads a drawing's text labels, one per `<text>` element.
//
// A `<tspan>` is a LINE BREAK inside one label, not a label of its own — the
// drawings wrap "turns the intent / into engineering / work" across three of
// them — so the runs are joined with a space rather than concatenated, or the
// phrase the page carries would never match. Each label is then normalised the
// way the page it is compared against is: lower-cased, whitespace collapsed,
// and outer sentence punctuation dropped, so a label that ends a sentence still
// matches the sentence.
func figureLabels(fig *htmlNode) []string {
	var out []string
	seen := map[string]bool{}
	for _, t := range findElements(fig, func(n *htmlNode) bool { return n.Name == "text" }) {
		var runs []string
		t.Walk(func(n *htmlNode) {
			if n.Kind == htmlTextNode && strings.TrimSpace(n.Text) != "" {
				runs = append(runs, n.Text)
			}
		})
		label := strings.ToLower(collapseSpaces(strings.Join(runs, " ")))
		label = strings.Trim(label, " .,;:!?")
		if label == "" || seen[label] {
			continue
		}
		seen[label] = true
		out = append(out, label)
	}
	return out
}
