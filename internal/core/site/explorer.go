package site

// The record explorer — every page under `/record/`, plus `/contributors/` and
// `/references/`.
//
// Ported from the clickable prototype in the investigation cluster, which is the
// behavioural spec: the sub-navigation, the dashboard's panels, the chart's
// stage and controls, the genealogy and the two attribution pages are its
// markup, with the hash router's routes replaced by real directories.
//
// Every page here is GENERIC-SIDE (itd-140): its inputs are the record format,
// git history and `CHANGELOG.md` — plus, for `/references/`, the CSL
// bibliography and `ACKNOWLEDGEMENTS.md`. An absent optional input omits the
// page it feeds AND that page's navigation entry, and the build succeeds.
//
// Every visible word is a count, a date, an id, a title, a file name, a span of
// a record file carrying `data-src`, or an interface label from
// `site-src/ui.json` (adr-47 decision 2). The verbatim record rendering under
// `/record/**` is the one part of the site exempt from the banned-token gate
// (adr-47 decision 3): the record legitimately contains change-narration, and
// rewriting it to look better on the site is what that decision forbids.

import (
	"fmt"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// The explorer's routes. They are directories with an index.html, so the served
// URL is the route itself.
const (
	routeDashboard    = "record/"
	routeGraph        = "record/graph/"
	routeTimeline     = "record/timeline/"
	routeFoundations  = "record/foundations/"
	routeDevelopment  = "record/development/"
	routeHealth       = "record/health/"
	routeContributors = "contributors/"
	routeReferences   = "references/"
)

// lifecycleRank orders a store's buckets from unstarted to retired, so a
// lifecycle bar reads left to right as the work moves and takes the same colour
// for the same meaning in every store.
var lifecycleRank = map[string]int{
	"drafts": 0, "proposed": 0, "open": 0,
	"planned": 1,
	// A discipline is a rule that holds rather than a change that ships, so it
	// sits outside the ramp entirely.
	disciplinesLifecycle: 2,
	"shipped":            3, "accepted": 3, "closed": 3, "resolved": 3, "active": 3,
	"superseded": 4, "wontfix": 4,
}

// lifecycleRamp is the colour each rank takes.
var lifecycleRamp = []string{"var(--seq-1)", "var(--seq-2)", "var(--ink-3)", "var(--seq-3)", "var(--rule-2)"}

// explorer holds everything the explorer's pages are rendered from.
type explorer struct {
	c      *composer
	export RecordExport
	// byID and byPath index the export's nodes the two ways the pages ask for
	// one: from a link, and from a relative path in a record's own prose.
	byID   map[string]ExportNode
	byPath map[string]ExportNode
	// out and in are the typed links, phrased from each end.
	out map[string][]ExportEdge
	in  map[string][]ExportEdge
	// mentions are the undirected body references, per record.
	mentions map[string][]string
	// stubs are the outbound references whose target no file answers to. They
	// render as dashed stubs — the ruling is that a retired id is shown as
	// absent, never as a link to nothing and never as an invented position.
	stubs map[string][]ExportEdge
	// principles and disciplines are the foundations page's two card decks.
	principles  []ExportNode
	disciplines []ExportNode
	// glossary is every term the repository declares, in the directory's own
	// order; glossaryByPath answers the other question a term file asks — which
	// entry a repo-relative path is — so a link between two term files reaches
	// the sibling's page rather than the forge.
	glossary       []glossaryEntry
	glossaryByPath map[string]glossaryEntry
	// terms links the first use of each term on a page to its entry.
	terms *termLinker
	// eyebrow is the record root's own heading, with its provenance.
	eyebrow, eyebrowSrc string
	// bib is the bibliography, or nil where the repository keeps none.
	bib *Bibliography
}

// newExplorer indexes the export for the pages.
//
// It reads one thing beyond the export: the glossary, whose entries become their
// own pages and whose terms become links on every record page. A repository that
// keeps none simply has none — the error is reserved for a glossary that is
// there and cannot be read.
func newExplorer(c *composer, export RecordExport, bib *Bibliography, recordRoot string) (*explorer, error) {
	e := &explorer{
		c: c, export: export, bib: bib,
		byID:           make(map[string]ExportNode, len(export.Nodes)),
		byPath:         make(map[string]ExportNode, len(export.Nodes)),
		out:            map[string][]ExportEdge{},
		in:             map[string][]ExportEdge{},
		mentions:       map[string][]string{},
		stubs:          map[string][]ExportEdge{},
		glossaryByPath: map[string]glossaryEntry{},
	}
	entries, err := loadGlossaryEntries(c.root)
	if err != nil {
		return nil, err
	}
	e.glossary = entries
	for _, en := range entries {
		e.glossaryByPath[en.Path] = en
	}
	e.terms = newTermLinker(entries)
	for _, n := range export.Nodes {
		e.byID[n.ID] = n
		e.byPath[n.Path] = n
		switch {
		case n.Type == "principle":
			e.principles = append(e.principles, n)
		case n.Lifecycle == disciplinesLifecycle:
			e.disciplines = append(e.disciplines, n)
		}
	}
	for _, ed := range export.Edges {
		e.out[ed.From] = append(e.out[ed.From], ed)
		e.in[ed.To] = append(e.in[ed.To], ed)
	}
	for _, m := range export.Mentions {
		e.mentions[m.From] = append(e.mentions[m.From], m.To)
		e.mentions[m.To] = append(e.mentions[m.To], m.From)
	}
	for _, d := range c.graph.Dangling {
		if _, ok := e.byID[d.From]; !ok {
			continue
		}
		rel := d.Field
		if m, ok := relationOf[d.Field]; ok {
			rel = m.rel
		}
		e.stubs[d.From] = append(e.stubs[d.From], ExportEdge{From: d.From, To: d.To, Rel: rel})
	}
	if recordRoot != "" {
		rel := recordRoot + "/README.md"
		if data, err := fsutil.ReadGuardedInRoot(c.root, rel, maxPageBytes); err == nil {
			if h := firstHeading(rel, string(data), ""); h != "" {
				e.eyebrow, e.eyebrowSrc = h, srcAttr(rel, "")
			}
		}
	}
	return e, nil
}

// hasFoundations reports whether the repository declares anything to found the
// work on. Without either store the page and its navigation entry are omitted.
func (e *explorer) hasFoundations() bool {
	return len(e.principles) > 0 || len(e.disciplines) > 0
}

// hasReferences reports whether the bibliography rendered.
func (e *explorer) hasReferences() bool { return e.bib != nil && len(e.bib.Entries) > 0 }

// Pages renders every explorer page, keyed by its output path.
func (e *explorer) Pages() (map[string]string, error) {
	pages := map[string]string{}
	add := func(route string, render func() (string, error)) error {
		html, err := render()
		if err != nil {
			return err
		}
		pages[route+"index.html"] = html
		return nil
	}
	if err := add(routeDashboard, e.dashboard); err != nil {
		return nil, err
	}
	if err := add(routeGraph, e.graphPage); err != nil {
		return nil, err
	}
	if err := add(routeContributors, e.contributorsPage); err != nil {
		return nil, err
	}
	if e.hasHealth() {
		if err := add(routeHealth, e.healthPage); err != nil {
			return nil, err
		}
	}
	if e.hasFoundations() {
		if err := add(routeFoundations, e.foundationsPage); err != nil {
			return nil, err
		}
	}
	if e.hasDevelopment() {
		if err := add(routeDevelopment, e.developmentPage); err != nil {
			return nil, err
		}
	}
	if e.hasReferences() {
		if err := add(routeReferences, e.referencesPage); err != nil {
			return nil, err
		}
	}
	if e.hasGlossary() {
		if err := add(routeGlossary, e.glossaryIndexPage); err != nil {
			return nil, err
		}
		for _, en := range e.glossary {
			html, err := e.glossaryTermPage(en)
			if err != nil {
				return nil, err
			}
			pages[en.Route+"index.html"] = html
		}
	}
	for _, n := range e.export.Nodes {
		html, err := e.recordPage(n)
		if err != nil {
			return nil, err
		}
		pages[RecordRoute(n)+"index.html"] = html
	}
	return pages, nil
}

// RecordRoute is where one record's page is served from.
func RecordRoute(n ExportNode) string { return "record/" + n.Type + "/" + n.ID + "/" }

// --- the shell ------------------------------------------------------------

// shell wraps one explorer page in the shared header, sub-navigation and footer.
// The build fact renders twice already — the header pill and the footer stamp —
// so the page heading carries no dateline of its own.
func (e *explorer) shell(route, title, script, body string) string {
	var b strings.Builder
	b.WriteString(e.c.headWith(title, script))
	b.WriteString(e.c.headerFor("/record/"))
	b.WriteString(`<main id="app"><div class="page">`)
	b.WriteString(e.subnav(route))
	b.WriteString(`<div class="wrap record">`)
	b.WriteString(`<div class="rechead">`)
	if e.eyebrow != "" {
		b.WriteString(`<p class="eyebrow"` + e.eyebrowSrc + `>` + escapeText(e.eyebrow) + `</p>`)
	}
	b.WriteString(`<h1 class="pagetitle">` + escapeText(title) + `</h1>`)
	b.WriteString(`</div>`)
	b.WriteString(body)
	b.WriteString(`</div></div></main>`)
	b.WriteString(e.c.footer())
	b.WriteString("</body>\n</html>\n")
	return b.String()
}

// subnav renders the explorer's own tab strip. An entry whose page is omitted is
// omitted here too, so the strip can never point at a route the build did not
// write.
func (e *explorer) subnav(active string) string {
	// The reading order runs from what the record HOLDS to what is wrong with
	// it: the dashboard, the two stores read as decks, how they connect, then
	// the findings. The glossary follows Foundations because it is the other
	// thing that holds — what the record's words MEAN, rather than what it has
	// decided — and because every page after it links into it. Contributors and
	// References are about the record's provenance rather than its content, so
	// they sit apart at the end — named, not marked. A glyph was tried in their
	// place and read as decoration.
	type tab struct{ route, label string }
	tabs := []tab{{routeDashboard, e.c.ui.RecordNav.Dashboard}}
	if e.hasFoundations() {
		tabs = append(tabs, tab{routeFoundations, e.c.ui.RecordNav.Foundations})
	}
	if e.hasGlossary() {
		tabs = append(tabs, tab{routeGlossary, e.c.ui.RecordNav.Glossary})
	}
	if e.hasDevelopment() {
		tabs = append(tabs, tab{routeDevelopment, e.c.ui.RecordNav.Development})
	}
	tabs = append(tabs, tab{routeGraph, e.c.ui.RecordNav.Graph})
	if e.hasHealth() {
		tabs = append(tabs, tab{routeHealth, e.c.ui.RecordNav.Health})
	}
	tabs = append(tabs, tab{routeContributors, e.c.ui.RecordNav.Contributors})
	if e.hasReferences() {
		tabs = append(tabs, tab{routeReferences, e.c.ui.NavReferences})
	}
	var b strings.Builder
	b.WriteString(`<nav class="sub" aria-label="` + escapeAttr(e.c.ui.NavRecord) + `"><div class="wrap">`)
	for _, t := range tabs {
		cls, cur := "", ""
		if t.route == active {
			cls, cur = ` class="on"`, ` aria-current="page"`
		}
		b.WriteString(`<a href="/` + t.route + `"` + cls + cur + `>` + escapeText(t.label) + `</a>`)
	}
	b.WriteString(`</div></nav>`)
	return b.String()
}

// --- shared pieces --------------------------------------------------------

// panel is one dashboard card: a heading, an optional right-aligned note, and a
// body. Its heading is an interface label from `site-src/ui.json`, which the
// generator is entitled to add and which names no repository span.
func panel(span, heading, note, body string) string {
	return panelSourced(span, heading, "", note, body)
}

// panelSourced is a panel whose HEADING is repository text rather than an
// interface label.
//
// The references page titles its two panels with the acknowledgement file's own
// headings — the generator holds those names only as search keys — so the
// heading has to name the span it was lifted from exactly as the body under it
// does. `headingSrc` is a `path#anchor` span, or empty for the interface-label
// case.
func panelSourced(span, heading, headingSrc, note, body string) string {
	cls := "panel"
	if span != "" {
		cls += " " + span
	}
	src := ""
	if headingSrc != "" {
		rel, anchor, _ := strings.Cut(headingSrc, "#")
		src = srcAttr(rel, anchor)
	}
	var b strings.Builder
	b.WriteString(`<div class="` + escapeAttr(cls) + `"><h3` + src + `>` + escapeText(heading))
	if note != "" {
		b.WriteString(`<span>` + escapeText(note) + `</span>`)
	}
	b.WriteString(`</h3>` + body + `</div>`)
	return b.String()
}

// panelDisclosure is a panel a reader opens: the heading is the summary, and
// the body is folded away until they ask for it. It is the shape the
// relationship chart's list already uses, for the case where a page offers two
// long bodies a reader chooses BETWEEN rather than reads side by side.
func panelDisclosure(span, heading, headingSrc, note, body string) string {
	return panelFold(span, heading, headingSrc, note, body, false)
}

// panelFold is panelDisclosure with the open state named. A deck a reader
// arrives to read opens; one they choose between stays shut.
func panelFold(span, heading, headingSrc, note, body string, open bool) string {
	cls := "panel fold"
	if span != "" {
		cls += " " + span
	}
	src := ""
	if headingSrc != "" {
		rel, anchor, _ := strings.Cut(headingSrc, "#")
		src = srcAttr(rel, anchor)
	}
	var b strings.Builder
	att := `<details class="` + escapeAttr(cls) + `"`
	if open {
		att += ` open`
	}
	b.WriteString(att + `><summary><h3` + src + `>` + escapeText(heading))
	if note != "" {
		b.WriteString(`<span>` + escapeText(note) + `</span>`)
	}
	b.WriteString(`</h3></summary>` + body + `</details>`)
	return b.String()
}

// segment is one slice of a lifecycle bar.
type segment struct {
	Label string
	N     int
	Rank  int
}

// stateSegments is how one store grades its records: by the directory a record
// sits in where the store moves them, by the frontmatter status where the store
// is flat. A store that does neither has nothing to bar.
func (e *explorer) stateSegments(typ string) []segment {
	segs := segments(e.export.Counts.ByLifecycle[typ])
	if len(segs) == 1 && segs[0].Label == "" {
		segs = segments(e.export.Counts.ByStatus[typ])
	}
	kept := segs[:0]
	for _, s := range segs {
		if s.Label != "" {
			kept = append(kept, s)
		}
	}
	return kept
}

// segments turns one store's state counts into ordered slices.
func segments(byLifecycle map[string]int) []segment {
	out := make([]segment, 0, len(byLifecycle))
	for k, v := range byLifecycle {
		rank, ok := lifecycleRank[k]
		if !ok {
			rank = len(lifecycleRamp) - 1
		}
		out = append(out, segment{Label: k, N: v, Rank: rank})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Rank != out[j].Rank {
			return out[i].Rank < out[j].Rank
		}
		return out[i].Label < out[j].Label
	})
	return out
}

