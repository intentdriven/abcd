package site

// The project route (itd-2610091010348892): a project site built from nothing
// but what abcd already writes to the repository.
//
// The product thinker's ruling of 2026-10-15, verbatim: "The website only
// renders what abcd already writes -- SSOT." A project's hand-written
// documentation is read on GitHub, not rendered. So the project route has no
// composition: `.abcd/site.json` holds the project site's SETTINGS (the hosting
// block and its switch, `"mode": "project"`), and nothing in it says where a
// word comes from. The pages are derived in code:
//
//   - the about page (`/`), from the identity block, the brief's opening, the
//     newest shipped intents' press releases and the release list in
//     CHANGELOG.md, each rendered only where the repository holds it;
//   - the record explorer, exactly as the composed route renders it, with the
//     two pages that quote hand-written files left out (the references page
//     reads ACKNOWLEDGEMENTS.md, the contributors policy a contribution guide);
//   - the interface words, the stylesheet, the scripts and the headers policy,
//     from the binary (setupsrc/), never from files in the repository.
//
// The composed route (a `.abcd/site.json` with no mode: abcd's own site) is
// untouched by any of this. `abcd site build --project` and `abcd lint site
// --project` name the route, and each refuses a settings file of the other
// shape, so neither route is ever entered by inference.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/adapter/hosting"
	"github.com/intentdriven/abcd/internal/core/changelog"
	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/core/positioning"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// ModeProject is the settings file's mode for a project site.
const ModeProject = "project"

// Project reports whether the file declares a project site's settings rather
// than a composition.
func (m Manifest) Project() bool { return m.Mode == ModeProject }

// projectSettingsPurpose documents a project site's settings file for the
// people who read it. Like a composition's purpose, it is never rendered.
const projectSettingsPurpose = "This repository's project-site settings, written by `abcd site setup`. " +
	"The site renders only what abcd already writes to the repository (the identity block, the brief, " +
	"the development record and CHANGELOG.md), so this file holds settings and no content: `hosting` " +
	"names where the site is published, and `hosting.publish` is the switch."

// projectSettings is the settings file as setup writes it, in this key order.
type projectSettings struct {
	SchemaVersion int      `json:"schema_version"`
	Mode          string   `json:"mode"`
	Purpose       string   `json:"purpose"`
	Hosting       *Hosting `json:"hosting"`
}

