package site

// The record explorer's tests, over the same in-process fixture repository the
// landing page's golden files use.
//
// The fixture carries one of every shape the pages have to handle: a flat store
// graded in frontmatter and a store graded by directory, a discipline, a
// superseded record, an opted-in issue ledger, two frontmatter-free principles,
// a supersession whose target has left the tree, and a bibliography.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// explorerGoldens are the routes pinned byte for byte. One page of each kind,
// plus a record page from a store with frontmatter and one from the store
// without, because those two take different paths through the build.
var explorerGoldens = []struct{ route, golden string }{
	{"record/index.html", "record-dashboard.html"},
	{"record/graph/index.html", "record-graph.html"},
	{"record/foundations/index.html", "record-foundations.html"},
	{"contributors/index.html", "contributors.html"},
	{"references/index.html", "references.html"},
	{"record/adr/adr-1/index.html", "record-adr-1.html"},
	{"record/intent/itd-2/index.html", "record-itd-2.html"},
	{"record/principle/one-fixture-principle/index.html", "record-principle.html"},
}

// TestExplorerGolden pins every explorer page the fixture produces.
func TestExplorerGolden(t *testing.T) {
	f := newFixture(t)
	out := t.TempDir()
	res := buildFixture(t, f, out)

	for _, g := range explorerGoldens {
		if !containsString(res.Files, g.route) {
			t.Fatalf("build did not write %s: %v", g.route, res.Files)
		}
		data, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(g.route)))
		if err != nil {
			t.Fatal(err)
		}
		golden(t, g.golden, data)
	}
	// Every record in the export has a page, and nothing else does.
	pages := 0
	for _, f := range res.Files {
		if strings.HasSuffix(f, "index.html") {
			pages++
		}
	}
	if res.Pages+1 != pages {
		t.Errorf("the build reported %d explorer pages but wrote %d index files", res.Pages, pages)
	}
}

// TestExplorerCoversEveryRecord is the promise the explorer makes: a page for
// every record in the tree, reachable from the list twin.
func TestExplorerCoversEveryRecord(t *testing.T) {
	f := newFixture(t)
	out := t.TempDir()
	res := buildFixture(t, f, out)

	for _, id := range []string{"adr-1", "adr-2", "adr-3", "itd-1", "itd-2", "itd-3", "itd-4",
		"itd-5", "itd-6", "spc-1", "iss-1", "one-fixture-principle", "a-second-fixture-principle"} {
		found := false
		for _, file := range res.Files {
			if strings.Contains(file, "/"+id+"/index.html") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s has no page", id)
		}
	}

	graph := outFile(t, out, "record/graph/index.html")
	for _, want := range []string{
		`href="/record/adr/adr-1/"`,
		`href="/record/intent/itd-5/"`,
		`href="/record/principle/one-fixture-principle/"`,
		`href="/record/issue/iss-1/"`,
	} {
		if !strings.Contains(graph, want) {
			t.Errorf("the list twin does not reach %s", want)
		}
	}
	// The list twin carries the LINKS as well as the records, and each one is a
	// link, so a keyboard visitor traverses the graph without the chart running.
	implements := `implemented by <a href="/record/graph/?focus=spc-1">spc-1</a>`
	if !strings.Contains(graph, implements) {
		t.Errorf("the list twin does not carry the intent↔spec link as a link: want %q", implements)
	}
}

// TestRecordPageRendersItsBodyAndLinks is the per-record page's contract: the
// frontmatter, the body verbatim, the links phrased from this record's side, and
// the two forge links.
func TestRecordPageRendersItsBodyAndLinks(t *testing.T) {
	f := newFixture(t)
	out := t.TempDir()
	buildFixture(t, f, out)

	page := outFile(t, out, "record/intent/itd-2/index.html")
	for _, want := range []string{
		// the record's own body, verbatim
		"A fixture user opens the page and sees the record.",
		// frontmatter, and the dates git carries for the file
		`<th scope="row">id</th><td>itd-2</td>`,
		`<th scope="row">lifecycle</th><td>shipped</td>`,
		`<th scope="row">entered</th><td>2026-02-10</td>`,
		// the spec link, read from the intent's side
		`implemented by`,
		`href="/record/spec/spc-1/"`,
		// the forge links: the file, and its commit history
		`https://example.invalid/fixture/repo/blob/HEAD/.abcd/development/intents/shipped/itd-2-the-shipped-one.md`,
		`https://example.invalid/fixture/repo/commits/main/.abcd/development/intents/shipped/itd-2-the-shipped-one.md`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("itd-2's page is missing %q", want)
		}
	}

	// The spec's own page phrases the same link the other way round.
	spec := outFile(t, out, "record/spec/spc-1/index.html")
	if !strings.Contains(spec, `implements`) || !strings.Contains(spec, `href="/record/intent/itd-2/"`) {
		t.Error("the spec's page does not phrase its link from its own side")
	}
}

