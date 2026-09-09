package site

// The landing page.
//
// Ported from `compose.py` in `.abcd/development/research/abcdev-site/` — the
// executable spec. The script carries LAYOUT ONLY, and so does this: every
// sentence on the page is a span of a repository file selected by path and
// heading through `.abcd/site.json`, every interface word comes from
// `site-src/ui.json`, and the composer's whole contribution is deciding which
// span becomes a card, which becomes a tab, and which becomes the figure.
//
// Every rendered block carries `data-src="path#heading"`. That attribute is the
// single-source rule made checkable: a later slice's provenance audit walks the
// generated HTML and refuses any visible text that is not inside one of these,
// so a block that grows here without a source is a build failure rather than a
// paragraph nobody can trace.

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/changelog"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/core/positioning"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// maxPageBytes bounds one composed page read.
const maxPageBytes = 1 << 20

// The released asset names the hero links directly, one per published platform.
//
// The manifest's `release.assets` says these names come from the release
// workflow, and that claim is EARNED rather than asserted: the build does not
// parse the workflow (a page that reads CI configuration to decide what to link
// is a worse dependency than a constant), so `installsurface_test.go` holds this
// list to the exact `abcd-<os>-<arch>` matrix the four committed install
// surfaces derive. A platform added to the release matrix without a link here —
// or a link here to a platform nothing publishes — fails that test.
const (
	AssetDarwinARM64 = "abcd-darwin-arm64"
	AssetDarwinAMD64 = "abcd-darwin-amd64"
	AssetLinuxARM64  = "abcd-linux-arm64"
	AssetLinuxAMD64  = "abcd-linux-amd64"
	// AssetChecksums is the manifest every install form verifies against, and
	// the one fixed-name asset the page links.
	AssetChecksums = "checksums.txt"
)

// LinkedBinaryAssets is the set of per-platform binaries the page offers.
var LinkedBinaryAssets = []string{AssetDarwinARM64, AssetDarwinAMD64, AssetLinuxARM64, AssetLinuxAMD64}

// composer holds everything the landing page is composed from.
type composer struct {
	repoRoot string
	// root is repoRoot opened as an os.Root containment scope. Every read of a
	// repo-relative source path resolves through it, so a committed directory
	// symlink (git mode 120000) planted as an ANCESTOR of a composed page cannot
	// walk the read outside the repository — the counterpart of the build's
	// destination os.Root, and the ancestor-safe form fsutil.ReadGuardedInRoot
	// gives that leaf-only fsutil.ReadGuarded cannot (gh #487).
	root     *os.Root
	manifest Manifest
	ui       UI
	repo     RepoMeta
	assets   *assetPipe
	graph    lint.RecordGraph
	history  History
	stamp    BuildStamp
	// pages caches each loaded source page: the composer reads several of them
	// twice (a chapter's own layout, and the anchors the navigation needs), and
	// reading a file twice invites two different answers.
	pages map[string]*docPage
	// beta is true while the newest released version's major is 0.
	beta bool
	// version and releaseDate describe the newest dated CHANGELOG heading.
	version     string
	releaseDate string
}

// docPage is one loaded source page.
type docPage struct {
	Rel      string
	Dir      string
	Sections []Section
}

// loadPage reads a composed page and splits it into sections.
func (c *composer) loadPage(rel string) (*docPage, error) {
	if p, ok := c.pages[rel]; ok {
		return p, nil
	}
	data, err := fsutil.ReadGuardedInRoot(c.root, rel, maxPageBytes)
	if err != nil {
		return nil, fmt.Errorf("site: composing %s: %w", rel, err)
	}
	body, consumed := StripFrontmatter(string(data))
	secs, err := Sections(rel, body, consumed)
	if err != nil {
		return nil, err
	}
	if len(secs) == 0 {
		return nil, fmt.Errorf("site: %s is empty; the manifest selects it as a source of text", rel)
	}
	p := &docPage{Rel: rel, Dir: path.Dir(rel), Sections: secs}
	if c.pages == nil {
		c.pages = map[string]*docPage{}
	}
	c.pages[rel] = p
	return p, nil
}

// renderer builds the markdown renderer for one page: images resolve relative
// to it, and its repo-relative links become site routes.
func (c *composer) renderer(p *docPage) *Renderer {
	return &Renderer{
		UI: c.ui,
		Image: func(src, alt string, at Source) (string, error) {
			return c.assets.render(p.Dir, src, alt, at)
		},
		Link: func(href string, at Source) string { return siteHref(p.Dir, href, c.repo.Repository) },
	}
}

// siteHref maps a link as the record wrote it to a link the site serves.
//
// An absolute URL and an in-page fragment are already right. A repo-relative
// markdown path under `docs/` becomes the docs route that renders it. A
// repo-relative markdown path ANYWHERE ELSE — a record file, a root document —
// has no page on this site yet, so it becomes the forge's own view of that file:
// a link that works today, rather than a relative path that 404s the moment
// somebody follows it. When the record explorer ships those targets get real
// pages and this arm narrows; a broken link in the meantime is not an
// acceptable placeholder.
func siteHref(pageDir, href, forge string) string {
	switch {
	case href == "",
		strings.HasPrefix(href, "http://"),
		strings.HasPrefix(href, "https://"),
		strings.HasPrefix(href, "mailto:"),
		strings.HasPrefix(href, "#"),
		strings.HasPrefix(href, "/"):
		return href
	}
	target, frag, _ := strings.Cut(href, "#")
	if !strings.HasSuffix(target, ".md") {
		return href
	}
	rel := path.Clean(path.Join(pageDir, target))
	if !strings.HasPrefix(rel, "docs/") {
		if forge == "" || !fsutil.ValidRelPath(rel) {
			return href
		}
		out := forge + "/blob/main/" + rel
		if frag != "" {
			out += "#" + frag
		}
		return out
	}
	route := strings.TrimSuffix(strings.TrimPrefix(rel, "docs/"), ".md")
	if base := path.Base(route); strings.EqualFold(base, "README") {
		route = path.Dir(route)
		if route == "." {
			route = ""
		}
	}
	out := "/docs/"
	if route != "" {
		out += route + "/"
	}
	if frag != "" {
		out += "#" + frag
	}
	return out
}

// srcAttr renders the provenance attribute every composed block carries.
func srcAttr(rel, anchor string) string {
	v := rel
	if anchor != "" {
		v += "#" + anchor
	}
	return ` data-src="` + escapeAttr(v) + `"`
}

// isImageBlock reports whether a block is an image on its own — the shape the
// layouts lift out as a figure, a portrait or a card icon.
func isImageBlock(b Block) bool { return strings.HasPrefix(strings.TrimSpace(b.Text), "![") }