// encodeProjectSettings renders a project site's settings file.
func encodeProjectSettings(h *Hosting) ([]byte, error) {
	raw, err := json.MarshalIndent(projectSettings{SchemaVersion: 1, Mode: ModeProject, Purpose: projectSettingsPurpose, Hosting: h}, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

// validateProject refuses a settings file that carries anything but settings.
// A composition key here would be a sentence's source written for the website,
// which is the thing the project route exists not to have.
func (m Manifest) validateProject(bad func(string, ...any) error) error {
	for _, f := range []struct {
		key string
		set bool
	}{
		{"identity", m.Identity != (BlockRef{})},
		{"ui_strings", m.UIStrings != ""},
		{"home", m.Home.Hero != (Hero{}) || len(m.Home.Chapters) > 0},
		{"docs", m.Docs != (DocsRefs{})},
		{"record_pages", m.RecordPages != (RecordPages{})},
		{"checks", m.Checks != (ManifestGate{})},
	} {
		if f.set {
			return bad("%s is a composition key, and a project site's settings carry none: its pages are built from what abcd writes to the repository", f.key)
		}
	}
	if m.Hosting == nil {
		return bad("a project site's settings need a hosting block: it names the provider and holds the switch")
	}
	if err := m.Hosting.validate(bad); err != nil {
		return err
	}
	if a, _ := adapterNamed(m.Hosting.Provider); !a.Profile(hosting.Site{}).Project {
		return bad("hosting.provider %q publishes a composed site, not a project site", m.Hosting.Provider)
	}
	return m.Pages.validate(bad)
}

// abcdWrittenSource reports whether rel is a file abcd itself writes to a
// repository, and so a file a project site may render a span of:
//
//   - `.abcd/**`, the record abcd keeps: the brief, intents, specs, ADRs and
//     principles under `.abcd/development/`, the issue ledger under
//     `.abcd/work/issues/`, and the identity block `abcd identity init` writes
//     to `.abcd/development/IDENTITY.md` (positioning.DefaultBlockLocation).
//     The gitignored local tier, `.abcd/.work.local/`, is no one's published
//     record and is refused.
//   - `CHANGELOG.md`, which abcd derives: the launch scaffold seeds it
//     (scaffold.ChangelogAnchor), and `abcd launch ship` writes each dated
//     section from the records that shipped (release.Ingest).
//
// Everything else, `docs/` and `README.md` above all, is a person's writing,
// read on the forge where it lives.
func abcdWrittenSource(rel string) bool {
	if rel == changelogFile {
		return true
	}
	return strings.HasPrefix(rel, ".abcd/") && !strings.HasPrefix(rel, ".abcd/.work.local/")
}

// routeAgrees refuses a build or a gate whose --project flag and settings file
// name different routes: the route is declared, never inferred.
func routeAgrees(m Manifest, project bool) error {
	switch {
	case m.Project() && !project:
		return fmt.Errorf("site: %s holds a project site's settings, built from what abcd writes; pass --project", ManifestRelPath)
	case !m.Project() && project:
		return fmt.Errorf("site: --project builds a project site, and %s is a composition; build it without --project", ManifestRelPath)
	}
	return nil
}

// projectUI is the interface-string allowlist the binary carries: a project
// site's interface words come from abcd, never from a file in the repository.
func projectUI() (UI, error) {
	data, err := setupSources.ReadFile("setupsrc/ui.json")
	if err != nil {
		return UI{}, err
	}
	return decodeUI(data, "the binary's ui.json")
}

// projectCopies are the static files a project site is served with, from the
// binary: the headers policy, the stylesheet and the two scripts. There is no
// redirect map (that is abcd's own URL history) and no installer.
var projectCopies = []struct{ src, dst string }{
	{"headers", "_headers"},
	{"site.css", "site.css"},
	{"site.js", "site.js"},
	{"record.js", "record.js"},
}

// BriefOpeningRelPath is where the brief opens: the first section of abcd's
// brief skeleton, `product/press-release`, laid out at
// `.abcd/development/brief/01-product/01-press-release.md` (the brief↔lifeboat
// contract, internal/core/lifeboat/mapping.go, and the native layout every
// managed repository's brief follows, lifeboat's nativeBriefFilePath).
const BriefOpeningRelPath = ".abcd/development/brief/01-product/01-press-release.md"

// aboutShippedIntents is how many shipped intents the about page quotes.
const aboutShippedIntents = 3

// ErrProjectEmpty refuses a project site with nothing to build its about page
// from.
var ErrProjectEmpty = errors.New("site setup: this repository holds nothing abcd writes that a project site is built from yet: " +
	"no identity block (`abcd identity init`), no brief opening (" + BriefOpeningRelPath + "), no shipped intent whose " +
	"audit was met, and no release in CHANGELOG.md; the site would be empty, so it is not switched on")

// aboutSources is what the about page is built from. Each part is set only
// where the repository holds it: anything absent is left out, never filled.
type aboutSources struct {
	identity bool
	brief    *docPage
	opening  int
	intents  []lint.RecordNode
	releases []changelog.DatedRelease
}

// empty reports whether the about page would have nothing on it.
func (a aboutSources) empty() bool {
	return !a.identity && a.brief == nil && len(a.intents) == 0 && len(a.releases) == 0
}

// projectIdentity resolves the identity block a project site renders: the one
// `.abcd/positioning.json` points at, when abcd wrote the file it lives in.
// One pointed at a person's own file (a README, a docs page) is absent here.
func projectIdentity(root string) (BlockRef, error) {
	cfg, ok, err := positioning.LoadConfig(root)
	if err != nil {
		return BlockRef{}, fmt.Errorf("site: the identity pointer cannot be read: %w", err)
	}
	if !ok || !fsutil.ValidRelPath(cfg.Block.File) || !abcdWrittenSource(cfg.Block.File) {
		return BlockRef{}, nil
	}
	return BlockRef{File: cfg.Block.File, Heading: cfg.Block.Heading}, nil
}

// aboutSources finds what the about page is built from.
func (c *composer) aboutSources() (aboutSources, error) {
	var a aboutSources
	_, a.identity = c.identity()
	p, i, err := c.briefOpening()
	if err != nil {
		return aboutSources{}, err
	}
	a.brief, a.opening = p, i
	if met := c.metIntents(); len(met) > aboutShippedIntents {
		a.intents = met[:aboutShippedIntents]
	} else {
		a.intents = met
	}
	releases, _, err := changelog.DatedReleases(c.repoRoot)
	if err != nil {
		return aboutSources{}, err
	}
	a.releases = releases
	return a, nil
}

// briefOpening reads the brief's opening: its first section with anything in
// it. A brief that does not open there, or opens on nothing, is absent.
func (c *composer) briefOpening() (*docPage, int, error) {
	data, err := fsutil.ReadGuardedInRoot(c.root, BriefOpeningRelPath, maxPageBytes)
	if os.IsNotExist(err) {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, fmt.Errorf("site: reading the brief's opening %s: %w", BriefOpeningRelPath, err)
	}
	body, consumed := StripFrontmatter(string(data))
	secs, err := Sections(BriefOpeningRelPath, body, consumed)
	if err != nil {
		return nil, 0, err
	}
	for i, s := range secs {
		if len(Blocks(s.Body, s.BodyLine)) > 0 {
			return &docPage{Rel: BriefOpeningRelPath, Dir: path.Dir(BriefOpeningRelPath), Sections: secs}, i, nil
		}
	}
	return nil, 0, nil
}

// projectHasContent reports whether the repository at root holds anything a
// project site's about page is built from. Setup asks it before switching a
// project site on.
func projectHasContent(root string) (bool, error) {
	srcRoot, err := os.OpenRoot(root)
	if err != nil {
		return false, err
	}
	defer srcRoot.Close()
	id, err := projectIdentity(root)
	if err != nil {
		return false, err
	}
	cfg, err := lint.LoadConfigInRoot(srcRoot, ".abcd/record-lint.json")
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	graph, err := lint.LoadRecordGraph(cfg, root)
	if err != nil {
		return false, err
	}
	c := &composer{repoRoot: root, root: srcRoot, identityLoc: id, graph: graph}
	a, err := c.aboutSources()
	if err != nil {
		return false, err
	}
	return !a.empty(), nil
}

// ComposeAbout renders a project site's about page: the identity block as the
// hero, then the brief's opening, the newest shipped intents' press releases
// and the release list, each only where the repository holds it.
func (c *composer) ComposeAbout() (string, error) {
	a, err := c.aboutSources()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString(c.head())
	b.WriteString(c.header())
	b.WriteString(`<main id="app"><div class="page">`)
	if a.identity {
		b.WriteString(c.aboutHero())
	}
	if a.brief != nil {
		h, err := c.aboutBrief(a.brief, a.opening)
		if err != nil {
			return "", err
		}
		b.WriteString(h)
	}
	if len(a.intents) > 0 {
		var quotes strings.Builder
		for _, n := range a.intents {
			h, err := c.featureBlockFor(n, &Feature{Kind: featureShippedPR, Pick: featurePickNewest, Parts: []string{featurePartPR}})
			if err != nil {
				return "", err
			}
			quotes.WriteString(h)
		}
		if quotes.Len() > 0 {
			b.WriteString(aboutSection("shipped", `<p class="eyebrow">`+escapeText(c.ui.Tiles.Intent)+`</p>`, quotes.String()))
		}
	}
	if len(a.releases) > 0 {
		h, err := c.aboutReleases(a.releases)
		if err != nil {
			return "", err
		}
		b.WriteString(h)
	}
	b.WriteString("</div></main>")
	b.WriteString(c.footer())
	b.WriteString("</body>\n</html>\n")
	return b.String(), nil
}

// aboutSection wraps one part of the about page. The composed route's chapter
// head sets a letter beside its heading, and a project site has no letters, so
// the heading sits in the column on its own.
func aboutSection(id, head, body string) string {
	return `<section class="chapter" id="` + escapeAttr(id) + `"><div class="wrap">` + head +
		`<div class="stack" style="--stack:22px;margin-top:28px">` + body + `</div></div></section>`
}

// aboutHero renders the identity block: its title, its tagline as the page's
// heading and its pitch beside them.
func (c *composer) aboutHero() string {
	blk, _ := c.identity()
	src := c.identitySrc()
	var b strings.Builder
	b.WriteString(`<section class="hero"><div class="wrap"><div class="grid"><div>`)
	b.WriteString(`<p class="eyebrow"` + src + `>` + escapeText(blk.Title) + `</p>`)
	b.WriteString(`<h1 class="tagline"` + src + `>` + escapeText(blk.Tagline) + `</h1>`)
	b.WriteString(`</div>`)
	if blk.Pitch != "" {
		b.WriteString(`<div class="hero-aside"><p class="pitch"` + src + `>` + escapeText(blk.Pitch) + `</p></div>`)
	}
	b.WriteString(`</div></div></section>`)
	return b.String()
}

// aboutBrief renders the brief's opening section, verbatim.
func (c *composer) aboutBrief(p *docPage, i int) (string, error) {
	s := p.Sections[i]
	body, err := c.renderer(p).RenderBlocks(p.Rel, Blocks(s.Body, s.BodyLine))
	if err != nil {
		return "", err
	}
	head := ""
	if s.Title != "" {
		head = `<h2` + srcAttr(p.Rel, s.Anchor) + `>` + escapeText(s.Title) + `</h2>`
	}
	id := s.Anchor
	if id == "" {
		id = "brief"
	}
	return aboutSection(id, head, `<div class="prose"`+srcAttr(p.Rel, s.Anchor)+`>`+body+`</div>`), nil
}

// aboutReleases renders the release list: each dated release in CHANGELOG.md,
// newest first, with the date it carries, linked to its release on the forge.
func (c *composer) aboutReleases(releases []changelog.DatedRelease) (string, error) {
	anchors := map[string]string{}
	data, err := fsutil.ReadGuardedInRoot(c.root, changelogFile, changelog.MaxChangelogBytes)
	if err != nil {
		return "", err
	}
	body, consumed := StripFrontmatter(string(data))
	secs, err := Sections(changelogFile, body, consumed)
	if err != nil {
		return "", err
	}
	for _, s := range secs {
		if v, _, ok := strings.Cut(strings.TrimPrefix(s.Title, "["), "]"); ok && strings.HasPrefix(s.Title, "[") {
			if _, seen := anchors[v]; !seen {
				anchors[v] = s.Anchor
			}
		}
	}
	var items strings.Builder
	items.WriteString(`<ul class="stack" style="--stack:6px;list-style:none;padding:0">`)
	for _, r := range releases {
		version := escapeText("v" + strings.TrimPrefix(r.Version, "v"))
		if c.repo.Repository != "" {
			version = `<a href="` + escapeAttr(c.repo.Repository+"/releases/tag/v"+strings.TrimPrefix(r.Version, "v")) + `">` + version + `</a>`
		}
		line := version
		if r.Date != "" {
			line += ` · ` + escapeText(r.Date)
		}
		items.WriteString(`<li` + srcAttr(changelogFile, anchors[r.Version]) + `>` + line + `</li>`)
	}
	items.WriteString(`</ul>`)
	if c.repo.Repository != "" {
		rr := c.repo.Repository
		items.WriteString(`<p class="small muted"><a href="` + escapeAttr(rr+"/releases") + `">` + escapeText(c.ui.AllReleases) + `</a> · ` +
			`<a href="` + escapeAttr(forgeBlob(rr, changelogFile)) + `">` + escapeText(changelogFile) + `</a></p>`)
	}
	return aboutSection("releases", `<p class="eyebrow">`+escapeText(c.ui.Tiles.Releases)+`</p>`, items.String()), nil
}

// metIntents are the shipped intents the record shows delivered: their audit
// was met and their press release was written, most recently shipped first,
// id descending as the tie-break (newestMetIntent's rule, for every intent).
func (c *composer) metIntents() []lint.RecordNode {
	type picked struct {
		node lint.RecordNode
		date string
	}
	var out []picked
	for _, n := range c.graph.Nodes {
		if n.Type != "intent" || n.Lifecycle != "shipped" {
			continue
		}
		if !c.auditIsMet(n.Path) || c.pressReleaseIsUnwritten(n.Path) {
			continue
		}
		out = append(out, picked{n, c.history.EnteredBucket(n.Path)})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].date != out[j].date {
			return out[i].date > out[j].date
		}
		return handleNum(out[i].node.ID) > handleNum(out[j].node.ID)
	})
	nodes := make([]lint.RecordNode, len(out))
	for i, p := range out {
		nodes[i] = p.node
	}
	return nodes
}

// checkProjectSources is the project route's half of the provenance gate:
// every span on every page, the record's own pages included, names a file abcd
// writes. A span from `docs/` or `README.md` is a person's writing, which a
// project site does not render (the product thinker's ruling of 2026-10-15).
func (c *checker) checkProjectSources() {
	for _, page := range c.res.Pages {
		seen := map[string]bool{}
		for _, el := range findElements(c.pages[page], func(n *htmlNode) bool { return n.Attr("data-src") != "" }) {
			src := el.Attr("data-src")
			rel, _, _ := strings.Cut(src, "#")
			if seen[rel] || abcdWrittenSource(rel) {
				continue
			}
			seen[rel] = true
			c.fail(CheckProvenance, page, src, "a span comes from %s, which abcd does not write; a project site renders only what abcd writes "+
				"(.abcd/** and CHANGELOG.md), and a person's own files are read on the forge", quote(rel))
		}
	}
}