// TestBlockedByReadsBothWays pins a directed relation whose two ends are named
// by different words. `blocked_by` had no inverse, so the blocker's page said it
// was "blocked by" the record it is in fact blocking — the relation stated
// backwards, on the page of the record that is not blocked.
func TestBlockedByReadsBothWays(t *testing.T) {
	f := newFixture(t)
	out := t.TempDir()
	buildFixture(t, f, out)

	blocked := outFile(t, out, "record/intent/itd-1/index.html")
	if !strings.Contains(blocked, "blocked by") {
		t.Error("the blocked record's page does not say it is blocked by anything")
	}
	blocker := outFile(t, out, "record/intent/itd-2/index.html")
	if !strings.Contains(blocker, ">blocks<") {
		t.Error("the blocker's page does not say it blocks anything")
	}
	if strings.Contains(blocker, "blocked by") {
		t.Error("the blocker's page states the relation backwards")
	}
	// The chart is handed the same word rather than deriving its own.
	graph := outFile(t, out, "record/graph/index.html")
	if !strings.Contains(graph, `data-rel-blocked-by="blocks"`) {
		t.Error("the chart was not given the inverse of blocked_by")
	}
}

// TestListTwinLinksItsRelations is what makes the twin a twin: the chart's edges
// are traversable, so the twin's must be too. Naming a record in a span a
// keyboard visitor cannot follow reproduces the chart's picture and withholds
// its one interaction.
func TestListTwinLinksItsRelations(t *testing.T) {
	f := newFixture(t)
	out := t.TempDir()
	buildFixture(t, f, out)
	page := outFile(t, out, "record/graph/index.html")

	for _, want := range []string{
		`<a href="/record/graph/?focus=spc-1">spc-1</a>`,
		`<a href="/record/graph/?focus=itd-2">itd-2</a>`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the list twin does not link %q", want)
		}
	}
	// A target that has left the tree is named and dashed, and is not a link.
	if !strings.Contains(page, `<span class="stub">adr-9</span>`) {
		t.Error("the list twin does not mark the reference whose target has left the tree")
	}
	if strings.Contains(page, `?focus=adr-9"`) {
		t.Error("the list twin links a record that is not in the tree")
	}
}

// TestFrontmatterlessRecordsInventNoMetadata is the rule for a store that grades
// its records by nothing: whatever the page shows, the record said. A lifecycle
// word supplied here would be published as though it had been declared, would
// caption a dashboard tile, and would decide how the chart fills the bubble.
func TestFrontmatterlessRecordsInventNoMetadata(t *testing.T) {
	f := newFixture(t)
	out := t.TempDir()
	buildFixture(t, f, out)

	page := outFile(t, out, "record/principle/one-fixture-principle/index.html")
	if strings.Contains(page, "active") {
		t.Error("a principle's page carries a lifecycle the record never declared")
	}
	if strings.Contains(page, ">Frontmatter<") {
		t.Error("a frontmatter-less record's page renders a Frontmatter panel")
	}
	// What it does have is a file, and the two views of it the forge serves.
	if !strings.Contains(page, ".abcd/development/principles/one-fixture-principle.md") {
		t.Error("a principle's page does not name its file")
	}

	dash := outFile(t, out, "record/index.html")
	if strings.Contains(dash, "active") {
		t.Error("the dashboard captions a principle count with an invented lifecycle")
	}
	if !strings.Contains(dash, `<span class="l">principles</span>`) {
		t.Error("the principle tile lost its caption")
	}

	// And the export says the fields are derived, so no consumer has to infer it
	// from the store's name.
	var exp RecordExport
	if err := json.Unmarshal([]byte(outFile(t, out, "record.json")), &exp); err != nil {
		t.Fatal(err)
	}
	for _, n := range exp.Nodes {
		switch n.Type {
		case "principle":
			if !n.Derived {
				t.Errorf("%s declares no frontmatter and is not marked derived", n.ID)
			}
			if n.Lifecycle != "" || n.Status != "" {
				t.Errorf("%s carries a state it never declared: %q / %q", n.ID, n.Lifecycle, n.Status)
			}
		default:
			if n.Derived {
				t.Errorf("%s came from the frontmatter scan and is marked derived", n.ID)
			}
		}
	}
}

// TestRelativeNonMarkdownLinksResolve is the link shape the record writes that
// nothing rewrote: a directory or a file beside the record. On the landing page
// nothing links one; on a record's page at `/record/<type>/<id>/` it resolved
// against a directory three levels from anything, and 404ed.
func TestRelativeNonMarkdownLinksResolve(t *testing.T) {
	f := newFixture(t)
	out := t.TempDir()
	buildFixture(t, f, out)
	page := outFile(t, out, "record/principle/one-fixture-principle/index.html")

	// A path the tree carries points at the forge's own view of it.
	want := `<a href="https://example.invalid/fixture/repo/tree/main/.abcd/development/decisions">the decisions</a>`
	if !strings.Contains(page, want) {
		t.Errorf("a relative directory link was not resolved:\nwant %s", want)
	}
	// A path the tree does not carry keeps the record's own text: inventing a
	// forge URL for it trades a broken link for a confident 404.
	if !strings.Contains(page, `<a href="../nowhere">a missing thing</a>`) {
		t.Error("a relative link to a path the tree does not carry was rewritten anyway")
	}
}