// segBar renders a lifecycle bar with its legend. The legend is real text
// carrying every label and count, so the bar needs no table twin — a twin
// only accompanies a visual whose numbers render nowhere else as text.
func (e *explorer) segBar(segs []segment) string {
	total := 0
	for _, s := range segs {
		total += s.N
	}
	if total == 0 {
		return ""
	}
	var bar, leg strings.Builder
	for _, s := range segs {
		colour := lifecycleRamp[s.Rank]
		pct := strconv.FormatFloat(float64(s.N)/float64(total)*100, 'f', 2, 64)
		bar.WriteString(`<i style="width:` + pct + `%;background:` + colour + `" title="` +
			escapeAttr(s.Label) + `: ` + strconv.Itoa(s.N) + `"></i>`)
		leg.WriteString(`<span><i style="background:` + colour + `"></i>` + escapeText(s.Label) +
			` <b class="tnum">` + strconv.Itoa(s.N) + `</b></span>`)
	}
	return `<div class="seg" role="presentation">` + bar.String() + `</div>` +
		`<div class="legend">` + leg.String() + `</div>`
}

// --- the dashboard --------------------------------------------------------

// dashboard renders `/record/`: what the record holds, counted.
func (e *explorer) dashboard() (string, error) {
	ui := e.c.ui
	var b strings.Builder
	b.WriteString(`<div class="dash">`)

	// Stat tiles. A store the repository does not keep gets no tile rather than
	// a tile reading zero.
	if n := len(e.export.Releases); n > 0 {
		sub := []string{}
		r := e.export.Releases[0]
		sub = append(sub, "v"+r.Version, r.Date)
		b.WriteString(tile(strconv.Itoa(n), ui.Tiles.Releases, sub))
	}
	for _, typ := range e.storeOrder() {
		n := e.export.Counts.ByType[typ]
		if n == 0 {
			continue
		}
		var sub []string
		for _, s := range e.stateSegments(typ) {
			sub = append(sub, strconv.Itoa(s.N)+" "+s.Label)
		}
		if len(sub) > 2 {
			sub = sub[:2]
		}
		b.WriteString(tileLinked(e.tileHref(typ), strconv.Itoa(n), ui.Tiles.ForType(typ), sub, ""))
	}

	// The genealogy sits directly under the counts, folded shut: it is how the
	// record got where it is, which a reader asks for rather than arrives at.
	b.WriteString(panelDisclosure("c12", ui.RecordNav.Timeline, "",
		strconv.Itoa(len(e.export.Releases))+" "+ui.Tiles.Releases, e.genealogy()))

	// State bars, one per store that grades its records at all.
	for _, typ := range e.storeOrder() {
		segs := e.stateSegments(typ)
		if len(segs) < 2 {
			continue
		}
		caption := ui.Tiles.ForType(typ)
		b.WriteString(panel("c6", caption, strconv.Itoa(e.export.Counts.ByType[typ]), e.segBar(segs)))
	}

	b.WriteString(e.latestDecisions())
	b.WriteString(e.health())
	b.WriteString(`</div>`)

	return e.shell(routeDashboard, ui.RecordNav.Dashboard, "", b.String()), nil
}

