package site

// `/record/graph/` — the relationship chart's page.
//
// The page itself is a shell: a canvas, the controls in the corners, and the
// list twin under the stage. Everything that moves is `site-src/record.js`,
// lifted from the prototype, and everything it draws comes from `record.json`,
// whose two arrangements were computed at build time. The page runs no layout
// engine and makes no network call beyond fetching that one file.
//
// Every string the script shows is put into the markup HERE, out of
// `site-src/ui.json`, and read back from a data attribute — so the allowlist
// stays the complete list of what a reader sees, script included.

import (
	"sort"
	"strings"
)

// graphPage renders `/record/graph/`.
func (e *explorer) graphPage() (string, error) {
	ui := e.c.ui
	g := ui.Graph
	var b strings.Builder

	b.WriteString(`<div class="bwrap" data-graph="/record.json"`)
	b.WriteString(` data-linked="` + escapeAttr(g.Linked) + `"`)
	b.WriteString(` data-nolinks="` + escapeAttr(g.NoLinks) + `"`)
	b.WriteString(` data-mentions="` + escapeAttr(ui.Record.Mentions) + `"`)
	b.WriteString(` data-open="` + escapeAttr(ui.Record.OpenOnForge) + `"`)
	b.WriteString(` data-history="` + escapeAttr(ui.Record.CommitHistory) + `"`)
	b.WriteString(` data-notintree="` + escapeAttr(ui.Record.NotInTree) + `"`)
	// The four dates on the card's continuum are named by the record's OWN
	// field names — the same words the per-record page's frontmatter table
	// prints — so the line can be read out as "date … created … touched …"
	// rather than as four unlabelled dates, without the generator inventing a
	// word for any of them.
	b.WriteString(` data-f-date="` + escapeAttr(fieldDate) + `"`)
	b.WriteString(` data-f-created="` + escapeAttr(fieldCreated) + `"`)
	b.WriteString(` data-f-touched="` + escapeAttr(fieldTouched) + `"`)
	b.WriteString(` data-close="` + escapeAttr(g.Close) + `"`)
	b.WriteString(` data-back="` + escapeAttr(g.Back) + `"`)
	b.WriteString(` data-forward="` + escapeAttr(g.Forward) + `"`)
	b.WriteString(` data-fs-on="` + escapeAttr(g.FullScreen) + `"`)
	b.WriteString(` data-fs-off="` + escapeAttr(g.ExitFullScreen) + `"`)
	b.WriteString(` data-navlabel="` + escapeAttr(g.History) + `"`)
	b.WriteString(` data-rel-blocked-by="` + escapeAttr(ui.Relations.BlockedBy) + `"`)
	b.WriteString(` data-rel-supersedes="` + escapeAttr(ui.Relations.Supersedes) + `"`)
	b.WriteString(` data-rel-implements="` + escapeAttr(ui.Relations.Implements) + `"`)
	b.WriteString(` data-rel-builds-on="` + escapeAttr(ui.Relations.BuildsOn) + `"`)
	if repo := e.c.repo.Repository; repo != "" {
		b.WriteString(` data-blob="` + escapeAttr(forgeBlob(repo, "")) + `"`)
		b.WriteString(` data-commits="` + escapeAttr(repo+"/commits/main/") + `"`)
	}
	b.WriteString(`>`)

	b.WriteString(`<div class="bstage">`)
	b.WriteString(`<canvas id="bc" role="img" aria-label="` + escapeAttr(ui.RecordNav.Graph) + `"></canvas>`)

	// Top-left: the arrangement control and the filters pop. The middle of the
	// stage belongs to the chart.
	b.WriteString(`<div class="brow2">`)
	b.WriteString(`<div class="bseg" id="barr" role="radiogroup" aria-label="` + escapeAttr(g.Arrange) + `">`)
	b.WriteString(`<button role="radio" aria-checked="true" data-arr="date">` + escapeText(g.ByDate) + `</button>`)
	b.WriteString(`<button role="radio" aria-checked="false" data-arr="links">` + escapeText(g.ByLinks) + `</button>`)
	b.WriteString(`</div>`)
	b.WriteString(`<button class="bfilters-btn" id="bfilters-btn" aria-expanded="false" aria-controls="bfilters">` +
		escapeText(g.Filters) + `</button></div>`)

	// Top-right: search.
	b.WriteString(`<div class="bsearch"><input type="search" id="bsearch" placeholder="` + escapeAttr(g.Find) +
		`" aria-label="` + escapeAttr(g.Find) + `" autocomplete="off"><div class="bhits" id="bhits"></div></div>`)

	// Bottom-right: full screen and zoom.
	b.WriteString(`<div class="bcorner"><button class="bfs" id="bfs" aria-label="` + escapeAttr(g.FullScreen) +
		`" aria-pressed="false">⤢</button><div class="bzoom">`)
	b.WriteString(`<button id="bzin" aria-label="` + escapeAttr(g.ZoomIn) + `">+</button>`)
	b.WriteString(`<button id="bzout" aria-label="` + escapeAttr(g.ZoomOut) + `">−</button>`)
	b.WriteString(`<button id="bzreset" aria-label="` + escapeAttr(g.ResetView) + `">⟲</button>`)
	b.WriteString(`</div></div>`)

	b.WriteString(`<div class="bfilters" id="bfilters" hidden><div class="chips" id="typechips">`)
	for _, typ := range e.storeOrder() {
		// A chip counts the records it actually filters. Disciplines are filed
		// under intents by the record and are their own KIND to a reader — their
		// own colour, their own chip — so they are counted out of the intents
		// chip rather than sitting inside it under a different colour.
		n := e.export.Counts.ByType[typ] - e.disciplineCount(typ)
		if n == 0 {
			continue
		}
		b.WriteString(`<button class="chip on" data-t="` + escapeAttr(typ) + `"><i style="background:var(` +
			typeColourToken(typ) + `)"></i>` + escapeText(e.c.ui.Tiles.ForType(typ)) +
			` <span class="tnum muted">` + itoaLen(n) + `</span></button>`)
	}
	if d := e.disciplineCount(""); d > 0 {
		b.WriteString(`<button class="chip on" data-t="` + escapeAttr(disciplinesLifecycle) + `"><i style="background:var(` +
			typeColourToken(disciplinesLifecycle) + `)"></i>` + escapeText(e.c.ui.Tiles.Discipline) +
			` <span class="tnum muted">` + itoaLen(d) + `</span></button>`)
	}
	b.WriteString(`</div><label class="sans small"><input type="checkbox" id="gmentions"> ` +
		escapeText(g.Mentions) + `</label>`)
	b.WriteString(e.legend())
	b.WriteString(`</div>`)

	b.WriteString(`<div class="bcard" id="bcard" hidden></div>`)
	b.WriteString(`<div class="bstandby" id="bstandby" role="status" aria-live="polite">` +
		`<span class="spin" aria-hidden="true"></span>` + escapeText(ui.Standby) + `</div>`)
	b.WriteString(`</div>`)

	// The list twin: the keyboard path, and what a reader gets in place of the
	// chart when they have asked for no motion. It is rendered HERE rather than
	// built by the script, so it reaches every record and every link with the
	// script switched off entirely.
	b.WriteString(`<details class="blist"><summary class="sans">` + escapeText(g.BrowseList) + `</summary>`)
	b.WriteString(e.listTwin())
	b.WriteString(`</details>`)
	b.WriteString(`</div>`)

	return e.shell(routeGraph, ui.RecordNav.Graph, "/record.js", b.String()), nil
}