// TestRecordPageRendersARetiredTargetAsAStub is the interview's ruling: a
// reference whose target has left the tree renders as the absence it is —
// dashed, labelled and UNLINKED — never as a link to a page that does not exist.
func TestRecordPageRendersARetiredTargetAsAStub(t *testing.T) {
	f := newFixture(t)
	out := t.TempDir()
	buildFixture(t, f, out)

	page := outFile(t, out, "record/adr/adr-2/index.html")
	if !strings.Contains(page, `<span class="id stub">adr-9</span>`) {
		t.Error("adr-2's dangling supersession does not render as a dashed stub")
	}
	if strings.Contains(page, `href="/record/adr/adr-9/"`) {
		t.Error("a reference to a record that is not in the tree rendered as a link")
	}
	// The dashboard says the same thing, measured against the ratchet.
	dash := outFile(t, out, "record/index.html")
	if !strings.Contains(dash, `<b class="stub">adr-9</b>`) {
		t.Error("record health does not name the unresolved reference")
	}
}

// TestTimelineIsDeterministicStaticSVG pins the genealogy as what it claims to
// be: one SVG emitted at build time, with a lane per store, links into the
// chart, and dashed stubs where a target has left the tree.
func TestTimelineIsDeterministicStaticSVG(t *testing.T) {
	f := newFixture(t)
	out := t.TempDir()
	buildFixture(t, f, out)
	// The genealogy renders inside the dashboard, folded; there is no page of
	// its own for it any more.
	page := outFile(t, out, "record/index.html")

	for _, want := range []string{
		`<svg viewBox="0 0 1000 `,
		// every mark opens its record in the relationship chart
		`href="/record/graph/?focus=adr-1"`,
		// releases tick to their own release on the forge
		`https://example.invalid/fixture/repo/releases/tag/v0.2.0`,
		// a supersession target that has left the tree is a dashed ×-stub
		`class="tlstub"`,
		`× adr-9`,
		// the issue ledger is a per-day histogram, not one dot per issue
		`class="tlbar"`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the genealogy is missing %q", want)
		}
	}
	// No arc may point at a record the tree does not carry.
	if strings.Contains(page, `Q `) && strings.Contains(page, `adr-9"`) {
		t.Error("an arc was drawn to a record that has no position")
	}
}

// TestDashboardVisualsCarryTheirNumbersAsText is the assistive-technology
// rule as amended (spc-38): every visual renders its labels and numbers as
// real text — a legend beside each bar, a version and a date on each cadence
// tick — and no separate table twin repeats them. Below 700px a plain list
// stands in for the cadence strip, so exactly one form renders at any width.
func TestDashboardVisualsCarryTheirNumbersAsText(t *testing.T) {
	f := newFixture(t)
	out := t.TempDir()
	buildFixture(t, f, out)
	page := outFile(t, out, "record/index.html")

	bars := strings.Count(page, `<div class="seg" role="presentation">`)
	legends := strings.Count(page, `<div class="legend">`)
	if bars == 0 {
		t.Fatal("the dashboard drew no lifecycle bars")
	}
	if legends != bars {
		t.Errorf("%d bars, %d legends — every bar carries its numbers as text", bars, legends)
	}
	if strings.Contains(page, `class="cadsvg"`) {
		for _, want := range []string{`>v0.1.0</text>`, `>2026-01-06</text>`, `>v0.2.0</text>`, `>2026-02-11</text>`} {
			if !strings.Contains(page, want) {
				t.Errorf("the cadence chart does not label %s as text", want)
			}
		}
		// Every tick keeps its version; a date is given up only where it would
		// print over the one before it on the same side of the axis.
		if v, d := strings.Count(page, `font-size="10"`), strings.Count(page, `font-size="8"`); d > v {
			t.Errorf("%d dates against %d versions — a date outran its tick", d, v)
		}
		if !strings.Contains(page, `class="list cadlist"`) {
			t.Error("the cadence panel has no narrow-screen list to stand in for the strip")
		}
	}
	if strings.Contains(page, `<details class="twin">`) {
		t.Error("a table twin remains beside a visual that already carries its numbers as text")
	}
}