// tile is one stat tile.
func tile(n, label string, sub []string) string { return tileExtra(n, label, sub, "") }

// tileHref is where a store's count leads: the page that reads that store,
// anchored at the store itself. A number a reader cannot follow is a dead end,
// and the destinations are DERIVED from which pages the build actually wrote —
// a store whose page is omitted gets an unlinked tile rather than a dead link
// (itd-140: graceful absence).
func (e *explorer) tileHref(typ string) string {
	switch typ {
	case "principle":
		if e.hasFoundations() {
			return "/" + routeFoundations + "#principle"
		}
	case "adr", "intent", "spec", "issue":
		if e.hasDevelopment() {
			return "/" + routeDevelopment + "#" + typ
		}
	}
	return ""
}

// tileExtra is a tile with a final pre-rendered line, for the case where a
// figure needs its number and its words in separate elements.
func tileExtra(n, label string, sub []string, extra string) string {
	return tileLinked("", n, label, sub, extra)
}

// tileLinked is a tile that leads somewhere. The whole tile is the target, so
// the number and its label are one thing to click rather than a label with a
// link hidden in it; an href of "" renders the same tile, inert.
func tileLinked(href, n, label string, sub []string, extra string) string {
	var b strings.Builder
	open, close := `<div class="tile">`, `</div>`
	if href != "" {
		open, close = `<a class="tile lead" href="`+escapeAttr(href)+`">`, `</a>`
	}
	b.WriteString(`<div class="panel c2">` + open)
	b.WriteString(`<span class="n">` + escapeText(n) + `</span>`)
	b.WriteString(`<span class="l">` + escapeText(label) + `</span>`)
	for _, s := range sub {
		if s == "" {
			continue
		}
		b.WriteString(`<span class="s">` + escapeText(s) + `</span>`)
	}
	b.WriteString(extra)
	b.WriteString(close + `</div>`)
	return b.String()
}

