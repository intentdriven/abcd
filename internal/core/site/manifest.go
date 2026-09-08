package site

// The composition manifest, `.abcd/site.json`.
//
// The manifest names WHERE every block of the site comes from and carries no
// prose of its own (adr-47 decision 2). It is decoded with unknown fields
// REFUSED, at every depth: a key the binary does not read is a composition
// instruction somebody wrote and the build silently ignored, which is the exact
// failure mode a manifest exists to prevent. The same strictness is what makes
// the file a contract rather than a suggestion.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// ManifestRelPath is the committed composition manifest.
const ManifestRelPath = ".abcd/site.json"

// maxManifestBytes bounds the manifest read. It is configuration, not content.
const maxManifestBytes = 256 * 1024

// pageRootPrefixes are the only directories a composed page may come from, the
// same closed set assetRootPrefix is for pictures and for the same reason: the
// build INLINES what it reads here into `index.html`, so a page path that is
// merely a valid repo-relative path reaches `.git/config` (a credential-bearing
// remote URL, and on a CI runner the checkout token in an `http.…extraheader`
// line), `.env`, and the gitignored `.abcd/.work.local/` tier — and publishes
// whichever it names. `docs/` is the documentation the landing page is composed
// from; `site-src/` is the site's own static input. Nothing else is prose the
// site was built to render (iss-2609081940475662).
var pageRootPrefixes = []string{"docs/", "site-src/"}

// ErrManifestInvalid is returned for a manifest the build cannot act on.
var ErrManifestInvalid = errors.New("site: manifest is invalid")

// The layouts the composer implements. A manifest naming any other layout is
// refused at load rather than at render, so the failure names the manifest.
const (
	LayoutCardsFromH2  = "cards-from-h2"
	LayoutLeadInCards  = "lead-in-cards"
	LayoutProse        = "prose"
	LayoutInstall      = "install"
	figureFirstImage   = "first-image"
	featureShippedPR   = "shipped-intent-press-release"
	featurePickNewest  = "newest-with-audit-MET"
	featurePartPR      = "press-release"
	featurePartFirstAC = "first-acceptance-criterion"
	// iconsBeforeLeadIn is the one icon rule the composer implements: an image
	// on its own line before a bold lead-in paragraph is that card's icon.
	iconsBeforeLeadIn = "image-before-lead-in"
	// tabsArrangement is the one tab arrangement the composer implements. The
	// value is a description rather than an enum, and it is compared exactly for
	// that reason: a manifest describing a DIFFERENT arrangement is asking for
	// something this build cannot do, and the honest answer is to say so rather
	// than to render the only arrangement there is and let the description rot.
	tabsArrangement = "left-h2s, then lead-h3s and remaining-h2s as a labelled group"
	// changelogFile is the release record the build reads. A manifest naming a
	// different `release.from` is pointing the site at a source nothing consults.
	changelogFile = "CHANGELOG.md"
)

// Manifest is `.abcd/site.json`.
//
// Every field here is either CONSUMED by this build or DEFERRED to a named
// later slice — and a deferred field is still validated, so a typo in it fails
// today rather than in three slices' time. Nothing is parsed and quietly
// dropped: a key the binary reads but never acts on is indistinguishable, to
// the person who wrote it, from a key that works.
type Manifest struct {
	SchemaVersion int `json:"schema_version"`
	// Purpose documents this file for the humans who edit it. It is the one
	// field with nothing to consume it, by design: it is never rendered, and
	// the single-source rule would forbid rendering it if it were.
	Purpose   string     `json:"purpose"`
	Identity  BlockRef   `json:"identity"`
	UIStrings string     `json:"ui_strings"`
	Home      Home       `json:"home"`
	Record    RecordOpts `json:"record"`
	// Docs names the documentation pages the site's docs surface renders.
	// DEFERRED: consumed by the docs-surface slice; validated here as paths.
	Docs DocsRefs `json:"docs"`
	// RecordPages carries the record explorer's selectors.
	// DEFERRED: consumed by spc-38's pages half; validated here as paths.
	RecordPages RecordPages `json:"record_pages"`
	// Checks declares which gates this repo arms.
	// DEFERRED: consumed by `abcd site check` (spc-37, spc-38). The build
	// MEASURES the unresolved references and publishes the count; the ratchet
	// that refuses a larger one is that verb's.
	Checks ManifestGate `json:"checks"`
}

// BlockRef selects a span of a file by heading.
type BlockRef struct {
	File    string `json:"file"`
	Heading string `json:"heading"`
}