// TestFoundationsListsAndLinks is the ruling on that page: it lists each
// principle and discipline as a card that LINKS its record, and never explains
// one — the explanation is the page the card opens.
func TestFoundationsListsAndLinks(t *testing.T) {
	f := newFixture(t)
	out := t.TempDir()
	buildFixture(t, f, out)
	page := outFile(t, out, "record/foundations/index.html")

	for _, want := range []string{
		`href="/record/principle/one-fixture-principle/"`,
		`href="/record/principle/a-second-fixture-principle/"`,
		`href="/record/intent/itd-5/"`,
		`One fixture principle`,
		`A Fixture Discipline`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the foundations page is missing %q", want)
		}
	}
	// The store's own index is not one of its records.
	if strings.Contains(page, "record/principle/README/") {
		t.Error("the principle store's README rendered as a principle")
	}
	// Lists and links, never explains: no card carries the body of its record.
	if strings.Contains(page, "the lint scan cannot see") {
		t.Error("a foundations card explained its record instead of linking it")
	}
}

// TestContributorsSeparatesAuthorshipFromDisclosure pins the page's whole point:
// humans are the authors of record, the trailer tallies are disclosure, and the
// policy that requires them is quoted beside the number.
func TestContributorsSeparatesAuthorshipFromDisclosure(t *testing.T) {
	f := newFixture(t)
	out := t.TempDir()
	buildFixture(t, f, out)
	page := outFile(t, out, "contributors/index.html")

	for _, want := range []string{
		"Authors of record",
		"Fixture",
		"Assisted-by trailers",
		"assistant-model-1",
		// the policy span the manifest selects, with its provenance and its file
		`data-src="CONTRIBUTING.md#attribution"`,
		// The manifest asks for the first BULLET. A section opens with its own
		// preamble more often than not, and quoting that instead leaves the rule
		// off the page under the number it was supposed to explain.
		//
		// The bullet's own bold lead-in survives intact: stripping the marker by
		// trimming a run of `-*+ ` would eat the opening `**` too and leave its
		// closing pair stranded as two visible asterisks.
		"<strong>Human author of record.</strong> The human contributor is the author of record",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the contributors page is missing %q", want)
		}
	}
	if strings.Contains(page, "A fixture preamble about disclosure") {
		t.Error("the policy panel quotes the section's lead-in instead of its first bullet")
	}
	// Model names are confined to this page under the attribution escape.
	for _, other := range []string{"record/index.html", "record/graph/index.html", "index.html"} {
		if strings.Contains(outFile(t, out, other), "assistant-model-1") {
			t.Errorf("%s names a model outside the attribution escape", other)
		}
	}
}

// TestContributorsRefuseWithoutTheirPolicy is the one place on the explorer
// where an absent source is a FAULT rather than a state: the manifest names the
// policy span, and publishing the assistance tallies without it leaves the
// number to be read as authorship — the reading the page exists to prevent.
func TestContributorsRefuseWithoutTheirPolicy(t *testing.T) {
	cases := []struct{ name, file, body string }{
		{"the declared file is gone", "", ""},
		{"the declared heading is gone", "CONTRIBUTING.md", "# Contributing\n\n## Something else\n\n- A bullet.\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newFixture(t)
			if c.file == "" {
				if err := os.Remove(filepath.Join(f.Root(), "CONTRIBUTING.md")); err != nil {
					t.Fatal(err)
				}
			} else {
				f.write(c.file, c.body)
			}
			_, err := Build(Request{RepoRoot: f.Root(), OutDir: t.TempDir(), Stamp: fixtureStamp})
			if err == nil {
				t.Fatal("the assistance tallies were published with no policy beside them")
			}
			if !strings.Contains(err.Error(), "record_pages.contributors.policy") {
				t.Errorf("the refusal does not name the declaration: %v", err)
			}
		})
	}
}

// TestEveryTableScrollsInsideItsOwnBox is the mobile rule stated where it can
// be checked: a table is the one block whose content the author does not choose
// the width of, so every one the build emits — composed, quoted or rendered
// verbatim out of a record — sits in its own overflow container.
//
// Without it the widest cell sets the page's width and the whole layout scrolls
// sideways at 390 px, which is a defect a reader meets on the first table and a
// reviewer never meets at all.
func TestEveryTableScrollsInsideItsOwnBox(t *testing.T) {
	f := newFixture(t)
	out := t.TempDir()
	res := buildFixture(t, f, out)

	tables := 0
	for _, name := range res.Files {
		if !strings.HasSuffix(name, ".html") {
			continue
		}
		page := outFile(t, out, name)
		for i := 0; ; {
			j := strings.Index(page[i:], "<table")
			if j < 0 {
				break
			}
			at := i + j
			tables++
			before := page[:at]
			if !strings.HasSuffix(before, `<div class="tablewrap">`) {
				t.Errorf("%s: a table at byte %d is not inside a .tablewrap — it will widen the page on a phone:\n… %s",
					name, at, clip120(page[max(0, at-90):at+40]))
			}
			i = at + 6
		}
	}
	if tables == 0 {
		t.Fatal("the fixture emitted no tables, so this proves nothing")
	}
}