// storeOrder is the order the stores are read in, decisions first — the same
// order the chart's same-day tie-break uses, so the two pages agree.
func (e *explorer) storeOrder() []string {
	types := make([]string, 0, len(e.export.Counts.ByType))
	for t := range e.export.Counts.ByType {
		types = append(types, t)
	}
	sort.Slice(types, func(i, j int) bool {
		if a, b := typeRank(types[i]), typeRank(types[j]); a != b {
			return a < b
		}
		return types[i] < types[j]
	})
	return types
}

// latestDecisions lists the newest ratified decisions by their own dates. It is
// ids, dates and titles: the record's words, never a summary written here.
func (e *explorer) latestDecisions() string {
	var adrs []ExportNode
	for _, n := range e.export.Nodes {
		if n.Type == "adr" {
			adrs = append(adrs, n)
		}
	}
	if len(adrs) == 0 {
		return ""
	}
	sort.SliceStable(adrs, func(i, j int) bool {
		if adrs[i].Date != adrs[j].Date {
			return adrs[i].Date > adrs[j].Date
		}
		return handleNum(adrs[i].ID) > handleNum(adrs[j].ID)
	})
	if len(adrs) > 7 {
		adrs = adrs[:7]
	}
	var b strings.Builder
	b.WriteString(`<ul class="list">`)
	for _, n := range adrs {
		b.WriteString(`<li><span class="id"><a href="/` + escapeAttr(RecordRoute(n)) + `">` + escapeText(n.ID) +
			`</a><span class="d">` + escapeText(n.Date) + `</span></span>` +
			`<span>` + escapeText(shortTitle(n)) + `</span></li>`)
	}
	b.WriteString(`</ul>`)
	return panel("c8", e.c.ui.Panels.Latest, e.c.ui.Tiles.ADR, b.String())
}