// unwrapParagraph strips the paragraph wrapper from a rendered single block, so
// an image can sit directly inside a figure or a table header.
func unwrapParagraph(h string) string {
	h = strings.TrimPrefix(h, "<p>")
	return strings.TrimSuffix(h, "</p>")
}

// ComposeLanding renders the whole landing page.
func (c *composer) ComposeLanding() (string, error) {
	hero, err := c.hero()
	if err != nil {
		return "", err
	}
	var chapters strings.Builder
	for _, ch := range c.manifest.Home.Chapters {
		h, err := c.chapter(ch)
		if err != nil {
			return "", err
		}
		chapters.WriteString(h)
	}
	var b strings.Builder
	b.WriteString(c.head())
	b.WriteString(c.header())
	b.WriteString(`<main id="app"><div class="page">`)
	b.WriteString(hero)
	b.WriteString(chapters.String())
	b.WriteString("</div></main>")
	b.WriteString(c.footer())
	b.WriteString("</body>\n</html>\n")
	return b.String(), nil
}

// head renders the document head. The one external request the page makes is
// the font stylesheet; there are no analytics, no trackers and no scripts from
// anywhere but this build (adr-38, adr-48).
func (c *composer) head() string { return c.headWith("", "") }

// headWith renders the head of a page that names itself — every explorer route
// does — and that may need one more script than the landing page.
//
// A named page's title is `<page> · <project>`: the leading half is what a tab
// strip and a search result show first, and it is a record heading or a
// navigation label rather than a sentence written here.
func (c *composer) headWith(pageTitle, script string) string {
	title, desc := c.repo.Name, ""
	if blk, ok := c.identity(); ok {
		title, desc = blk.Title, blk.Tagline
	}
	if pageTitle != "" {
		title = pageTitle + " · " + title
		desc = ""
	}
	var b strings.Builder
	b.WriteString("<!doctype html>\n<html lang=\"en\">\n<head>\n")
	b.WriteString(`<meta charset="utf-8">` + "\n")
	b.WriteString(`<meta name="viewport" content="width=device-width, initial-scale=1">` + "\n")
	b.WriteString("<title>" + escapeText(title) + "</title>\n")
	if desc != "" {
		b.WriteString(`<meta name="description" content="` + escapeAttr(desc) + `">` + "\n")
	}
	b.WriteString(`<link rel="preconnect" href="https://fonts.googleapis.com">` + "\n")
	b.WriteString(`<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>` + "\n")
	b.WriteString(`<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Bricolage+Grotesque:opsz,wdth,wght@12..96,75..100,300..800&family=Newsreader:ital,opsz,wght@0,6..72,300..700;1,6..72,300..700&family=IBM+Plex+Mono:wght@400;500;600&display=swap">` + "\n")
	// Root-absolute, because the shared assets sit at one served path and the
	// pages that link them sit at every depth the site has.
	b.WriteString(`<link rel="stylesheet" href="/site.css">` + "\n")
	b.WriteString(`<script src="/site.js" defer></script>` + "\n")
	if script != "" {
		b.WriteString(`<script src="` + escapeAttr(script) + `" defer></script>` + "\n")
	}
	b.WriteString("</head>\n<body>\n")
	return b.String()
}

// identity reads the canonical Identity block. A repository that records none
// renders the page without it rather than failing: the hero's headline and lede
// come from the rationale page, and those are what carry the page.
func (c *composer) identity() (positioning.Block, bool) {
	blk, err := positioning.ParseBlock(c.repoRoot, positioning.BlockLocation{
		File: c.manifest.Identity.File, Heading: c.manifest.Identity.Heading,
	})
	if err != nil {
		return positioning.Block{}, false
	}
	return blk, true
}

// identitySrc is the provenance of every span the Identity block supplies.
func (c *composer) identitySrc() string {
	return srcAttr(c.manifest.Identity.File, Slug(c.manifest.Identity.Heading))
}

// brandName is the short handle the header shows beside the mark: the first word
// of the canonical title, which is the project's own name for itself.
func (c *composer) brandName() string {
	if blk, ok := c.identity(); ok {
		if f := strings.Fields(blk.Title); len(f) > 0 {
			return f[0]
		}
	}
	return c.repo.Name
}

// header renders the shared site header on the landing page.
func (c *composer) header() string { return c.headerFor("") }

// headerFor renders the shared site header: the mark, the beta badge while the
// release major is 0, the navigation, and the release pill.
//
// `active` is the top-level route the current page belongs to. It stays lit
// while the reader moves between that route's sub-pages, so the Record label
// does not go dark on the way from the dashboard to the chart.
func (c *composer) headerFor(active string) string {
	var b strings.Builder
	b.WriteString(`<header class="site-head"><div class="wrap">`)
	b.WriteString(`<a class="brand" href="/">`)
	// The mark is a committed asset like any other picture; the stylesheet sizes
	// it, so its own dimensions ride along unchanged. A repository without one
	// gets a wordmark.
	if logo, err := c.assets.render("docs/assets/img", "logo.png", "", Source{Path: ManifestRelPath}); err == nil {
		b.WriteString(logo)
	}
	b.WriteString(escapeText(c.brandName()) + `</a>`)
	if c.beta {
		b.WriteString(`<span class="beta">` + escapeText(c.ui.Beta) + `</span>`)
	}
	// No aria-label: this is the page's only nav landmark, so a screen reader
	// already announces it unambiguously, and labelling it "Story" would name it
	// after one of its links.
	b.WriteString(`<nav class="site-nav">`)
	on := func(route string) string {
		if route != active {
			return ""
		}
		return ` class="on" aria-current="page"`
	}
	b.WriteString(`<a href="/#` + escapeAttr(c.firstChapterAnchor()) + `">` + escapeText(c.ui.NavStory) + `</a>`)
	b.WriteString(`<a href="/#` + escapeAttr(c.installChapterAnchor()) + `">` + escapeText(c.ui.NavInstall) + `</a>`)
	b.WriteString(`<a href="/docs/">` + escapeText(c.ui.NavDocs) + `</a>`)
	b.WriteString(`<a href="/record/"` + on("/record/") + `>` + escapeText(c.ui.NavRecord) + `</a>`)
	if c.repo.Repository != "" {
		b.WriteString(`<a class="gh" href="` + escapeAttr(c.repo.Repository) + `">` + escapeText(c.forgeLabel()) + ` ↗</a>`)
	}
	b.WriteString(`</nav>`)
	if c.version != "" && c.repo.Repository != "" {
		b.WriteString(`<a class="pill" href="` + escapeAttr(c.repo.Repository) + `/releases/latest" title="` + escapeAttr(c.ui.LatestRelease) + `">`)
		b.WriteString(`<span class="dot"></span>v` + escapeText(c.version))
		if c.releaseDate != "" {
			b.WriteString(`<span class="pdate"> · ` + escapeText(c.releaseDate) + `</span>`)
		}
		b.WriteString(`</a>`)
	}
	b.WriteString(`</div></header>`)
	return b.String()
}