// TestEveryAssetIsReachableFromEveryDepth is the rule a relative href quietly
// breaks: the same picture, stylesheet and script are linked from `/` and from
// `/record/adr/adr-1/`, so their addresses are root-absolute or they resolve on
// exactly one page and 404 on the other seven hundred.
func TestEveryAssetIsReachableFromEveryDepth(t *testing.T) {
	f := newFixture(t)
	out := t.TempDir()
	res := buildFixture(t, f, out)

	checked := 0
	for _, name := range res.Files {
		if !strings.HasSuffix(name, ".html") {
			continue
		}
		page := outFile(t, out, name)
		for i := 0; ; {
			j := strings.Index(page[i:], ` src="`)
			if j < 0 {
				break
			}
			at := i + j + len(` src="`)
			end := strings.IndexByte(page[at:], '"')
			if end < 0 {
				break
			}
			ref := page[at : at+end]
			i = at + end
			if ref == "" || strings.HasPrefix(ref, "/") ||
				strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") ||
				strings.HasPrefix(ref, "data:") {
				checked++
				continue
			}
			t.Errorf("%s: src=%q is relative — it resolves only at the depth of the page that wrote it", name, ref)
		}
	}
	if checked == 0 {
		t.Fatal("the fixture emitted no asset references, so this proves nothing")
	}
}

// TestBibliographyRefusesAnExecutableAddress is the same guard the markdown path
// applies to every repository-sourced link, at the one file that bypasses it.
// Escaping an attribute is no defence: a well-formed `javascript:` href needs no
// quote to break out of.
func TestBibliographyRefusesAnExecutableAddress(t *testing.T) {
	f := newFixture(t)
	f.write(".abcd/development/research/references.csl.json", `[
  {"id": "one", "type": "webpage", "author": [{"family": "Quill", "given": "Fenella"}],
   "title": "One", "issued": {"date-parts": [[2019]]},
   "URL": "javascript:alert(1)"}
]
`)
	_, err := Build(Request{RepoRoot: f.Root(), OutDir: t.TempDir(), Stamp: fixtureStamp})
	if err == nil {
		t.Fatal("a bibliography address that runs code was published as a link")
	}
	if !strings.Contains(err.Error(), "javascript:") {
		t.Errorf("the refusal does not name the scheme: %v", err)
	}
}

// headerBlock is one path pattern in `_headers` and the headers it sets.
type headerBlock struct {
	pattern string
	headers map[string]string
}

// parseHeaders reads the emitted `_headers` the way the host reads it: a line at
// the left margin opens a block, an indented `Name: value` sets a header on it,
// and a `#` line is a comment.
func parseHeaders(text string) []headerBlock {
	var out []headerBlock
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if line[0] != ' ' && line[0] != '\t' {
			out = append(out, headerBlock{pattern: trimmed, headers: map[string]string{}})
			continue
		}
		if len(out) == 0 {
			continue
		}
		name, value, ok := strings.Cut(trimmed, ":")
		if !ok {
			continue
		}
		out[len(out)-1].headers[strings.TrimSpace(name)] = strings.TrimSpace(value)
	}
	return out
}

// headerPathMatches is the host's glob: `*` stands for any run of characters.
func headerPathMatches(pattern, path string) bool {
	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return pattern == path
	}
	if !strings.HasPrefix(path, parts[0]) {
		return false
	}
	rest := path[len(parts[0]):]
	for i := 1; i < len(parts)-1; i++ {
		j := strings.Index(rest, parts[i])
		if j < 0 {
			return false
		}
		rest = rest[j+len(parts[i]):]
	}
	return strings.HasSuffix(rest, parts[len(parts)-1])
}

// The headers a route must carry, by what the host serves it as.
//
// A DOCUMENT is the only thing a content policy governs: the directives name
// what the page may load, and a stylesheet or an image loads nothing. So an
// asset is held to the two that do mean something for it — `nosniff`, so the
// declared content type is the one that is honoured, and a referrer policy, so
// a request for it carries no full URL off-origin.
var (
	documentHeaders = []string{"Content-Security-Policy", "X-Content-Type-Options", "Referrer-Policy"}
	assetHeaders    = []string{"X-Content-Type-Options", "Referrer-Policy"}
)

// servedRoute maps one emitted file to the path a reader requests and the
// headers that path must carry.
//
// Three files are not routes. `_headers` and `_redirects` are the host's own
// control files: it consumes them and serves neither. The build marker is this
// tool's own metadata — it exists so a rebuild may clear the directory, and it
// describes nothing but itself.
//
// They are excluded BY NAME, not by a rule like "skip dotfiles" or "skip files
// starting with an underscore". A pattern would quietly exempt every future file
// that happened to match it, and the whole job of this walk is to notice files
// nobody thought about.
func servedRoute(name string) (route string, required []string, served bool) {
	switch name {
	case "_headers", "_redirects", siteMarkerName:
		return "", nil, false
	}
	if strings.HasSuffix(name, "index.html") {
		// The served route is the directory the index sits in.
		return "/" + strings.TrimSuffix(name, "index.html"), documentHeaders, true
	}
	if strings.HasSuffix(name, ".html") {
		return "/" + name, documentHeaders, true
	}
	return "/" + name, assetHeaders, true
}