// Home is the landing page's composition.
type Home struct {
	Hero     Hero      `json:"hero"`
	Chapters []Chapter `json:"chapters"`
}

// Hero names the page the headline, lede and figure come from.
type Hero struct {
	Page   string `json:"page"`
	Figure string `json:"figure"`
}

// Chapter is one lettered section of the landing page.
type Chapter struct {
	Letter  string   `json:"letter"`
	Page    string   `json:"page"`
	Layout  string   `json:"layout"`
	Feature *Feature `json:"feature,omitempty"`
	// Icons declares how a card gets its picture: an image on its own line
	// before a bold lead-in paragraph is that card's icon.
	Icons string `json:"icons,omitempty"`
	// TablePortraits names the chapter whose sections supply the portraits that
	// sit above this chapter's table column labels.
	TablePortraits string  `json:"table_portraits,omitempty"`
	Figure         *Figure `json:"figure,omitempty"`
	Lead           string  `json:"lead,omitempty"`
	// Release names where the chapter's release links read from. The build reads
	// `from` (the changelog) and links the released asset names, which the
	// committed install-surface agreement test holds to `assets`.
	Release *Release `json:"release,omitempty"`
	After   string   `json:"after,omitempty"`
	// Tabs describes the arrangement the install layout builds.
	Tabs string   `json:"tabs,omitempty"`
	Left []string `json:"left,omitempty"`
}

// Feature declares the quoted record block a chapter closes with.
type Feature struct {
	Kind  string   `json:"kind"`
	Pick  string   `json:"pick"`
	Parts []string `json:"parts"`
}

// Figure declares a chapter's lifted illustration.
type Figure struct {
	Kind string `json:"kind"`
	// LabelsFromPage asks that every label in the figure be a phrase on the page
	// it illustrates, so a diagram cannot drift from the prose beside it.
	// CONSUMED by `abcd site check`'s loop-figure gate (spc-37, ported from the
	// script's closing check). Validated here too — it is meaningless without a
	// figure to check — so a manifest that asks for it and names no figure is
	// refused at load rather than at render.
	LabelsFromPage bool `json:"labels-from-page,omitempty"`
}

// Release names where the install chapter reads version and asset names from.
type Release struct {
	From   string `json:"from"`
	Assets string `json:"assets"`
}

// RecordOpts carries the working-tier publication opt-in (adr-32, itd-140).
type RecordOpts struct {
	IssueLedger bool `json:"issue_ledger"`
}

// DocsRefs names the documentation pages the site links its docs surface to.
type DocsRefs struct {
	Index string `json:"index"`
	CLI   string `json:"cli"`
}

// RecordPages carries the explorer pages' selectors.
type RecordPages struct {
	Contributors Contributors `json:"contributors"`
}

// Contributors names the attribution policy span the contributors page quotes.
type Contributors struct {
	Policy PolicyRef `json:"policy"`
}

// PolicyRef selects a part of a span of a file by heading.
type PolicyRef struct {
	File    string `json:"file"`
	Heading string `json:"heading"`
	Part    string `json:"part"`
}

// ManifestGate is the check declaration. The gates themselves are a later
// slice; the manifest records which of them this repo arms.
type ManifestGate struct {
	EveryTextNodeHasSource        bool   `json:"every_text_node_has_source"`
	DocsLintOnRenderedText        bool   `json:"docs_lint_on_rendered_text"`
	CommandSnippetsPinnedToCLIRef bool   `json:"command_snippets_pinned_to_cli_reference"`
	UnresolvedReferenceBaseline   string `json:"unresolved_reference_baseline"`
}

// LoadManifest reads and validates the composition manifest at repoRoot.
func LoadManifest(repoRoot string) (Manifest, error) {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return Manifest{}, err
	}
	defer root.Close()
	data, err := fsutil.ReadGuardedInRoot(root, ManifestRelPath, maxManifestBytes)
	if err != nil {
		return Manifest{}, err
	}
	var m Manifest
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		return Manifest{}, fmt.Errorf("%w: %s: %v", ErrManifestInvalid, ManifestRelPath, err)
	}
	if err := m.validate(); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