// health renders the record's own reference hygiene: every typed reference the
// tree cannot resolve, measured against the committed ratchet.
func (e *explorer) health() string {
	h := e.export.Health
	ui := e.c.ui
	var b strings.Builder
	b.WriteString(`<div class="health">`)
	// Two lines per finding: the fact, then its explanation — a flowing line
	// wrapped mid-phrase in the narrow panel.
	for _, u := range h.Unresolved {
		b.WriteString(`<div class="hitem"><span class="hfact"><span class="w">!</span> <a href="/` +
			escapeAttr(e.routeOf(u.From)) + `">` + escapeText(u.From) + `</a> → <b class="stub">` +
			escapeText(u.To) + `</b></span><span class="hwhy">` +
			escapeText(relationWord(u.Rel)) + ` ` + escapeText(ui.Record.NotInTree) + `</span></div>`)
	}
	// Every summary number carries its word; three bare numbers read as a
	// rendering fault.
	// Each figure is its own element: a number glued to a word is neither a
	// number nor an interface string to the provenance walk, and three of them
	// run together read as one sentence that says nothing.
	b.WriteString(`<div class="hsum">`)
	for _, f := range []struct {
		n     int
		label string
	}{
		{len(h.Unresolved), ui.Panels.Unresolved},
		{h.BaselineCount, ui.Panels.Baseline},
		{e.export.Layout.Isolated, ui.Panels.Isolated},
	} {
		b.WriteString(`<span><b class="tnum">` + strconv.Itoa(f.n) + `</b> ` + escapeText(f.label) + `</span>`)
	}
	b.WriteString(`</div>`)
	b.WriteString(`</div>`)
	return panel("c4", ui.Panels.Health, strconv.Itoa(h.BaselineCount), b.String())
}