// assertHeaderCoverage holds one route to the headers its kind must carry.
func assertHeaderCoverage(t *testing.T, source string, blocks []headerBlock, route string, required []string) {
	t.Helper()

	found := map[string]bool{}
	for _, b := range blocks {
		if !headerPathMatches(b.pattern, route) {
			continue
		}
		for _, h := range required {
			if b.headers[h] != "" {
				found[h] = true
			}
		}
	}
	for _, h := range required {
		if !found[h] {
			t.Errorf("%s: %s matches no block setting %s", source, route, h)
		}
	}
}

// TestEveryRouteHasASecurityHeaderBlock is the rule a new route breaks by
// existing: `_headers` is a hand-written file and the routes are generated, so
// nothing but this connects the two.
//
// A route that matches no block is served with no content policy, no `nosniff`
// and no referrer policy — and it is served that way silently, because a missing
// header looks exactly like a page that works. The walk covers EVERY file the
// build emits, not the HTML alone: the stylesheet, the scripts, the record
// export, the installer and the copied rasters are all routes a reader can
// request, and an unprotected one is unprotected whatever its extension.
func TestEveryRouteHasASecurityHeaderBlock(t *testing.T) {
	f := newFixture(t)
	out := t.TempDir()
	res := buildFixture(t, f, out)

	blocks := parseHeaders(outFile(t, out, "_headers"))
	if len(blocks) == 0 {
		t.Fatal("the build emitted no _headers blocks")
	}

	docs, assets := 0, 0
	for _, name := range res.Files {
		route, required, served := servedRoute(name)
		if !served {
			continue
		}
		if len(required) == len(documentHeaders) {
			docs++
		} else {
			assets++
		}
		assertHeaderCoverage(t, "the emitted _headers", blocks, route, required)
	}
	// A walk that found nothing to walk proves nothing, and it proves nothing
	// twice over if it found only one of the two kinds: the asset rule is the one
	// this test exists to keep, and a run that saw no assets would pass while
	// enforcing it on nobody.
	if docs == 0 || assets == 0 {
		t.Fatalf("the build emitted %d document routes and %d asset routes; this proves nothing without both", docs, assets)
	}

	// The chart is the one route that fetches, and a policy that forbade it
	// would leave a blank stage with the failure only visible in a console.
	graph := false
	for _, b := range blocks {
		if headerPathMatches(b.pattern, "/record/graph/") &&
			strings.Contains(b.headers["Content-Security-Policy"], "connect-src 'self'") {
			graph = true
		}
	}
	if !graph {
		t.Error("/record/graph/ has no policy permitting the same-origin fetch of record.json")
	}
	// And the file this repository actually ships, held to the same rule. The
	// fixture proves the mechanism; this proves the committed policy covers the
	// route families the committed build emits — the asset routes included, which
	// the fixture cannot prove because its `_headers` is a copy of the shape
	// rather than the shipped policy.
	shipped, err := os.ReadFile(filepath.FromSlash("../../../site-src/headers"))
	if err != nil {
		t.Fatal(err)
	}
	shippedBlocks := parseHeaders(string(shipped))
	for _, route := range []string{"/", "/record/", "/record/graph/",
		"/record/foundations/", "/record/adr/adr-1/", "/record/principle/retire-the-name/",
		"/contributors/", "/references/"} {
		assertHeaderCoverage(t, "site-src/headers", shippedBlocks, route, documentHeaders)
	}
	// The asset routes are DERIVED from the build's own copy list, not restated
	// here: a source added to `copiedSources` then demands a block in the SHIPPED
	// policy, rather than only in the fixture's — which a developer can make green
	// by editing the fixture, leaving the file that is actually served short.
	//
	// The three the copy list cannot name are named: the rendered installer, the
	// record export, and one copied raster standing for the asset root the
	// manifest's own paths land under. Committed SVGs need no route — the build
	// inlines them as markup and copies none.
	assetRoutes := []string{"/" + installScriptName, "/record.json", "/assets/img/logo.png"}
	for _, cp := range copiedSources {
		if _, _, served := servedRoute(cp.dst); served {
			assetRoutes = append(assetRoutes, "/"+cp.dst)
		}
	}
	for _, route := range assetRoutes {
		assertHeaderCoverage(t, "site-src/headers", shippedBlocks, route, assetHeaders)
	}

	// Nothing anywhere may run inline script or eval.
	for _, b := range blocks {
		csp := b.headers["Content-Security-Policy"]
		if csp == "" {
			continue
		}
		script := ""
		for _, d := range strings.Split(csp, ";") {
			if strings.HasPrefix(strings.TrimSpace(d), "script-src") {
				script = d
			}
		}
		if strings.Contains(script, "unsafe-inline") || strings.Contains(script, "unsafe-eval") {
			t.Errorf("%s permits inline script: %q", b.pattern, script)
		}
	}
}