// legend names what the chart's fills and arrowheads mean, in the record's own
// words: a store name and a lifecycle bucket are facts about the record, so the
// legend needs no sentence to explain itself.
func (e *explorer) legend() string {
	swatch := func(typ, state string) string {
		c := "var(" + typeColourToken(typ) + ")"
		style := `fill="` + c + `" stroke="none"`
		switch state {
		case "ring":
			style = `fill="var(--surface)" stroke="` + c + `" stroke-width="2"`
		case "dash":
			style = `fill="var(--surface)" stroke="` + c + `" stroke-width="2" stroke-dasharray="2 1.5"`
		case "fade":
			style = `fill="` + c + `" opacity=".45"`
		}
		return `<svg viewBox="0 0 14 14" aria-hidden="true"><circle cx="7" cy="7" r="5" ` + style + `/></svg>`
	}
	// TWO encodings, in two blocks. Colour is the store a record belongs to;
	// border is the state its store grades it in. Run together in one wrapping
	// row they read as one list, and a lifecycle word wraps under a store name
	// as though it belonged to it — "closed" under "specs", which says something
	// the chart never meant. The blocks carry no headings: a swatch beside a
	// word is already the whole of what either half says, and a sentence
	// explaining a legend is a sentence the single-source rule would have to
	// find a home for.
	var b strings.Builder
	b.WriteString(`<div class="glegend">`)

	b.WriteString(`<div class="glrow">`)
	for _, typ := range e.storeOrder() {
		if e.export.Counts.ByType[typ] == 0 {
			continue
		}
		b.WriteString(`<span>` + swatch(typ, "solid") + escapeText(e.c.ui.Tiles.ForType(typ)) + `</span>`)
	}
	// Disciplines are filed under intents by the record and drawn in their own
	// colour by the chart, so the legend names them in their own right: to a
	// reader a discipline is a kind of record, not a state of an intent.
	if e.hasDisciplineNodes() {
		b.WriteString(`<span>` + swatch(disciplinesLifecycle, "solid") +
			escapeText(e.c.ui.Tiles.Discipline) + `</span>`)
	}
	b.WriteString(`</div>`)

	// One row per fill, labelled with the lifecycle words the record actually
	// uses for that state. The swatch is drawn in a neutral colour: the shape is
	// what this half of the legend is about, and colouring it would say the state
	// belongs to one store.
	fills := map[string][]string{}
	for _, n := range e.export.Nodes {
		f := lifecycleFill(n.Lifecycle)
		if !containsStr(fills[f], n.Lifecycle) && n.Lifecycle != "" && n.Lifecycle != disciplinesLifecycle {
			fills[f] = append(fills[f], n.Lifecycle)
		}
	}
	var states strings.Builder
	for _, f := range []string{"solid", "ring", "dash", "fade"} {
		words := fills[f]
		if len(words) == 0 {
			continue
		}
		sort.Strings(words)
		for _, w := range words {
			states.WriteString(`<span>` + swatch("", f) + escapeText(w) + `</span>`)
		}
	}
	if states.Len() > 0 {
		b.WriteString(`<div class="glrow glstates">`)
		b.WriteString(states.String())
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}

// disciplineCount is how many records of one store are filed as disciplines; an
// empty store name counts them across the whole record.
func (e *explorer) disciplineCount(typ string) int {
	n := 0
	for _, nd := range e.export.Nodes {
		if nd.Lifecycle == disciplinesLifecycle && (typ == "" || nd.Type == typ) {
			n++
		}
	}
	return n
}

// hasDisciplineNodes reports whether any record is filed as a discipline.
func (e *explorer) hasDisciplineNodes() bool {
	for _, n := range e.export.Nodes {
		if n.Lifecycle == disciplinesLifecycle {
			return true
		}
	}
	return false
}

// lifecycleFill is how a bubble in that state is drawn: solid when the work is
// settled, a ring while it is in play, dashed while it is a draft, faded once it
// has been set aside.
//
// A record whose store grades it by NEITHER a lifecycle directory nor a status
// field — a principle — declares no state, and is drawn solid because it has
// none. Reading a fabricated one would fade every principle on the chart as
// though it had been set aside.
func lifecycleFill(state string) string {
	if state == "" {
		return "solid"
	}
	switch statusTone(state) {
	case "done":
		return "solid"
	case "open":
		return "ring"
	case "draft":
		return "dash"
	case "notplanned":
		return "fade"
	}
	return "solid"
}

func containsStr(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

// twinLink renders one relation in the list twin: the relation's word, and the
// record it names as a link that focuses it in the chart.
func (e *explorer) twinLink(word, id string) string {
	return escapeText(word) + ` <a href="/` + escapeAttr(routeGraph) + `?focus=` + escapeAttr(id) + `">` +
		escapeText(id) + `</a>`
}

// listTwin is every record and every link the chart shows, as markup. A keyboard
// visitor and a reader who has asked for no motion both get the whole graph here
// without the chart ever running.
func (e *explorer) listTwin() string {
	nodes := append([]ExportNode{}, e.export.Nodes...)
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].Date != nodes[j].Date {
			return nodes[i].Date < nodes[j].Date
		}
		if a, b := typeRank(nodes[i].Type), typeRank(nodes[j].Type); a != b {
			return a < b
		}
		return handleNum(nodes[i].ID) < handleNum(nodes[j].ID)
	})
	var b strings.Builder
	b.WriteString(`<ul class="nodelist">`)
	for _, n := range nodes {
		b.WriteString(`<li><a class="id" href="/` + escapeAttr(RecordRoute(n)) + `">` + escapeText(n.ID) + `</a>`)
		b.WriteString(`<span>` + escapeText(shortTitle(n)) + `</span>`)
		b.WriteString(`<span class="d">` + escapeText(n.Date) + `</span>`)
		// Each link is a LINK. The twin is the keyboard path through the chart,
		// and a chart whose edges a keyboard visitor can read but not follow is
		// only half a twin: every relation named here reaches the record it
		// names, focused in the chart exactly as tapping the bubble would.
		var rels []string
		for _, ed := range e.out[n.ID] {
			rels = append(rels, e.twinLink(relationWord(ed.Rel), ed.To))
		}
		for _, ed := range e.in[n.ID] {
			rels = append(rels, e.twinLink(e.c.ui.Relations.Inverse(ed.Rel), ed.From))
		}
		for _, ed := range e.stubs[n.ID] {
			// The target has left the tree: named, dashed, and not a link.
			rels = append(rels, escapeText(relationWord(ed.Rel))+` <span class="stub">`+
				escapeText(ed.To)+`</span> `+escapeText(e.c.ui.Record.NotInTree))
		}
		if len(rels) > 0 {
			sort.Strings(rels)
			b.WriteString(`<span class="rel">` + strings.Join(rels, " · ") + `</span>`)
		}
		b.WriteString(`</li>`)
	}
	b.WriteString(`</ul>`)
	return b.String()
}