// firstChapterAnchor and installChapterAnchor are the two in-page destinations
// the navigation offers. They are the composed chapters' own anchors, so a
// renamed page moves the link with it.
func (c *composer) chapterAnchor(ch Chapter) string {
	p, err := c.loadPage(ch.Page)
	if err != nil {
		return ""
	}
	return Slug(p.Sections[0].Title)
}

func (c *composer) firstChapterAnchor() string {
	if len(c.manifest.Home.Chapters) == 0 {
		return ""
	}
	return c.chapterAnchor(c.manifest.Home.Chapters[0])
}

func (c *composer) installChapterAnchor() string {
	for _, ch := range c.manifest.Home.Chapters {
		if ch.Layout == LayoutInstall {
			return c.chapterAnchor(ch)
		}
	}
	return c.firstChapterAnchor()
}

// footer renders the site footer: file names, links, and build metadata only.
func (c *composer) footer() string {
	var b strings.Builder
	b.WriteString(`<footer class="site-foot"><div class="wrap">`)
	// The year comes from the build stamp, which a repository with no releases
	// and no injected date cannot supply. "© <nothing> REPPL · MIT" is worse
	// than no line, so the span is omitted rather than rendered half-empty.
	year := c.stamp.GeneratedAt
	if len(year) >= 4 {
		year = year[:4]
	} else {
		year = ""
	}
	if year != "" && c.repo.AuthorName != "" && c.repo.License != "" {
		// Two facts, spaced rather than punctuated: a dot between them reads as
		// a full stop that is not one, and the licence is the quieter of the two.
		b.WriteString(`<span>© ` + escapeText(year) + " " + escapeText(c.repo.AuthorName) +
			` <span class="quiet">` + escapeText(c.repo.License) + `</span></span>`)
	}
	if c.repo.Repository != "" {
		for _, f := range []string{"SECURITY.md", "ACKNOWLEDGEMENTS.md", "CITATION.cff", "CHANGELOG.md"} {
			if _, err := c.root.Stat(f); err != nil {
				continue
			}
			b.WriteString(`<a href="` + escapeAttr(c.repo.Repository+"/blob/main/"+f) + `">` + escapeText(f) + `</a>`)
		}
		b.WriteString(`<a href="` + escapeAttr(c.repo.Repository) + `">` + escapeText(c.forgeLabel()) + `</a>`)
	}
	var meta []string
	if c.beta {
		meta = append(meta, strings.ToLower(c.ui.Beta))
	}
	// What this build IS. A preview is built from an untagged tree, so it says
	// so; only a build of a release names one. The Beta badge above is a separate
	// fact — it reads the newest CHANGELOG major, which is still 0.x whether or
	// not this particular build was cut from a tag.
	switch {
	case c.stamp.Preview:
		meta = append(meta, c.ui.Unreleased)
	case c.version != "":
		meta = append(meta, "v"+c.version)
	}
	if c.stamp.Commit != "" {
		meta = append(meta, c.stamp.Commit)
	}
	if len(meta) > 0 {
		// The build stamp is three independent facts, so they are set apart by
		// space and weight rather than strung together on dots.
		b.WriteString(`<span class="mono small foot-meta">`)
		for _, m := range meta {
			b.WriteString(`<span>` + escapeText(m) + `</span>`)
		}
		b.WriteString(`</span>`)
	}
	b.WriteString(`</div></footer>`)
	return b.String()
}

// forgeLabel names the header and footer forge links: the interface string
// declared for the repository's forge host, or the owner/name handle where no
// name is declared — the generic fallback a fixture forge exercises. The name
// comes from ui.json's closed allowlist, never from code, so the generic
// explorer carries no forge assumption.
func (c *composer) forgeLabel() string {
	if h := forgeHost(c.repo.Repository); h != "" {
		if name, ok := c.ui.ForgeNames[h]; ok && strings.TrimSpace(name) != "" {
			return name
		}
	}
	return repoHandle(c.repo.Repository)
}

// forgeHost is the host of a forge URL: the scheme stripped, the path cut.
func forgeHost(u string) string {
	if i := strings.Index(u, "://"); i >= 0 {
		u = u[i+3:]
	}
	if i := strings.IndexByte(u, '/'); i >= 0 {
		u = u[:i]
	}
	return u
}

// repoHandle is the owner/name of a forge URL — a name, not a sentence.
func repoHandle(u string) string {
	u = strings.TrimSuffix(u, "/")
	parts := strings.Split(u, "/")
	if len(parts) < 2 {
		return u
	}
	return parts[len(parts)-2] + "/" + parts[len(parts)-1]
}

// hero renders the opening: the eyebrow, tagline and pitch from the Identity
// block, the headline and lede from the rationale page's H1 section, and that
// section's first image as the figure.
func (c *composer) hero() (string, error) {
	p, err := c.loadPage(c.manifest.Home.Hero.Page)
	if err != nil {
		return "", err
	}
	r := c.renderer(p)
	h1 := p.Sections[0]
	bl := Blocks(h1.Body, h1.BodyLine)

	var lede, figure string
	for _, b := range bl {
		if isImageBlock(b) {
			if figure == "" && c.manifest.Home.Hero.Figure == figureFirstImage {
				h, err := r.RenderBlock(p.Rel, b)
				if err != nil {
					return "", err
				}
				figure = h
			}
			continue
		}
		if lede == "" {
			h, err := r.RenderBlock(p.Rel, b)
			if err != nil {
				return "", err
			}
			lede = h
		}
	}

	var b strings.Builder
	b.WriteString(`<section class="hero"><div class="wrap"><div class="grid"><div>`)
	blk, haveIdentity := c.identity()
	if haveIdentity {
		b.WriteString(`<p class="eyebrow"` + c.identitySrc() + `>` + escapeText(blk.Title) + `</p>`)
	}
	b.WriteString(`<h1` + srcAttr(p.Rel, "") + `>` + escapeText(h1.Title) + `</h1>`)
	if lede != "" {
		b.WriteString(`<div class="lede"` + srcAttr(p.Rel, h1.Anchor) + `>` + lede + `</div>`)
	}
	b.WriteString(`</div><div class="hero-aside">`)
	if haveIdentity {
		b.WriteString(`<p class="tagline"` + c.identitySrc() + `>` + escapeText(blk.Tagline) + `</p>`)
		if blk.Pitch != "" {
			b.WriteString(`<p class="pitch"` + c.identitySrc() + `>` + escapeText(blk.Pitch) + `</p>`)
		}
	}
	b.WriteString(c.downloads())
	b.WriteString(`</div></div>`)
	if figure != "" {
		b.WriteString(`<figure class="comic"` + srcAttr(p.Rel, h1.Anchor) + `>` + figure + `</figure>`)
	}
	b.WriteString(`</div></section>`)
	return b.String(), nil
}