// validate refuses a manifest the composer cannot act on. Every refusal names
// the key, because the reader's next move is an edit to this file.
func (m Manifest) validate() error {
	bad := func(format string, args ...any) error {
		return fmt.Errorf("%w: %s: %s", ErrManifestInvalid, ManifestRelPath, fmt.Sprintf(format, args...))
	}
	if m.SchemaVersion != 1 {
		return bad("schema_version is %d, want 1", m.SchemaVersion)
	}
	if m.Identity.File == "" || m.Identity.Heading == "" {
		return bad("identity.file and identity.heading are both required — the hero renders from the Identity block")
	}
	if !fsutil.ValidRelPath(m.Identity.File) {
		return bad("identity.file %q is not a repo-relative path", m.Identity.File)
	}
	// identity.file is a heading extract rather than a whole-file page source,
	// so it keeps its home in the durable record outside docs/ — but a heading
	// extract out of .git/config is still a quote out of .git/config.
	if err := refuseGitDir(bad, "identity.file", m.Identity.File); err != nil {
		return err
	}
	if m.UIStrings == "" || !fsutil.ValidRelPath(m.UIStrings) {
		return bad("ui_strings %q is not a repo-relative path", m.UIStrings)
	}
	if err := refuseGitDir(bad, "ui_strings", m.UIStrings); err != nil {
		return err
	}
	if m.Home.Hero.Page == "" || !fsutil.ValidRelPath(m.Home.Hero.Page) {
		return bad("home.hero.page %q is not a repo-relative path", m.Home.Hero.Page)
	}
	if err := refusePageSource(bad, "home.hero.page", m.Home.Hero.Page); err != nil {
		return err
	}
	if m.Home.Hero.Figure != "" && m.Home.Hero.Figure != figureFirstImage {
		return bad("home.hero.figure %q is not a figure rule (want %q)", m.Home.Hero.Figure, figureFirstImage)
	}
	if len(m.Home.Chapters) == 0 {
		return bad("home.chapters is empty — the landing page is composed of chapters")
	}
	letters := map[string]bool{}
	for i, ch := range m.Home.Chapters {
		where := fmt.Sprintf("home.chapters[%d]", i)
		if ch.Letter == "" {
			return bad("%s.letter is empty", where)
		}
		if letters[ch.Letter] {
			return bad("%s.letter %q is used twice — a chapter's letter is its identity", where, ch.Letter)
		}
		letters[ch.Letter] = true
		if !fsutil.ValidRelPath(ch.Page) {
			return bad("%s.page %q is not a repo-relative path", where, ch.Page)
		}
		if err := refusePageSource(bad, where+".page", ch.Page); err != nil {
			return err
		}
		switch ch.Layout {
		case LayoutCardsFromH2, LayoutLeadInCards, LayoutProse, LayoutInstall:
		default:
			return bad("%s.layout %q is not a layout the composer implements (%s)", where, ch.Layout,
				strings.Join([]string{LayoutCardsFromH2, LayoutLeadInCards, LayoutProse, LayoutInstall}, ", "))
		}
		if ch.Layout == LayoutInstall && ch.Lead == "" {
			return bad("%s.lead is required for the install layout — it names the section whose sub-headings become the CLI tabs", where)
		}
		if ch.Feature != nil {
			if ch.Feature.Kind != featureShippedPR {
				return bad("%s.feature.kind %q is not a feature the composer implements (want %q)", where, ch.Feature.Kind, featureShippedPR)
			}
			if ch.Feature.Pick != featurePickNewest {
				return bad("%s.feature.pick %q is not a pick rule the composer implements (want %q)", where, ch.Feature.Pick, featurePickNewest)
			}
			for _, p := range ch.Feature.Parts {
				if p != featurePartPR && p != featurePartFirstAC {
					return bad("%s.feature.parts names %q, which is not a quotable part (%s, %s)", where, p, featurePartPR, featurePartFirstAC)
				}
			}
		}
		if ch.Figure != nil {
			if ch.Figure.Kind != figureFirstImage {
				return bad("%s.figure.kind %q is not a figure rule (want %q)", where, ch.Figure.Kind, figureFirstImage)
			}
			// Only the prose layout lifts a figure out of its page. A figure
			// declared on any other layout is read and then dropped, which is the
			// failure this validation exists to prevent — including the deferred
			// `labels-from-page`, which would be asking `site check` to compare a
			// diagram nothing renders.
			if ch.Layout != LayoutProse {
				return bad("%s.figure is declared on the %q layout, which lifts no figure (only %q does)",
					where, ch.Layout, LayoutProse)
			}
		}
		if ch.Icons != "" && ch.Icons != iconsBeforeLeadIn {
			return bad("%s.icons %q is not an icon rule the composer implements (want %q)", where, ch.Icons, iconsBeforeLeadIn)
		}
		if ch.Tabs != "" && ch.Tabs != tabsArrangement {
			return bad("%s.tabs describes %q, but the composer builds one arrangement: %q", where, ch.Tabs, tabsArrangement)
		}
		if ch.Release != nil {
			if ch.Release.From != changelogFile {
				return bad("%s.release.from is %q, but the build reads releases from %q", where, ch.Release.From, changelogFile)
			}
			if ch.Release.Assets == "" {
				return bad("%s.release.assets is empty; it names the workflow whose published asset names the page links", where)
			}
		}
	}
	if err := m.validateDeferred(bad); err != nil {
		return err
	}
	return nil
}