// TestReferencesRenderFromCSL pins the in-repo formatter: the sources render in
// the CSL file's own order, with a DOI or a URL linked, beside the credited
// inspirations under the acknowledgement file's own heading.
func TestReferencesRenderFromCSL(t *testing.T) {
	f := newFixture(t)
	out := t.TempDir()
	buildFixture(t, f, out)
	page := outFile(t, out, "references/index.html")

	for _, want := range []string{
		"References &amp; sources",
		"Fenella Quill. 2019. On the making of fixtures.",
		`<a href="https://doi.org/10.5555/fixture.2019.4">doi:10.5555/fixture.2019.4</a>`,
		"Orrin Tamsin and Iris Vole. 2021.",
		`<a href="https://example.invalid/rendering-the-record">`,
		"Inspirations",
		"A first invented inspiration",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the references page is missing %q", want)
		}
	}
	// The numbering is the citation key, so the order is the file's order.
	quill := strings.Index(page, "Fenella Quill")
	tamsin := strings.Index(page, "Orrin Tamsin")
	if quill < 0 || tamsin < 0 || quill > tamsin {
		t.Error("the sources are not rendered in the bibliography's own order")
	}
}

// TestNumberingDisagreementFailsTheBuild is the check the references page rests
// on. The record cites its sources BY NUMBER, so a page that numbered them
// differently would silently point every citation at the wrong source.
func TestNumberingDisagreementFailsTheBuild(t *testing.T) {
	cases := []struct {
		name, ack, says string
	}{
		{
			name: "a source is missing from the numbered list",
			ack: `# Acknowledgements

## References & sources

1. Fenella Quill. 2019. On the making of fixtures.
`,
			says: "entry for entry",
		},
		{
			name: "the two lists are in different orders",
			ack: `# Acknowledgements

## References & sources

1. Orrin Tamsin and Iris Vole. 2021. Rendering the record.
2. Fenella Quill. 2019. On the making of fixtures.
`,
			says: "would point at the wrong source",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newFixture(t)
			f.write("ACKNOWLEDGEMENTS.md", c.ack)
			_, err := Build(Request{RepoRoot: f.Root(), OutDir: t.TempDir(), Stamp: fixtureStamp})
			if err == nil {
				t.Fatal("a bibliography numbered differently from the acknowledgement list must fail the build")
			}
			if !strings.Contains(err.Error(), c.says) {
				t.Errorf("the refusal does not say %q: %v", c.says, err)
			}
			if !strings.Contains(err.Error(), "ACKNOWLEDGEMENTS.md") {
				t.Errorf("the refusal does not name the file: %v", err)
			}
		})
	}
}

// TestReferencesHeadingsComeFromTheFile is adr-47 decision 2 at the one page
// that had a fallback: the generator held the two heading names as literals and
// printed them when the acknowledgement file did not supply them.
//
// That is text written for the website, and worse, it is a title invented for
// somebody else's sources. Present and unmatched is a fault; absent is no page.
func TestReferencesHeadingsComeFromTheFile(t *testing.T) {
	t.Run("a present file that does not carry the heading is a fault", func(t *testing.T) {
		f := newFixture(t)
		f.write("ACKNOWLEDGEMENTS.md", "# Acknowledgements\n\n## Something else\n\nNot the heading.\n")
		_, err := Build(Request{RepoRoot: f.Root(), OutDir: t.TempDir(), Stamp: fixtureStamp})
		if err == nil {
			t.Fatal("the sources were published under a heading the repository never wrote")
		}
		for _, want := range []string{"ACKNOWLEDGEMENTS.md", "References & sources"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("the refusal does not name %q: %v", want, err)
			}
		}
	})

	t.Run("the heading is rendered as the file spells it", func(t *testing.T) {
		f := newFixture(t)
		ack := outFile(t, f.Root(), "ACKNOWLEDGEMENTS.md")
		f.write("ACKNOWLEDGEMENTS.md", strings.Replace(ack,
			"## References & sources", "## References & Sources", 1))
		out := t.TempDir()
		if _, err := Build(Request{RepoRoot: f.Root(), OutDir: out, Stamp: fixtureStamp}); err != nil {
			t.Fatal(err)
		}
		page := outFile(t, out, "references/index.html")
		if !strings.Contains(page, "References &amp; Sources") {
			t.Error("the page does not use the file's own spelling of the heading")
		}
	})

	t.Run("no acknowledgement file omits the page", func(t *testing.T) {
		f := newFixture(t)
		if err := os.Remove(filepath.Join(f.Root(), "ACKNOWLEDGEMENTS.md")); err != nil {
			t.Fatal(err)
		}
		out := t.TempDir()
		res, err := Build(Request{RepoRoot: f.Root(), OutDir: out, Stamp: fixtureStamp})
		if err != nil {
			t.Fatalf("a repository with no acknowledgement file must still build: %v", err)
		}
		if containsString(res.Files, "references/index.html") {
			t.Error("the references page rendered with no heading to title it")
		}
		if strings.Contains(outFile(t, out, "record/index.html"), `href="/references/"`) {
			t.Error("the navigation points at a references page the build did not write")
		}
	})
}