// downloads renders the direct release links. Each link's text is the platform's
// name from ui.json and its target is the released asset's own file name, so the
// block is names and URLs and nothing else. Without a known forge URL there is
// nothing to link to, and the block is omitted.
func (c *composer) downloads() string {
	if c.repo.Repository == "" {
		return ""
	}
	base := c.repo.Repository + "/releases/latest/download/"
	link := func(asset, label string) string {
		return `<a href="` + escapeAttr(base+asset) + `" title="` + escapeAttr(asset) + `">` + escapeText(label) + `</a>`
	}
	// The two builds for one operating system share a row, and the second is
	// named by the part of its label that differs from the first.
	tail := func(s string) string {
		if _, after, ok := strings.Cut(s, " · "); ok {
			return after
		}
		return s
	}
	var b strings.Builder
	b.WriteString(`<div class="getit"><span class="eyebrow">` + escapeText(c.ui.LatestRelease) + `</span>`)
	b.WriteString(`<span class="dl">` + link(AssetDarwinARM64, c.ui.Platform.DarwinARM64) + ` · ` + link(AssetDarwinAMD64, tail(c.ui.Platform.DarwinAMD64)) + `</span>`)
	b.WriteString(`<span class="dl">` + link(AssetLinuxARM64, c.ui.Platform.LinuxARM64) + ` · ` + link(AssetLinuxAMD64, tail(c.ui.Platform.LinuxAMD64)) + `</span>`)
	b.WriteString(`<a class="more" href="#` + escapeAttr(c.installChapterAnchor()) + `">` + escapeText(c.ui.CTAInstall) + ` ↓</a>`)
	b.WriteString(`</div>`)
	return b.String()
}

// chapter renders one lettered section of the landing page.
func (c *composer) chapter(ch Chapter) (string, error) {
	p, err := c.loadPage(ch.Page)
	if err != nil {
		return "", err
	}
	title := p.Sections[0].Title
	anchor := Slug(title)

	var body string
	switch ch.Layout {
	case LayoutCardsFromH2:
		body, err = c.cardsFromH2(p)
	case LayoutLeadInCards:
		body, err = c.leadInCards(p, ch)
		if err == nil && ch.Feature != nil {
			var feat string
			feat, err = c.featureBlock(ch.Feature)
			body += feat
		}
	case LayoutProse:
		var text, figure string
		text, figure, err = c.prose(p)
		body = `<div class="loop"><div>` + text + `</div>` + figure + `</div>`
	case LayoutInstall:
		body, err = c.install(p, ch)
	}
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString(`<section class="chapter" id="` + escapeAttr(anchor) + `"><div class="wrap">`)
	b.WriteString(`<div class="head"><div class="letter ` + escapeAttr(ch.Letter) + `" aria-hidden="true">` + escapeText(ch.Letter) + `</div>`)
	b.WriteString(`<h2` + srcAttr(p.Rel, "") + `>` + escapeText(title) + `</h2></div>`)
	b.WriteString(`<div class="body stack" style="--stack:22px">` + body + `</div>`)
	b.WriteString(`</div></section>`)
	return b.String(), nil
}

// cardsFromH2 turns a page's H2 sections into cards, with the H1 section's text
// as the intro. A section opening on an image gets it as a portrait.
func (c *composer) cardsFromH2(p *docPage) (string, error) {
	r := c.renderer(p)
	intro, err := r.RenderBlocks(p.Rel, Blocks(p.Sections[0].Body, p.Sections[0].BodyLine))
	if err != nil {
		return "", err
	}
	var cards strings.Builder
	for _, s := range p.Sections[1:] {
		bl := Blocks(s.Body, s.BodyLine)
		portrait := ""
		if len(bl) > 0 && isImageBlock(bl[0]) {
			h, err := r.RenderBlock(p.Rel, bl[0])
			if err != nil {
				return "", err
			}
			portrait = strings.Replace(unwrapParagraph(h), `<img `, `<img class="portrait-lg" `, 1)
			bl = bl[1:]
		}
		body, err := r.RenderBlocks(p.Rel, bl)
		if err != nil {
			return "", err
		}
		cards.WriteString(`<div class="card"` + srcAttr(p.Rel, s.Anchor) + `>` + portrait +
			`<h3>` + escapeText(s.Title) + `</h3>` + body + `</div>`)
	}
	return `<div class="prose lede-2"` + srcAttr(p.Rel, p.Sections[0].Anchor) + `>` + intro +
		`</div><div class="cards two">` + cards.String() + `</div>`, nil
}

// leadInCards turns a page's bold lead-in paragraphs into cards. An image line
// before a lead-in is that card's icon; a table becomes a scrollable table, with
// the role portraits from the roles page sitting above the column labels.
func (c *composer) leadInCards(p *docPage, ch Chapter) (string, error) {
	r := c.renderer(p)
	s := p.Sections[0]
	var out, cards strings.Builder
	icon := ""
	flush := func() {
		if cards.Len() > 0 {
			out.WriteString(`<div class="cards">` + cards.String() + `</div>`)
			cards.Reset()
		}
	}
	for _, b := range Blocks(s.Body, s.BodyLine) {
		text := strings.TrimLeft(b.Text, " ")
		switch {
		case strings.HasPrefix(text, "|"):
			flush()
			th, err := r.RenderBlock(p.Rel, b)
			if err != nil {
				return "", err
			}
			if ch.TablePortraits != "" {
				th, err = c.tablePortraits(th, ch.TablePortraits)
				if err != nil {
					return "", err
				}
			}
			// The renderer supplies the table's own overflow container, so this
			// wrapper carries nothing but the provenance attribute.
			out.WriteString(`<div` + srcAttr(p.Rel, s.Anchor) + `>` + th + `</div>`)
		case isImageBlock(b):
			h, err := r.RenderBlock(p.Rel, b)
			if err != nil {
				return "", err
			}
			// The manifest declares whether an image before a lead-in is that
			// card's icon. Without the declaration the picture stays a picture
			// in the running text, which is what the source page shows.
			if ch.Icons != iconsBeforeLeadIn {
				flush()
				out.WriteString(`<div class="prose"` + srcAttr(p.Rel, s.Anchor) + `>` + h + `</div>`)
				continue
			}
			icon = unwrapParagraph(h)
		default:
			title, rest, ok := leadIn(b.Text)
			if !ok {
				flush()
				h, err := r.RenderBlock(p.Rel, b)
				if err != nil {
					return "", err
				}
				out.WriteString(`<div class="prose"` + srcAttr(p.Rel, s.Anchor) + `>` + h + `</div>`)
				continue
			}
			h, err := r.RenderBlock(p.Rel, Block{Text: rest, Line: b.Line})
			if err != nil {
				return "", err
			}
			cards.WriteString(`<div class="card"` + srcAttr(p.Rel, s.Anchor) + `>`)
			if icon != "" {
				cards.WriteString(`<div class="icon">` + icon + `</div>`)
				icon = ""
			}
			cards.WriteString(`<h3>` + escapeText(title) + `</h3>` + h + `</div>`)
		}
	}
	flush()
	return out.String(), nil
}