// routeOf is a record's page, or "" where no file answers to the id.
func (e *explorer) routeOf(id string) string {
	n, ok := e.byID[id]
	if !ok {
		return ""
	}
	return RecordRoute(n)
}

// shortTitle drops the handle a record repeats in its own H1 ("ADR-47: …"), so
// a list of ids does not print each one twice.
func shortTitle(n ExportNode) string {
	prefix := strings.ToUpper(n.ID) + ":"
	if strings.HasPrefix(strings.ToUpper(n.Title), prefix) {
		return strings.TrimSpace(n.Title[len(prefix):])
	}
	return n.Title
}

// dayNumber converts a YYYY-MM-DD date to a day count, for placing a mark on an
// axis. It is a plain civil-calendar computation: no clock, no zone, no library.
func dayNumber(date string) int {
	if len(date) < 10 {
		return 0
	}
	y, err1 := strconv.Atoi(date[0:4])
	m, err2 := strconv.Atoi(date[5:7])
	d, err3 := strconv.Atoi(date[8:10])
	if err1 != nil || err2 != nil || err3 != nil {
		return 0
	}
	// Howard Hinnant's days-from-civil: exact for every date, and it is the same
	// arithmetic on every machine, which is what the golden files rest on.
	if m <= 2 {
		y--
	}
	era := y / 400
	if y < 0 {
		era = (y - 399) / 400
	}
	yoe := y - era*400
	mp := (m + 9) % 12
	doy := (153*mp+2)/5 + d - 1
	doe := yoe*365 + yoe/4 - yoe/100 + doy
	return era*146097 + doe - 719468
}

// --- foundations ----------------------------------------------------------

// foundationsPage lists what the work is founded on: the principles that hold
// across the record, and the disciplines that state a rule rather than ship a
// change. It LISTS AND LINKS and never explains — the explanation is the record
// page each card opens, and the context belongs in the documentation.
func (e *explorer) foundationsPage() (string, error) {
	var b strings.Builder
	b.WriteString(`<div class="dash">`)
	deck := func(anchor, label string, nodes []ExportNode) {
		if len(nodes) == 0 {
			return
		}
		var cards strings.Builder
		cards.WriteString(`<div class="fcards">`)
		for _, n := range nodes {
			cards.WriteString(`<a class="fcard" href="/` + escapeAttr(RecordRoute(n)) + `">` +
				`<span class="t">` + escapeText(shortTitle(n)) + `</span>` +
				`<span class="id">` + escapeText(n.ID) + `</span></a>`)
		}
		cards.WriteString(`</div>`)
		// The deck's own name is its anchor, so a dashboard tile can land on the
		// store it counts rather than at the top of the page.
		b.WriteString(`<div id="` + escapeAttr(anchor) + `" class="c12">`)
		b.WriteString(panelFold("c12", label, "", strconv.Itoa(len(nodes)), cards.String(), true))
		b.WriteString(`</div>`)
	}
	deck("principle", e.c.ui.Tiles.Principle, e.principles)
	deck("discipline", e.c.ui.Tiles.Discipline, e.disciplines)
	b.WriteString(`</div>`)
	return e.shell(routeFoundations, e.c.ui.RecordNav.Foundations, "", b.String()), nil
}

// --- contributors ---------------------------------------------------------