// validateDeferred checks the keys this build does not act on yet.
//
// They are validated anyway, and that is the point of the rule: a manifest key
// nobody reads is indistinguishable, to the person who wrote it, from one that
// works. A path typo in `record_pages` would otherwise sit in the file being
// silently correct for however many slices it takes to reach its consumer, and
// then fail in a change that did not cause it.
func (m Manifest) validateDeferred(bad func(string, ...any) error) error {
	// DEFERRED to the docs-surface slice. Checked in a fixed order, because a
	// manifest with two bad paths must name the same one on every run — an error
	// message that changes between identical builds is a bug report nobody can
	// reproduce.
	for _, f := range []struct{ key, path string }{
		{"docs.index", m.Docs.Index},
		{"docs.cli", m.Docs.CLI},
	} {
		if f.path == "" {
			continue
		}
		if !fsutil.ValidRelPath(f.path) {
			return bad("%s %q is not a repo-relative path", f.key, f.path)
		}
		if err := refuseGitDir(bad, f.key, f.path); err != nil {
			return err
		}
	}
	// DEFERRED to spc-38's pages half (the contributors page).
	policy := m.RecordPages.Contributors.Policy
	if policy.File != "" {
		if !fsutil.ValidRelPath(policy.File) {
			return bad("record_pages.contributors.policy.file %q is not a repo-relative path", policy.File)
		}
		// A one-bullet quote, not a whole-file page source, so the attribution
		// policy stays quotable from CONTRIBUTING.md at the repository root —
		// but not from inside .git.
		if err := refuseGitDir(bad, "record_pages.contributors.policy.file", policy.File); err != nil {
			return err
		}
		if policy.Heading == "" {
			return bad("record_pages.contributors.policy.heading is empty; the page quotes a span selected by heading")
		}
	}
	// DEFERRED to `abcd site check`.
	if b := m.Checks.UnresolvedReferenceBaseline; b != "" {
		if !fsutil.ValidRelPath(b) {
			return bad("checks.unresolved_reference_baseline %q is not a repo-relative path", b)
		}
		if err := refuseGitDir(bad, "checks.unresolved_reference_baseline", b); err != nil {
			return err
		}
	}
	return nil
}

// refuseGitDir refuses a manifest path that names the git directory. EVERY path
// field passes it, deferred ones included: each names a file the build reads and
// renders some span of, and fsutil.ValidRelPath accepts ".git/config" because it
// is a clean, relative, in-root path. The refusal is fsutil.InsideGitDir, shared
// with the positioning registry's identical gate (iss-150) so the two cannot
// drift — a case-fold or a segment rule fixed in one is fixed in both.
func refuseGitDir(bad func(string, ...any) error, key, p string) error {
	if fsutil.InsideGitDir(p) {
		return bad("%s %q is inside .git, which is not a composition source", key, p)
	}
	return nil
}

// refusePageSource is the gate on the two WHOLE-FILE page sources —
// home.hero.page and home.chapters[].page. loadPage composes everything the
// named file holds into `index.html`, so the selection is closed to the page
// roots rather than merely denied the worst known destination: a denylist of
// .git and .env leaves the gitignored local tier, the private record, and every
// file a future contributor adds still reachable.
func refusePageSource(bad func(string, ...any) error, key, p string) error {
	if err := refuseGitDir(bad, key, p); err != nil {
		return err
	}
	for _, prefix := range pageRootPrefixes {
		if strings.HasPrefix(p, prefix) {
			return nil
		}
	}
	return bad("%s %q is outside %s — a composed page is inlined whole into the published site, so it comes from the documentation or the site's own source",
		key, p, strings.Join(pageRootPrefixes, " or "))
}

// FeatureWants reports whether a chapter's feature block quotes the named part.
func (f *Feature) FeatureWants(part string) bool {
	if f == nil {
		return false
	}
	for _, p := range f.Parts {
		if p == part {
			return true
		}
	}
	return false
}