// leadIn reads a paragraph that opens with a bold phrase — optionally after up
// to two leading words — and splits it into that phrase and the rest.
func leadIn(text string) (title, rest string, ok bool) {
	words := 0
	i := 0
	for i < len(text) && words <= 2 {
		if text[i] == '*' && i+1 < len(text) && text[i+1] == '*' {
			end := strings.Index(text[i+2:], "**")
			if end < 0 || strings.Contains(text[i+2:i+2+end], "*") {
				return "", "", false
			}
			title = strings.TrimSpace(text[:i] + text[i+2:i+2+end])
			rest = strings.TrimLeft(text[i+2+end+2:], " \t\n\r")
			return title, rest, title != ""
		}
		// A word ends at ANY whitespace, not just a space: the script's `\S+\s+`
		// counts a line-wrapped lead-in the same as a single-line one, and the
		// record wraps its paragraphs.
		j := i
		for j < len(text) && !isSpace(text[j]) {
			j++
		}
		for j < len(text) && isSpace(text[j]) {
			j++
		}
		if j == i {
			break
		}
		i = j
		words++
	}
	return "", "", false
}

// isSpace reports whether a byte is markdown whitespace.
func isSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' }

// tablePortraits puts the role portraits above the column labels they name. The
// portrait is not configured by asset name: the manifest names the PAGE the
// roles live on, and each column label is matched to the section of that page
// whose title names the same role, so a renamed portrait file follows the page.
func (c *composer) tablePortraits(tableHTML, wanted string) (string, error) {
	p, err := c.chapterNamed(wanted)
	if err != nil || p == nil {
		return tableHTML, err
	}
	r := c.renderer(p)
	for _, s := range p.Sections[1:] {
		bl := Blocks(s.Body, s.BodyLine)
		if len(bl) == 0 || !isImageBlock(bl[0]) {
			continue
		}
		label, ok := columnLabelFor(tableHTML, s.Title)
		if !ok {
			continue
		}
		img, err := r.RenderBlock(p.Rel, bl[0])
		if err != nil {
			return "", err
		}
		img = strings.Replace(unwrapParagraph(img), `<img `, `<img class="th-portrait" `, 1)
		// `label` is the cell's text AS IT SITS in the rendered table, entities
		// and all. Re-escaping it here would turn a header like "Research &
		// design" into `&amp;amp;`, match nothing, and silently drop the
		// portrait — so it is substituted back exactly as it was taken.
		tableHTML = strings.Replace(tableHTML,
			"<th>"+label+"</th>",
			"<th>"+img+"<span>"+label+"</span></th>", 1)
	}
	return tableHTML, nil
}

// chapterNamed resolves a manifest cross-reference from one chapter to another.
// A chapter answers to its letter, to its page's file stem, and to the slug of
// its H1 — the three ways a manifest author would naturally write "the roles
// chapter" — and to nothing else. A name that matches none of them is not an
// error: the cross-reference is an enrichment, and losing it costs a portrait,
// not a page.
func (c *composer) chapterNamed(name string) (*docPage, error) {
	for _, ch := range c.manifest.Home.Chapters {
		p, err := c.loadPage(ch.Page)
		if err != nil {
			return nil, err
		}
		stem := strings.TrimSuffix(path.Base(ch.Page), ".md")
		if ch.Letter == name || stem == name || Slug(p.Sections[0].Title) == name {
			return p, nil
		}
	}
	return nil, nil
}

// columnLabelFor finds the table column whose label names the same role as a
// section title — "Facilitator" and "Technical facilitator" are one role — and
// returns the cell EXACTLY as it sits in the rendered table, so the caller can
// substitute it back. An ambiguous or unmatched label gets no portrait rather
// than the wrong one.
//
// Matching is done on the DECODED text: a header reading "Research & design"
// renders as `Research &amp; design`, and comparing that against a section title
// would fail on the entity alone — quietly, with the only symptom a missing
// picture.
func columnLabelFor(tableHTML, sectionTitle string) (string, bool) {
	var labels []string
	rest := tableHTML
	for {
		_, after, found := strings.Cut(rest, "<th>")
		if !found {
			break
		}
		cell, tail, found := strings.Cut(after, "</th>")
		rest = tail
		if !found {
			break
		}
		if !strings.Contains(cell, "<") && strings.TrimSpace(cell) != "" {
			labels = append(labels, cell)
		}
	}
	title := strings.ToLower(sectionTitle)
	var hits []string
	for _, l := range labels {
		low := strings.ToLower(decodeEntities(l))
		if strings.Contains(title, low) || strings.Contains(low, title) {
			hits = append(hits, l)
		}
	}
	if len(hits) != 1 {
		return "", false
	}
	return hits[0], true
}