// contributorsPage renders `/contributors/`: who authored the history, and what
// assisted.
//
// The two are different facts and the page keeps them apart. Humans are the
// authors of record. The `Assisted-by:` tallies are DISCLOSURE, presented next
// to the policy that requires them — and this page is the one place on the site
// where a model name may appear, under the declared attribution escape
// (adr-47 decision 3).
func (e *explorer) contributorsPage() (string, error) {
	a := e.export.Authorship
	ui := e.c.ui
	var b strings.Builder
	b.WriteString(`<div class="dash">`)

	// The page carries two things and nothing else: who authored the history,
	// and what assisted. The stat tiles that stood above them repeated numbers
	// the two panels already hold — the authors table has the commit counts, the
	// trailers figure has its own total — and the disclosure rate belongs with
	// the other findings, on the health page.
	if len(a.Humans) > 0 || len(a.Bots) > 0 {
		var rows strings.Builder
		rows.WriteString(`<thead><tr><th>` + escapeText(ui.Contributors.Authors) + `</th><th class="tnum">` +
			escapeText(ui.Tiles.Commits) + `</th></tr></thead><tbody>`)
		for _, h := range a.Humans {
			name := escapeText(h.Name)
			// A noreply-derived profile links the author's own page; an author
			// without one stays plain text (itd-140: graceful absence).
			if h.Profile != "" {
				name = `<a href="` + escapeAttr(h.Profile) + `">` + name + `</a>`
			}
			rows.WriteString(`<tr><td>` + name + `</td><td class="tnum">` + strconv.Itoa(h.Commits) + `</td></tr>`)
		}
		for _, t := range a.Bots {
			rows.WriteString(`<tr class="muted"><td>` + escapeText(t.Name) + ` <span class="rel">` +
				escapeText(ui.Contributors.Tools) + `</span></td><td class="tnum">` + strconv.Itoa(t.Commits) + `</td></tr>`)
		}
		rows.WriteString(`</tbody>`)
		policy, err := e.policyQuote()
		if err != nil {
			return "", err
		}
		body := `<div class="tablewrap"><table>` + rows.String() + `</table></div>` + policy
		b.WriteString(panelDisclosure("c12", ui.Contributors.Authors, "",
			strconv.Itoa(len(a.Humans)), body))
	}

	if len(a.ByModel) > 0 {
		maxN := a.ByModel[0].Commits
		var bars strings.Builder
		bars.WriteString(`<div class="bars">`)
		for _, m := range a.ByModel {
			pct := strconv.FormatFloat(float64(m.Commits)/float64(maxN)*100, 'f', 1, 64)
			bars.WriteString(`<span class="lab">` + escapeText(m.Model) + `</span>` +
				`<span class="bar"><i style="width:` + pct + `%"></i></span>` +
				`<span class="tnum small">` + strconv.Itoa(m.Commits) + `</span>`)
		}
		bars.WriteString(`</div>`)
		// The chart tallies OCCURRENCES and its note says what the bars sum to.
		// The two commit-level facts that are not assistance — the human-only
		// declaration, and the commits carrying no trailer at all — are stated
		// beneath it rather than folded into a chart they would falsify.
		var foot strings.Builder
		foot.WriteString(`<p class="small muted trailerfoot">`)
		foot.WriteString(escapeText(ui.Contributors.DeclaredNone) + ` <b class="tnum">` +
			strconv.Itoa(a.DeclaredNone) + `</b>`)
		foot.WriteString(`</p><p class="small muted trailerfoot">` + escapeText(ui.Contributors.Undeclared) + ` <b class="tnum">` +
			strconv.Itoa(a.Undeclared) + `</b>`)
		foot.WriteString(`</p>`)
		b.WriteString(panelDisclosure("c12", ui.Contributors.Trailers, "",
			strconv.Itoa(a.Assisted), bars.String()+foot.String()))
	}
	b.WriteString(`</div>`)

	return e.shell(routeContributors, ui.RecordNav.Contributors, "", b.String()), nil
}