// --- graceful absence -----------------------------------------------------
//
// itd-140's rule, three times over: a missing optional source omits the page it
// feeds AND that page's navigation entry, and the build succeeds.

// TestBuildWithoutBibliography omits `/references/` rather than shipping a
// broken or half-rendered page.
func TestBuildWithoutBibliography(t *testing.T) {
	f := newFixture(t)
	if err := os.Remove(filepath.Join(f.Root(),
		filepath.FromSlash(".abcd/development/research/references.csl.json"))); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	res, err := Build(Request{RepoRoot: f.Root(), OutDir: out, Stamp: fixtureStamp})
	if err != nil {
		t.Fatalf("a repository with no bibliography must still build: %v", err)
	}
	if containsString(res.Files, "references/index.html") {
		t.Error("the references page rendered with no bibliography to render")
	}
	dash := outFile(t, out, "record/index.html")
	if strings.Contains(dash, `href="/references/"`) {
		t.Error("the navigation points at a references page the build did not write")
	}
	if !strings.Contains(dash, `href="/record/graph/"`) {
		t.Error("the rest of the navigation went with it")
	}
}

// TestBuildWithoutFoundations omits the foundations page when the repository
// declares neither principles nor disciplines.
func TestBuildWithoutFoundations(t *testing.T) {
	f := newFixture(t)
	for _, dir := range []string{
		".abcd/development/principles",
		".abcd/development/intents/disciplines",
	} {
		if err := os.RemoveAll(filepath.Join(f.Root(), filepath.FromSlash(dir))); err != nil {
			t.Fatal(err)
		}
	}
	out := t.TempDir()
	res, err := Build(Request{RepoRoot: f.Root(), OutDir: out, Stamp: fixtureStamp})
	if err != nil {
		t.Fatalf("a repository with no principles or disciplines must still build: %v", err)
	}
	if containsString(res.Files, "record/foundations/index.html") {
		t.Error("the foundations page rendered with nothing to found on")
	}
	dash := outFile(t, out, "record/index.html")
	if strings.Contains(dash, `href="/record/foundations/"`) {
		t.Error("the navigation points at a foundations page the build did not write")
	}
	if strings.Contains(dash, `principles`) {
		t.Error("the dashboard captioned a principle tile with no principles")
	}
}

// TestExplorerWithoutChangelog drops the release lane, the cadence panel and the
// release tiles, and keeps every page that does not depend on them.
func TestExplorerWithoutChangelog(t *testing.T) {
	f := newFixture(t)
	if err := os.Remove(filepath.Join(f.Root(), "CHANGELOG.md")); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if _, err := Build(Request{RepoRoot: f.Root(), OutDir: out,
		Stamp: BuildStamp{Commit: "abcdef1", GeneratedAt: "2026-02-11"}}); err != nil {
		t.Fatalf("a repository with no changelog must still build its explorer: %v", err)
	}
	dash := outFile(t, out, "record/index.html")
	if !strings.Contains(dash, `class="tlsvg"`) {
		t.Fatal("the genealogy vanished with the changelog")
	}
	tl := dash
	if strings.Contains(tl, "releases/tag/") {
		t.Error("the genealogy drew a release lane with no releases")
	}
	if !strings.Contains(tl, `href="/record/graph/?focus=adr-1"`) {
		t.Error("the rest of the genealogy went with the release lane")
	}
}

// TestIssueLedgerIsOptIn is adr-32's tiering at the site: the working-tier
// ledger is published only because this repository asks for it, and a
// repository that does not ask gets no ledger pages at all.
func TestIssueLedgerIsOptIn(t *testing.T) {
	f := newFixture(t)
	manifest := outFile(t, f.Root(), ".abcd/site.json")
	f.write(".abcd/site.json", strings.Replace(manifest,
		`"record": {"issue_ledger": true}`, `"record": {"issue_ledger": false}`, 1))
	out := t.TempDir()
	res, err := Build(Request{RepoRoot: f.Root(), OutDir: out, Stamp: fixtureStamp})
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range res.Files {
		if strings.Contains(file, "record/issue/") {
			t.Errorf("the ledger was published without the opt-in: %s", file)
		}
	}
	if strings.Contains(outFile(t, out, "record/index.html"), "iss-1") {
		t.Error("the dashboard names a ledger entry the repository did not opt in to publishing")
	}
}

// outFile reads one file out of a build tree.
func outFile(t *testing.T, out, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