// decodeEntities reverses escapeText for comparison purposes. It is the exact
// inverse of what the renderer emits — no more — so it cannot resurrect markup
// that was never there; `&amp;` is undone last, because undoing it first would
// turn `&amp;lt;` into `<`.
func decodeEntities(s string) string {
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", `"`)
	s = strings.ReplaceAll(s, "&#39;", "'")
	return strings.ReplaceAll(s, "&amp;", "&")
}

// prose renders a page as running text with its sub-headings, lifting the first
// image out as a figure the layout floats beside the words.
func (c *composer) prose(p *docPage) (string, string, error) {
	r := c.renderer(p)
	var out strings.Builder
	figure := ""
	for _, s := range p.Sections {
		anchor := s.Anchor
		if anchor == "" {
			anchor = Slug(s.Title)
		}
		if s.Level > 1 {
			out.WriteString(`<h3 class="subh"` + srcAttr(p.Rel, anchor) + `>` + escapeText(s.Title) + `</h3>`)
		}
		bl := Blocks(s.Body, s.BodyLine)
		if figure == "" {
			for i, b := range bl {
				if !isImageBlock(b) {
					continue
				}
				h, err := r.RenderBlock(p.Rel, b)
				if err != nil {
					return "", "", err
				}
				figure = `<figure class="card loopfig"` + srcAttr(p.Rel, anchor) + `>` + unwrapParagraph(h) + `</figure>`
				bl = append(bl[:i:i], bl[i+1:]...)
				break
			}
		}
		body, err := r.RenderBlocks(p.Rel, bl)
		if err != nil {
			return "", "", err
		}
		out.WriteString(`<div class="prose"` + srcAttr(p.Rel, anchor) + `>` + body + `</div>`)
	}
	return out.String(), figure, nil
}

// install renders the install chapter as a tab row: the manifest's left-hand
// sections first — the product thinker's path — then the command-line group
// under its own label, made of the lead section's sub-headings and whatever H2s
// remain. Each operating-system panel opens with the lead section's paragraph
// and closes with the page's closing section, its tail folded away.
func (c *composer) install(p *docPage, ch Chapter) (string, error) {
	r := c.renderer(p)
	intro, err := r.RenderBlocks(p.Rel, Blocks(p.Sections[0].Body, p.Sections[0].BodyLine))
	if err != nil {
		return "", err
	}

	var h2 []int
	leadIdx := -1
	for i, s := range p.Sections {
		if s.Level != 2 {
			continue
		}
		h2 = append(h2, i)
		// FIRST match wins, as the script's `next(...)` does. A page carrying two
		// sections of one name is malformed either way, but resolving to the last
		// would silently pick the opposite section from the one a reader of the
		// executable spec expects.
		if leadIdx < 0 && s.Title == ch.Lead {
			leadIdx = i
		}
	}
	if leadIdx < 0 {
		return "", fmt.Errorf("site: %s has no '## %s' section; the manifest names it as the install lead", p.Rel, ch.Lead)
	}

	var h3 []int
	for i := leadIdx + 1; i < len(p.Sections); i++ {
		if p.Sections[i].Level == 2 {
			break
		}
		if p.Sections[i].Level == 3 {
			h3 = append(h3, i)
		}
	}
	afterIdx := -1
	for _, i := range h3 {
		if afterIdx < 0 && ch.After != "" && p.Sections[i].Title == ch.After {
			afterIdx = i
		}
	}

	left := map[int]bool{}
	var leftOrder []int
	for _, i := range h2 {
		for _, name := range ch.Left {
			if p.Sections[i].Title == name {
				left[i] = true
				leftOrder = append(leftOrder, i)
			}
		}
	}
	var osTabs []int
	for _, i := range h3 {
		if i != afterIdx {
			osTabs = append(osTabs, i)
		}
	}
	group := append([]int{}, osTabs...)
	for _, i := range h2 {
		if i != leadIdx && !left[i] {
			group = append(group, i)
		}
	}

	type tab struct {
		idx  int
		isOS bool
	}
	var tabs []tab
	for _, i := range leftOrder {
		tabs = append(tabs, tab{i, false})
	}
	for _, i := range osTabs {
		tabs = append(tabs, tab{i, true})
	}
	for _, i := range group {
		if !containsInt(osTabs, i) {
			tabs = append(tabs, tab{i, false})
		}
	}
	first := map[int]bool{}
	if len(tabs) > 0 {
		first[tabs[0].idx] = true
	}

	btn := func(i int) string {
		s := p.Sections[i]
		sel, tabindex := "false", "-1"
		if first[i] {
			sel, tabindex = "true", "0"
		}
		return `<button role="tab" id="tab-` + escapeAttr(s.Anchor) + `" aria-selected="` + sel +
			`" aria-controls="panel-` + escapeAttr(s.Anchor) + `" tabindex="` + tabindex + `"` +
			srcAttr(p.Rel, s.Anchor) + `>` + escapeText(s.Title) + `</button>`
	}
	var row strings.Builder
	for _, i := range leftOrder {
		row.WriteString(btn(i))
	}
	row.WriteString(`<div class="tabgroup"><span class="grp">` + escapeText(c.ui.CLIGroup) + `</span>`)
	for _, i := range group {
		row.WriteString(btn(i))
	}
	row.WriteString(`</div>`)

	lead := p.Sections[leadIdx]
	_, installStatErr := c.root.Stat(installTemplateRelPath)
	servesInstallScript := installStatErr == nil
	var panels strings.Builder
	for _, t := range tabs {
		s := p.Sections[t.idx]
		bl := Blocks(s.Body, s.BodyLine)
		hasCode := false
		for _, b := range bl {
			if strings.HasPrefix(b.Text, "```") {
				hasCode = true
			}
		}
		head := ""
		if t.isOS && hasCode {
			h, err := r.RenderBlocks(p.Rel, Blocks(lead.Body, lead.BodyLine))
			if err != nil {
				return "", err
			}
			head = `<div class="prose small tabprose lead"` + srcAttr(p.Rel, lead.Anchor) + `>` + h + `</div>`
		}
		extra := ""
		if t.isOS && hasCode && afterIdx >= 0 {
			after := p.Sections[afterIdx]
			abl := Blocks(after.Body, after.BodyLine)
			opening, err := r.RenderBlocks(p.Rel, headBlocks(abl, 2))
			if err != nil {
				return "", err
			}
			tail, err := r.RenderBlocks(p.Rel, tailBlocks(abl, 2))
			if err != nil {
				return "", err
			}
			extra = `<div class="prose small tabprose after"` + srcAttr(p.Rel, after.Anchor) + `><h4>` +
				escapeText(after.Title) + `</h4>` + opening +
				`<details class="more"><summary>` + escapeText(c.ui.More) + `</summary>` + tail + `</details></div>`
		}
		body, err := r.RenderBlocks(p.Rel, headBlocks(bl, 4))
		if err != nil {
			return "", err
		}
		if len(bl) > 4 {
			rest, err := r.RenderBlocks(p.Rel, tailBlocks(bl, 4))
			if err != nil {
				return "", err
			}
			body += `<details class="more"><summary>` + escapeText(c.ui.More) + `</summary>` + rest + `</details>`
		}
		// The read-before-you-run link, beside the command it reads. It is offered
		// only where the command is — an operating-system panel — and only when
		// this repository actually serves the script, because the build writes
		// /install.sh from that template and a link to a route nothing emitted is
		// a dead link on the one page that is asking for trust.
		read := ""
		if t.isOS && hasCode && servesInstallScript {
			read = `<p class="small muted readscript"><a href="/` + installScriptName + `">` +
				escapeText(c.ui.ReadScript) + `</a></p>`
		}
		hidden := " hidden"
		if first[t.idx] {
			hidden = ""
		}
		panels.WriteString(`<div role="tabpanel" id="panel-` + escapeAttr(s.Anchor) + `" aria-labelledby="tab-` +
			escapeAttr(s.Anchor) + `"` + hidden + srcAttr(p.Rel, s.Anchor) + `>` + head +
			`<div class="prose small tabbody">` + body + `</div>` + read + extra + `</div>`)
	}

	var b strings.Builder
	b.WriteString(`<div class="stack installwrap" style="--stack:16px">`)
	b.WriteString(`<div class="prose"` + srcAttr(p.Rel, p.Sections[0].Anchor) + `>` + intro + `</div>`)
	b.WriteString(`<div class="tabs install"><div role="tablist" aria-label="` + escapeAttr(p.Sections[0].Title) +
		`" data-mine-title="` + escapeAttr(c.ui.MatchesSystem) + `">` + row.String() + `</div>` + panels.String() + `</div>`)
	if c.repo.Repository != "" {
		rr := c.repo.Repository
		b.WriteString(`<p class="small muted">` +
			`<a href="` + escapeAttr(rr+"/releases/latest") + `">` + escapeText(c.ui.LatestRelease) + `</a> · ` +
			`<a href="` + escapeAttr(rr+"/releases/latest/download/"+AssetChecksums) + `">` + escapeText(AssetChecksums) + `</a> · ` +
			`<a href="` + escapeAttr(rr+"/blob/main/CHANGELOG.md") + `">CHANGELOG.md</a> · ` +
			`<a href="` + escapeAttr(rr+"/releases") + `">` + escapeText(c.ui.AllReleases) + `</a></p>`)
	}
	b.WriteString(`</div>`)
	return b.String(), nil
}