// policyQuote renders the attribution policy the manifest selects, verbatim,
// with the file it came from linked. The number above it means nothing without
// the rule beside it.
//
// A repository that DECLARES no policy simply has none, and the page renders
// without it. A repository that names one and cannot supply it REFUSES: the
// tallies would go out unaccompanied, which is the reading — assistance as
// authorship — the whole page exists to prevent.
func (e *explorer) policyQuote() (string, error) {
	p := e.c.manifest.RecordPages.Contributors.Policy
	if p.File == "" {
		return "", nil
	}
	bad := func(why string) error {
		return fmt.Errorf("site: record_pages.contributors.policy names %s § %s, and %s — the assistance tallies are not published without the rule beside them",
			p.File, p.Heading, why)
	}
	data, err := fsutil.ReadGuardedInRoot(e.c.root, p.File, maxPageBytes)
	if err != nil {
		if os.IsNotExist(err) {
			return "", bad("the repository does not carry it")
		}
		return "", err
	}
	body, consumed := StripFrontmatter(string(data))
	secs, err := Sections(p.File, body, consumed)
	if err != nil {
		return "", err
	}
	for _, s := range secs {
		if !strings.EqualFold(s.Title, p.Heading) {
			continue
		}
		blocks := Blocks(s.Body, s.BodyLine)
		if len(blocks) == 0 {
			return "", bad("that section is empty")
		}
		if p.Part == "first-bullet" {
			// The first BULLET, not the first block. A policy section opens with
			// its own preamble more often than not, and quoting that instead
			// published a dangling lead-in — "The rules:" and then nothing —
			// under the number it was supposed to explain.
			item := -1
			for i, blk := range blocks {
				if isUnorderedItem(strings.TrimLeft(blk.Text, " \t")) {
					item = i
					break
				}
			}
			if item < 0 {
				return "", bad("that section has no bullet to quote")
			}
			text, line := firstListItem(blocks[item])
			blocks = []Block{{Text: text, Line: line}}
		}
		r := &Renderer{UI: e.c.ui, Refs: LinkDefinitions(body),
			Image: func(src, alt string, at Source) (string, error) {
				return e.c.assets.render(path.Dir(p.File), src, alt, at)
			},
			Link: func(href string, at Source) string { return e.href(p.File, href) }}
		h, err := r.RenderBlocks(p.File, blocks)
		if err != nil {
			return "", err
		}
		out := `<div class="prose small policy"` + srcAttr(p.File, s.Anchor) + `>` + h
		if e.c.repo.Repository != "" {
			out += `<p class="small"><a href="` + escapeAttr(e.c.repo.Repository+"/blob/main/"+p.File) + `">` +
				escapeText(p.File) + `</a></p>`
		}
		return out + `</div>`, nil
	}
	return "", bad("that file has no such heading")
}

// --- link rewriting -------------------------------------------------------

// href maps a link as a record wrote it to the link the site serves.
//
// A relative path to another RECORD becomes that record's page here, which is
// what the explorer adds: before it, the only honest destination was the file on
// the forge. Everything else falls through to the landing page's own rule.
func (e *explorer) href(fromPath, target string) string {
	switch {
	case target == "",
		strings.HasPrefix(target, "http://"),
		strings.HasPrefix(target, "https://"),
		strings.HasPrefix(target, "mailto:"),
		strings.HasPrefix(target, "#"),
		strings.HasPrefix(target, "/"):
		return target
	}
	file, frag, _ := strings.Cut(target, "#")
	rel := path.Clean(path.Join(path.Dir(fromPath), file))
	if strings.HasSuffix(file, ".md") {
		if n, ok := e.byPath[rel]; ok {
			out := "/" + RecordRoute(n)
			if frag != "" {
				out += "#" + frag
			}
			return out
		}
		// A term file naming a sibling term reaches that term's page. Without
		// this the glossary's own cross-references leave the site for the forge,
		// which is the one place a reader who followed a term link did not mean
		// to end up.
		if en, ok := e.glossaryByPath[rel]; ok {
			out := "/" + en.Route
			if frag != "" {
				out += "#" + frag
			}
			return out
		}
		return siteHref(path.Dir(fromPath), target, e.c.repo.Repository)
	}
	// A relative target that is NOT markdown — a directory, a script, a
	// configuration file. `siteHref` leaves those exactly as the record wrote
	// them, which on the landing page is harmless (nothing links one) and on a
	// record's page is a link relative to `/record/<type>/<id>/` that resolves
	// to nothing. It resolves against the repository root instead, and points at
	// the forge's own view of whatever is there.
	if e.c.repo.Repository == "" || file == "" || !fsutil.ValidRelPath(rel) {
		return target
	}
	kind := "blob"
	if dir, err := e.c.root.Stat(rel); err == nil && dir.IsDir() {
		kind = "tree"
	} else if err != nil {
		// The tree does not carry it. The record's own text stands: inventing a
		// forge URL for a path that is not there trades a broken relative link
		// for a confident 404.
		return target
	}
	out := e.c.repo.Repository + "/" + kind + "/main/" + rel
	if frag != "" {
		out += "#" + frag
	}
	return out
}

// forgeBlob is the record file on the forge, or "" without a known forge.
func (e *explorer) forgeBlob(rel string) string {
	if e.c.repo.Repository == "" {
		return ""
	}
	return e.c.repo.Repository + "/blob/main/" + rel
}

// forgeCommits is the record file's commit history on the forge — the link that
// makes an amendment traceable from the date rather than merely visible as one.
func (e *explorer) forgeCommits(rel string) string {
	if e.c.repo.Repository == "" {
		return ""
	}
	return e.c.repo.Repository + "/commits/main/" + rel
}

// nodesOfType is every record of one store, in the export's order.
func (e *explorer) nodesOfType(typ string) []ExportNode {
	var out []ExportNode
	for _, n := range e.export.Nodes {
		if n.Type == typ {
			out = append(out, n)
		}
	}
	return out
}