func headBlocks(bl []Block, n int) []Block {
	if len(bl) <= n {
		return bl
	}
	return bl[:n]
}

func tailBlocks(bl []Block, n int) []Block {
	if len(bl) <= n {
		return nil
	}
	return bl[n:]
}

func containsInt(xs []int, x int) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

// featureBlock quotes the record's own evidence that the process works: the
// newest shipped intent whose audit says its criteria were met, with its press
// release and first acceptance criterion verbatim. It is derived on every build,
// never pinned — the only testimonial adr-47 allows is a real one.
func (c *composer) featureBlock(f *Feature) (string, error) {
	node, ok := c.newestMetIntent()
	if !ok {
		return "", nil
	}
	data, err := fsutil.ReadGuardedInRoot(c.root, node.Path, maxPageBytes)
	if err != nil {
		return "", err
	}
	body, consumed := StripFrontmatter(string(data))
	secs, err := Sections(node.Path, body, consumed)
	if err != nil {
		return "", err
	}
	p := &docPage{Rel: node.Path, Dir: path.Dir(node.Path)}
	r := c.renderer(p)

	quote := ""
	crit := ""
	for _, s := range secs {
		switch s.Title {
		case "Press Release":
			if !f.FeatureWants(featurePartPR) {
				continue
			}
			h, err := r.RenderBlocks(node.Path, Blocks(s.Body, s.BodyLine))
			if err != nil {
				return "", err
			}
			quote = h
		case "Acceptance Criteria":
			if !f.FeatureWants(featurePartFirstAC) {
				continue
			}
			bl := Blocks(s.Body, s.BodyLine)
			if len(bl) == 0 {
				continue
			}
			text, line := firstListItem(bl[0])
			h, err := r.RenderBlock(node.Path, Block{Text: text, Line: line})
			if err != nil {
				return "", err
			}
			crit = h
		}
	}
	if quote == "" && crit == "" {
		return "", nil
	}

	var b strings.Builder
	b.WriteString(`<div class="quote"` + srcAttr(node.Path, Slug("Press Release")) + `><div class="pr"><span>`)
	if c.repo.Repository != "" {
		b.WriteString(`<a href="` + escapeAttr(c.repo.Repository+"/blob/main/"+node.Path) + `">` + escapeText(node.ID) + `</a>`)
	} else {
		b.WriteString(escapeText(node.ID))
	}
	b.WriteString(` · ` + escapeText(node.Lifecycle))
	if rel := c.releaseOf(node.ID); rel != "" {
		b.WriteString(` · v` + escapeText(rel))
	}
	b.WriteString(`</span><span>` + escapeText(c.ui.FromTheRecord) + `</span></div>`)
	b.WriteString(quote)
	if crit != "" {
		b.WriteString(`<div class="crit"` + srcAttr(node.Path, Slug("Acceptance Criteria")) + `>` + crit + `</div>`)
	}
	b.WriteString(`</div>`)
	return b.String(), nil
}

// firstListItem returns the first bullet of a list block, and the source line it
// sits on. The manifest asks for ONE acceptance criterion, so it gets one.
func firstListItem(b Block) (string, int) {
	lines := strings.Split(b.Text, "\n")
	first := strings.TrimLeft(lines[0], " \t")
	switch {
	case isUnorderedItem(first):
		first = strings.TrimSpace(first[2:])
	case orderedItemRe.MatchString(first):
		first = strings.TrimSpace(orderedItemRe.FindStringSubmatch(first)[2])
	default:
		return b.Text, b.Line
	}
	item := []string{first}
	for _, ln := range lines[1:] {
		t := strings.TrimLeft(ln, " \t")
		if isUnorderedItem(t) || orderedItemRe.MatchString(t) {
			break
		}
		item = append(item, ln)
	}
	// Rendered as prose, not as a one-item list: it is a quotation.
	return strings.TrimSpace(strings.Join(item, "\n")), b.Line
}

// newestMetIntent picks the shipped intent whose audit verdict is MET, most
// recently shipped first, id descending as the tie-break. "Shipped" and its date
// come from the git pass: the record moves an intent's FILE into shipped/, so
// the day that move landed is the day it shipped.
func (c *composer) newestMetIntent() (lint.RecordNode, bool) {
	var best lint.RecordNode
	bestDate := ""
	found := false
	for _, n := range c.graph.Nodes {
		if n.Type != "intent" || n.Lifecycle != "shipped" {
			continue
		}
		if !c.auditIsMet(n.Path) {
			continue
		}
		if c.pressReleaseIsUnwritten(n.Path) {
			continue
		}
		d := c.history.EnteredBucket(n.Path)
		switch {
		case !found,
			d > bestDate,
			d == bestDate && handleNum(n.ID) > handleNum(best.ID):
			best, bestDate, found = n, d, true
		}
	}
	return best, found
}

// pressReleaseIsUnwritten reports whether an intent's `## Press Release` is
// still, in its entirety, one of the placeholders the intent store mints.
//
// This is the ONE exclusion the derivation carries, and it is mechanical rather
// than a judgement about quality: the body, reduced to its words, must be a
// template `intent.IsSeedNote` recognises — the package that writes them owns
// the predicate, so a reworded template cannot leave the two disagreeing. An
// intent that has had a single sentence written into it is a candidate again.
//
// It exists because the featured quotation is the page's only testimonial, and
// the record can legitimately hold a shipped, audit-met intent whose press
// release nobody went back and wrote. Quoting the template at a reader is worse
// than quoting the next intent down.
func (c *composer) pressReleaseIsUnwritten(rel string) bool {
	data, err := fsutil.ReadGuardedInRoot(c.root, rel, maxPageBytes)
	if err != nil {
		return false
	}
	body, consumed := StripFrontmatter(string(data))
	secs, err := Sections(rel, body, consumed)
	if err != nil {
		return false
	}
	for _, s := range secs {
		if s.Title != "Press Release" {
			continue
		}
		return intent.IsSeedNote(plainPressReleaseText(s.Body))
	}
	return false
}

// plainPressReleaseText reduces a press-release body to its words: quote
// markers, emphasis markers and whitespace runs removed. Nothing else is
// stripped, so any actual prose survives and fails the comparison.
func plainPressReleaseText(body string) string {
	var out []string
	for _, line := range strings.Split(body, "\n") {
		t := strings.TrimSpace(line)
		t = strings.TrimSpace(strings.TrimPrefix(t, ">"))
		t = strings.Map(func(r rune) rune {
			if r == '*' || r == '_' {
				return -1
			}
			return r
		}, t)
		if f := strings.Fields(t); len(f) > 0 {
			out = append(out, strings.Join(f, " "))
		}
	}
	return strings.Join(out, " ")
}

// auditIsMet reads an intent's `## Audit Notes` rollup: at least one criterion
// met, and none not met. The rollup line is the audit's own machine-readable
// summary (`internal/core/intent`), so this reads what the auditor wrote rather
// than re-grading anything.
//
// Two things it reads narrowly, and both matter, because between them they are
// the whole difference between quoting an intent the audit passed and quoting
// one it did not.
//
// It reads THE `## Audit Notes` section — the first one, and only where there
// is exactly one. An intent has one audit, so a second section of that title,
// at any heading level, is not a second verdict; it is evidence that the file is
// not the shape it claims. Accumulating across every matching section instead
// let a duplicate reading `MET 1` lift an honest concerns-only rollup, whose
// notMet is already zero, straight onto the homepage: the fenced-line failure
// below, reached by duplication rather than by fencing, and no harder to write
// (iss-2609090951277880). A duplicate is refused on the same ground a negative
// count is — a malformed record is not evidence that the criteria were met.
//
// It reads that section through the same fence-aware walk every other reader of
// these files uses, and only the prose of it. A rollup
// line elsewhere in the document — in the frontmatter, in another section, or
// quoted inside a fenced block as an example of the shape — is not a verdict
// about this intent, and a whole-file substring scan cannot tell the difference.
// A fenced `Acceptance rollup: MET 1` needs no negative to do damage: it lifts a
// concerns-only rollup, whose notMet is already 0, straight past the met > 0
// test.
//
// And it refuses a negative count outright. No audit writes one, `%d` accepts
// one, and one negative NOT_MET cancels a real one — so a rollup carrying a
// negative anywhere is malformed, and a malformed rollup is not evidence that
// the criteria were met.
func (c *composer) auditIsMet(rel string) bool {
	data, err := fsutil.ReadGuardedInRoot(c.root, rel, maxPageBytes)
	if err != nil {
		return false
	}
	body, consumed := StripFrontmatter(string(data))
	secs, err := Sections(rel, body, consumed)
	if err != nil {
		return false
	}
	notes, found := Section{}, false
	for _, s := range secs {
		if s.Title != "Audit Notes" {
			continue
		}
		if found {
			return false
		}
		notes, found = s, true
	}
	if !found {
		return false
	}
	met, notMet := 0, 0
	fence := false
	for _, line := range strings.Split(notes.Body, "\n") {
		if isFenceLine(line) {
			fence = !fence
			continue
		}
		if fence {
			continue
		}
		_, after, ok := strings.Cut(line, "Acceptance rollup:")
		if !ok {
			continue
		}
		for _, part := range strings.Split(after, "·") {
			fields := strings.Fields(part)
			if len(fields) != 2 {
				continue
			}
			n, err := strconv.Atoi(fields[1])
			if err != nil {
				continue
			}
			if n < 0 {
				return false
			}
			switch fields[0] {
			case "MET":
				met += n
			case "NOT_MET":
				notMet += n
			}
		}
	}
	return met > 0 && notMet == 0
}

// releaseOf finds the release whose changelog section credits a record id. The
// changelog is the record of what shipped when, and the intent id is how an
// entry says which promise it delivered, so the version is a lookup rather than
// a thing anyone types onto the page.
//
// A credit is the record's OWN handle, matched at a word boundary: `itd-1990`
// names a different record, not a longer spelling of `itd-199`. A substring
// test cannot tell the two apart, and because the walk takes the newest dated
// section first, the answer it gave a short handle was whichever longer handle
// happened to sit above it — `itd-1` stamped with the release that credits
// itd-130, `itd-9` with the one that credits itd-93. Worse than wrong once: a
// superstring landing in a FUTURE section restamps the featured record without
// anything about that record changing.
func (c *composer) releaseOf(id string) string {
	data, err := fsutil.ReadGuardedInRoot(c.root, "CHANGELOG.md", changelog.MaxChangelogBytes)
	if err != nil {
		return ""
	}
	version := ""
	want := normalizeHandle(id)
	for _, line := range strings.Split(string(data), "\n") {
		if changelog.IsDatedHeading(line) {
			if _, after, ok := strings.Cut(line, "["); ok {
				version, _, _ = strings.Cut(after, "]")
				version = strings.TrimPrefix(version, "v")
			}
			continue
		}
		if version != "" && creditsHandle(line, want) {
			return version
		}
	}
	return ""
}

// creditsHandle reports whether a changelog line names want — an already
// normalised handle — as a handle in its own right.
//
// The boundary comes from bodyHandleRe, this package's one definition of a
// record handle as it appears in prose: every handle on the line is read out
// and compared whole. bodyHandleRe knows the four record families the graph
// exports, and a family it does not know would find no credit at all — which
// fails closed, with the page carrying no version stamp, rather than open, with
// the page carrying somebody else's.
func creditsHandle(line, want string) bool {
	for _, m := range bodyHandleRe.FindAllString(line, -1) {
		if normalizeHandle(m) == want {
			return true
		}
	}
	return false
}

// normalizeHandle lower-cases a record handle and strips the leading zeros from
// its number, so `ITD-007` and `itd-7` are the same record — the same
// normalisation the record.json mentions pass applies to a handle it finds in
// prose. A value that is not a handle is returned lower-cased and otherwise
// untouched, and so matches only itself.
func normalizeHandle(h string) string {
	m := bodyHandleRe.FindStringSubmatch(h)
	if m == nil || m[0] != h {
		return strings.ToLower(h)
	}
	n := strings.TrimLeft(m[2], "0")
	if n == "" {
		n = "0"
	}
	return strings.ToLower(m[1]) + "-" + n
}
