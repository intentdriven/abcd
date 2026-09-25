package site

// `/record/timeline/` — the genealogy, as ONE STATIC SVG emitted here.
//
// Five lanes over one axis: releases, decisions, intents, specs, and issues as a
// per-day histogram. It is drawn at build time rather than by a script because
// everything it shows is already known then: the picture is deterministic,
// printable, needs no library, and scrolls inside its own panel on a phone.
//
// The rules the drawing follows are the record's, not the chart's convenience:
// decisions sit at their own frontmatter dates and everything else at the day
// its file first appeared in git; a day too crowded to show one by one becomes a
// capsule carrying its count; a supersession arc is drawn only where BOTH ends
// exist, and a target that has left the tree gets a short dashed stub ending in
// ×, never an arc to a position nothing occupies. Every mark opens its record in
// the relationship chart.

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/core/changelog"
)

// The drawing's fixed geometry, in SVG user units.
const (
	tlWidth      = 1000
	tlLeft       = 40
	tlRight      = 24
	tlTopPad     = 22
	tlLaneGap    = 26
	tlReleaseTop = 36
	tlIssueH     = 54
	tlBarWidth   = 4.2
	// tlTailDays pads the axis past the last mark so a label at the end has
	// somewhere to sit.
	tlTailDays = 6
)

// tlLane describes one dot lane's drawing parameters.
type tlLane struct {
	Type string
	// R is the mark radius, PerCol how many stack before spilling sideways, and
	// Cap the crowding above which a day becomes a capsule.
	R      float64
	PerCol int
	Cap    int
	// Height is the vertical room the lane needs above and below its axis.
	Height int
}

// tlLanes are the dot lanes, in reading order. A store the record does not keep
// is skipped, so a sparse repository gets a shorter picture rather than empty
// bands.
var tlLanes = []tlLane{
	{Type: "adr", R: 5.5, PerCol: 7, Cap: 14, Height: 120},
	{Type: "intent", R: 3.6, PerCol: 9, Cap: 18, Height: 100},
	{Type: "spec", R: 4, PerCol: 5, Cap: 10, Height: 84},
	{Type: "principle", R: 4, PerCol: 5, Cap: 10, Height: 84},
}

// genealogy is the whole genealogy — the drawing and the supersessions read as
// text — as one block. It is rendered into the DASHBOARD, folded shut, because
// it answers "how did the record get here" rather than "what does it hold": a
// reader who wants it asks for it, and one who does not is not made to scroll
// past a full-width chart to reach the counts.
func (e *explorer) genealogy() string {
	svg, _ := e.genealogySVG()
	return `<div class="panel tl">` + svg + `</div>`
}

// supersessions is the arcs and the stubs read as text. It belongs with the
// findings rather than beside the drawing: a supersession the tree cannot
// resolve is a health finding, and one it can is a fact about two records that
// their own pages already carry.
func (e *explorer) supersessions() (body string, n int) {
	_, positions := e.genealogySVG()
	body = e.supersessionList(positions)
	for _, ed := range e.export.Edges {
		if ed.Rel == "supersedes" {
			if _, ok := e.byID[ed.To]; ok {
				n++
			}
		}
	}
	for _, u := range e.export.Health.Unresolved {
		if u.Rel == "supersedes" {
			n++
		}
	}
	return body, n
}

// tlPoint is where one record's mark ended up, so an arc can find both ends.
type tlPoint struct{ X, Y float64 }

// genealogySVG draws the whole picture and reports where each record landed.
func (e *explorer) genealogySVG() (string, map[string]tlPoint) {
	first, last := e.axisRange()
	span := float64(last - first)
	if span <= 0 {
		span = 1
	}
	x := func(date string) float64 {
		return tlLeft + float64(dayNumber(date)-first)/span*float64(tlWidth-tlLeft-tlRight)
	}

	var lanes strings.Builder
	pos := map[string]tlPoint{}
	y := float64(tlTopPad + tlReleaseTop)

	// The releases lane.
	relTop := y
	if len(e.export.Releases) > 0 {
		lanes.WriteString(e.releaseLane(x, y))
		y += tlLaneGap + 34
	}

	// The dot lanes.
	type drawn struct {
		lane tlLane
		y    float64
	}
	var order []drawn
	for _, lane := range tlLanes {
		nodes := e.nodesOfType(lane.Type)
		if len(nodes) == 0 {
			continue
		}
		y += float64(lane.Height) / 2
		order = append(order, drawn{lane, y})
		y += float64(lane.Height) / 2
	}

	// Supersession arcs are drawn UNDER the marks, so a mark is never hidden by
	// a line that passes through it. They need every position first.
	for _, d := range order {
		for id, p := range swarmPositions(e.nodesOfType(d.lane.Type), x, d.y, d.lane) {
			pos[id] = p
		}
	}
	lanes.WriteString(e.arcs(x, pos, relTop))
	for _, d := range order {
		lanes.WriteString(e.dotLane(x, d.y, d.lane, pos))
	}

	// The issues lane, as a histogram: 374 dots is not a picture.
	issues := e.nodesOfType("issue")
	if len(issues) > 0 {
		y += float64(tlIssueH) + 30
		lanes.WriteString(e.issueLane(x, y, issues))
		y += 16
	}

	// The first commit: everything dated before it was designed on paper.
	marker := ""
	if fc := e.export.History.FirstCommit; fc != "" && dayNumber(fc) >= first && dayNumber(fc) <= last {
		fx := f1(x(fc))
		marker = `<line x1="` + fx + `" y1="` + strconv.Itoa(tlTopPad) + `" x2="` + fx + `" y2="` +
			f1(y) + `" stroke="var(--hazard)" stroke-width="1.5" stroke-dasharray="5 4"/>` +
			`<text x="` + f1(x(fc)-6) + `" y="` + f1(y+16) +
			`" font-size="10" text-anchor="end" fill="var(--hazard-ink)">` + escapeText(fc) + `</text>`
	}
	height := int(y) + 26

	var b strings.Builder
	b.WriteString(`<svg viewBox="0 0 ` + strconv.Itoa(tlWidth) + ` ` + strconv.Itoa(height) +
		`" class="tlsvg" role="img" aria-label="` + escapeAttr(e.c.ui.RecordNav.Timeline) + `">`)
	b.WriteString(`<defs><marker id="tlarr" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="6" markerHeight="6" orient="auto">` +
		`<path d="M0 0L10 5L0 10z" fill="var(--ink-2)"/></marker></defs>`)
	// Month gridlines, labelled with the month itself.
	for _, m := range monthStarts(first, last) {
		mx := f1(x(m))
		b.WriteString(`<line x1="` + mx + `" y1="` + strconv.Itoa(tlTopPad) + `" x2="` + mx + `" y2="` +
			f1(y) + `" class="grid"/>`)
		b.WriteString(`<text x="` + f1(x(m)+4) + `" y="16" font-size="11" fill="var(--ink-3)">` +
			escapeText(m[:7]) + `</text>`)
	}
	b.WriteString(marker)
	b.WriteString(lanes.String())
	b.WriteString(`</svg>`)
	return b.String(), pos
}

// axisRange is the first and last day the picture has to reach: every record's
// effective date and every release date, padded to the month boundary at the
// start so the first gridline sits before the first mark.
func (e *explorer) axisRange() (int, int) {
	first, last := 0, 0
	note := func(date string) {
		if date == "" {
			return
		}
		d := dayNumber(date)
		if first == 0 || d < first {
			first = d
		}
		if d > last {
			last = d
		}
	}
	for _, n := range e.export.Nodes {
		note(n.Date)
	}
	for _, r := range e.export.Releases {
		note(r.Date)
	}
	note(e.export.History.FirstCommit)
	if first == 0 && last == 0 {
		return 0, 1
	}
	// Start at the first of that month, so the leading gridline is a boundary.
	y, m, _ := civilFromDays(first)
	return dayNumber(isoDate(y, m, 1)), last + tlTailDays
}

// monthStarts lists the first of each month the axis covers.
func monthStarts(first, last int) []string {
	y, m, _ := civilFromDays(first)
	var out []string
	for {
		d := dayNumber(isoDate(y, m, 1))
		if d > last {
			return out
		}
		if d >= first {
			out = append(out, isoDate(y, m, 1))
		}
		m++
		if m > 12 {
			y, m = y+1, 1
		}
		if len(out) > 600 {
			return out
		}
	}
}

// releaseLane draws the release ticks. Each links its own release on the forge,
// and a label steps further out when it would sit on its neighbour.
func (e *explorer) releaseLane(x func(string) float64, y float64) string {
	rel := append([]changelog.DatedRelease{}, e.export.Releases...)
	sort.SliceStable(rel, func(i, j int) bool { return rel[i].Date < rel[j].Date })
	type slot struct {
		x    float64
		side int
		lvl  int
	}
	var taken []slot
	var b strings.Builder
	b.WriteString(e.laneHead(y-30, e.c.ui.Tiles.Releases, strconv.Itoa(len(rel))))
	b.WriteString(`<line x1="` + strconv.Itoa(tlLeft) + `" y1="` + f1(y) + `" x2="` +
		strconv.Itoa(tlWidth-tlRight) + `" y2="` + f1(y) + `" class="axis"/>`)
	for i, r := range rel {
		xx := x(r.Date)
		side := -1
		if i%2 == 1 {
			side = 1
		}
		lvl := 0
		for {
			clash := false
			for _, s := range taken {
				if s.side == side && s.lvl == lvl && abs(s.x-xx) < 38 {
					clash = true
					break
				}
			}
			if !clash {
				break
			}
			lvl++
		}
		taken = append(taken, slot{xx, side, lvl})
		ly, ty := y+14+float64(lvl)*12, y+26+float64(lvl)*12
		if side < 0 {
			ly, ty = y-14-float64(lvl)*12, y-20-float64(lvl)*12
		}
		open, closeTag := "", ""
		if href := e.releaseHref(r.Version); href != "" {
			open, closeTag = `<a href="`+escapeAttr(href)+`" class="tick">`, `</a>`
		} else {
			open, closeTag = `<g class="tick">`, `</g>`
		}
		b.WriteString(open)
		b.WriteString(`<title>v` + escapeText(r.Version) + ` · ` + escapeText(r.Date) + `</title>`)
		b.WriteString(`<line x1="` + f1(xx) + `" y1="` + f1(min64(y, ly)) + `" x2="` + f1(xx) + `" y2="` +
			f1(max64(y, ly)) + `" stroke="var(--ink)" stroke-width="2"/>`)
		b.WriteString(`<text x="` + f1(xx) + `" y="` + f1(ty) +
			`" font-size="10" text-anchor="middle" fill="var(--ink-2)">v` + escapeText(r.Version) + `</text>`)
		b.WriteString(closeTag)
	}
	return b.String()
}

// releaseHref is a release's own page on the forge.
func (e *explorer) releaseHref(version string) string {
	if e.c.repo.Repository == "" {
		return ""
	}
	return e.c.repo.Repository + "/releases/tag/v" + version
}

// swarmPositions places one store's marks: stacked by day, spilling into
// neighbouring columns when a day is busy, and collapsed onto the day's own
// column when it is too crowded to show one by one.
func swarmPositions(nodes []ExportNode, x func(string) float64, laneY float64, lane tlLane) map[string]tlPoint {
	pos := map[string]tlPoint{}
	for _, day := range groupByDay(nodes) {
		cx := x(day.Date)
		if len(day.Nodes) > lane.Cap {
			for _, n := range day.Nodes {
				pos[n.ID] = tlPoint{cx, laneY}
			}
			continue
		}
		cols := (len(day.Nodes) + lane.PerCol - 1) / lane.PerCol
		sp := lane.R*2 + 1.5
		for k, n := range day.Nodes {
			c := k / lane.PerCol
			row := k % lane.PerCol
			inCol := len(day.Nodes) - c*lane.PerCol
			if inCol > lane.PerCol {
				inCol = lane.PerCol
			}
			px := cx + (float64(c)-float64(cols-1)/2)*sp
			py := laneY + (float64(row)-float64(inCol-1)/2)*sp
			pos[n.ID] = tlPoint{px, py}
		}
	}
	return pos
}

// dayGroup is one day's records, in the record's own order.
type dayGroup struct {
	Date  string
	Nodes []ExportNode
}

// groupByDay buckets a store by effective date, oldest first.
func groupByDay(nodes []ExportNode) []dayGroup {
	byDay := map[string][]ExportNode{}
	for _, n := range nodes {
		byDay[n.Date] = append(byDay[n.Date], n)
	}
	days := make([]string, 0, len(byDay))
	for d := range byDay {
		days = append(days, d)
	}
	sort.Strings(days)
	out := make([]dayGroup, 0, len(days))
	for _, d := range days {
		ns := byDay[d]
		sort.SliceStable(ns, func(i, j int) bool { return handleNum(ns[i].ID) < handleNum(ns[j].ID) })
		out = append(out, dayGroup{Date: d, Nodes: ns})
	}
	return out
}

// dotLane draws one store's marks and its crowded-day capsules.
func (e *explorer) dotLane(x func(string) float64, y float64, lane tlLane, pos map[string]tlPoint) string {
	nodes := e.nodesOfType(lane.Type)
	colour := "var(" + typeColourToken(lane.Type) + ")"
	var b, caps strings.Builder
	b.WriteString(e.laneHead(y-float64(lane.Height)/2+8, e.c.ui.Tiles.ForType(lane.Type), strconv.Itoa(len(nodes))))
	b.WriteString(`<line x1="` + strconv.Itoa(tlLeft) + `" y1="` + f1(y) + `" x2="` +
		strconv.Itoa(tlWidth-tlRight) + `" y2="` + f1(y) + `" class="axis"/>`)
	for _, day := range groupByDay(nodes) {
		if len(day.Nodes) > lane.Cap {
			cx := x(day.Date)
			var ids []string
			for _, n := range day.Nodes {
				ids = append(ids, n.ID)
			}
			caps.WriteString(`<g class="tlcap"><title>` + escapeText(day.Date) + ` · ` +
				strconv.Itoa(len(day.Nodes)) + `&#10;` + escapeText(strings.Join(ids, " · ")) + `</title>`)
			caps.WriteString(`<rect x="` + f1(cx-20) + `" y="` + f1(y-11) +
				`" width="40" height="22" rx="11" fill="` + colour + `" stroke="var(--surface)" stroke-width="1.5"/>`)
			caps.WriteString(`<text x="` + f1(cx) + `" y="` + f1(y+4) +
				`" font-size="11" font-weight="650" text-anchor="middle" fill="var(--surface)">` +
				strconv.Itoa(len(day.Nodes)) + `</text></g>`)
			continue
		}
		for _, n := range day.Nodes {
			p := pos[n.ID]
			b.WriteString(e.mark(n, p, lane.R, colour))
		}
	}
	return b.String() + caps.String()
}

// mark draws one record: a circle, always — never a shape code — with its fill
// carrying its lifecycle and its colour its store.
func (e *explorer) mark(n ExportNode, p tlPoint, r float64, colour string) string {
	fill := lifecycleFill(n.Lifecycle)
	style := `fill="` + colour + `" stroke="var(--surface)"`
	extra := ""
	switch fill {
	case "ring":
		style = `fill="var(--surface)" stroke="` + colour + `"`
	case "dash":
		style = `fill="var(--surface)" stroke="` + colour + `"`
		extra = ` stroke-dasharray="2.5 2"`
	case "fade":
		extra = ` opacity="0.45"`
	}
	// A mark opens the record in the graph; with the graph switched off it
	// opens the record's own page instead, so it never points at a page the
	// site does not have.
	href := "/" + escapeAttr(routeGraph) + "?focus=" + escapeAttr(n.ID)
	if !e.pages.graph {
		href = "/" + escapeAttr(RecordRoute(n))
	}
	return `<a class="tlmark" href="` + href + `">` +
		`<title>` + escapeText(n.ID+" · "+n.Date+" · "+n.Lifecycle) + `&#10;` + escapeText(shortTitle(n)) + `</title>` +
		`<circle cx="` + f1(p.X) + `" cy="` + f1(p.Y) + `" r="` + f1(r) + `" ` + style +
		` stroke-width="1.8"` + extra + `/></a>`
}

// issueLane draws the ledger as a per-day histogram: the total captured that
// day, with the settled part filled in.
func (e *explorer) issueLane(x func(string) float64, y float64, issues []ExportNode) string {
	type tally struct{ total, settled int }
	byDay := map[string]*tally{}
	maxN := 1
	for _, n := range issues {
		t := byDay[n.Date]
		if t == nil {
			t = &tally{}
			byDay[n.Date] = t
		}
		t.total++
		if statusTone(n.Lifecycle) != "open" {
			t.settled++
		}
		if t.total > maxN {
			maxN = t.total
		}
	}
	days := make([]string, 0, len(byDay))
	for d := range byDay {
		days = append(days, d)
	}
	sort.Strings(days)
	var b strings.Builder
	b.WriteString(e.laneHead(y-float64(tlIssueH)-14, e.c.ui.Tiles.Issue, strconv.Itoa(len(issues))))
	b.WriteString(`<line x1="` + strconv.Itoa(tlLeft) + `" y1="` + f1(y) + `" x2="` +
		strconv.Itoa(tlWidth-tlRight) + `" y2="` + f1(y) + `" class="axis"/>`)
	for _, d := range days {
		t := byDay[d]
		h := float64(t.total) / float64(maxN) * tlIssueH
		hs := float64(t.settled) / float64(maxN) * tlIssueH
		xx := x(d) - tlBarWidth/2
		b.WriteString(`<g class="tlbar"><title>` + escapeText(d) + ` · ` + strconv.Itoa(t.total) + ` · ` +
			strconv.Itoa(t.settled) + `</title>`)
		b.WriteString(`<rect x="` + f1(xx) + `" y="` + f1(y-h) + `" width="` + f1(tlBarWidth) + `" height="` +
			f1(h) + `" fill="var(--surface)" stroke="var(--ink-3)" stroke-width="1"/>`)
		b.WriteString(`<rect x="` + f1(xx) + `" y="` + f1(y-hs) + `" width="` + f1(tlBarWidth) + `" height="` +
			f1(hs) + `" fill="var(--ink-3)"/>`)
		b.WriteString(`</g>`)
	}
	return b.String()
}

// arcs draws supersession where both ends exist, and a dashed ×-stub where the
// target has left the tree.
func (e *explorer) arcs(x func(string) float64, pos map[string]tlPoint, relTop float64) string {
	var b strings.Builder
	var drawn []ExportEdge
	for _, ed := range e.export.Edges {
		if ed.Rel == "supersedes" {
			drawn = append(drawn, ed)
		}
	}
	sortEdgesByTarget(drawn, func(ed ExportEdge) string { return ed.From })
	for _, ed := range drawn {
		pa, aok := pos[ed.From]
		pb, bok := pos[ed.To]
		if !aok || !bok {
			continue
		}
		cy := relTop + 40
		if pb.Y > pa.Y+40 {
			cy = (pa.Y + pb.Y) / 2
		}
		mx := (pa.X + pb.X) / 2
		b.WriteString(`<path d="M` + f1(pa.X) + ` ` + f1(pa.Y) + ` Q ` + f1(mx) + ` ` + f1(cy) + ` ` +
			f1(pb.X) + ` ` + f1(pb.Y) + `" fill="none" stroke="var(--ink-2)" stroke-width="1.4" ` +
			`marker-end="url(#tlarr)"><title>` + escapeText(ed.From+" · "+ed.Rel+" · "+ed.To) + `</title></path>`)
	}

	// The stubs. Their angles and lengths come from a fixed table indexed by
	// order, so the same tree draws the same picture on every build.
	var stubs []ExportEdge
	for _, u := range e.export.Health.Unresolved {
		if u.Rel == "supersedes" {
			stubs = append(stubs, u)
		}
	}
	sortEdgesByTarget(stubs, func(ed ExportEdge) string { return ed.From })
	angles := []float64{-150, -112, -78, -42, -170, -20}
	lengths := []float64{48, 64, 52, 68, 90, 56}
	for k, u := range stubs {
		pa, ok := pos[u.From]
		if !ok {
			continue
		}
		ang := angles[k%len(angles)] * math.Pi / 180
		l := lengths[k%len(lengths)]
		ex, ey := pa.X+math.Cos(ang)*l, pa.Y+math.Sin(ang)*l
		anchor, dx := "start", 3.0
		if math.Cos(ang) < -0.2 {
			anchor, dx = "end", -3.0
		}
		b.WriteString(`<g class="tlstub"><title>` + escapeText(u.From+" · "+u.Rel+" · "+u.To+" · "+e.c.ui.Record.NotInTree) + `</title>`)
		b.WriteString(`<path d="M` + f1(pa.X) + ` ` + f1(pa.Y) + ` L` + f1(ex) + ` ` + f1(ey) +
			`" fill="none" stroke="var(--hazard)" stroke-width="1.3" stroke-dasharray="3 2.5"/>`)
		b.WriteString(`<text x="` + f1(ex+dx) + `" y="` + f1(ey+3) + `" font-size="9" text-anchor="` + anchor +
			`" fill="var(--hazard-ink)">× ` + escapeText(u.To) + `</text></g>`)
	}
	return b.String()
}

// laneHead labels one lane with its store's caption and its count.
//
// The count is placed past the caption by measuring it in RUNES: a caption with
// an accent or an em-dash is fewer glyphs than it is bytes, and measuring bytes
// pushes the count away from the label it belongs to.
func (e *explorer) laneHead(y float64, label, count string) string {
	return `<text x="` + strconv.Itoa(tlLeft) + `" y="` + f1(y) +
		`" font-size="11" font-weight="650" fill="var(--ink)">` + escapeText(label) + `</text>` +
		`<text x="` + f1(float64(tlLeft)+8+float64(utf8.RuneCountInString(label))*6.4) + `" y="` + f1(y) +
		`" font-size="10" fill="var(--ink-3)">` + escapeText(count) + `</text>`
}

// supersessionList is the arcs and the stubs as text — the accessible twin of
// the drawing, and the only place the stubs can be read out.
func (e *explorer) supersessionList(pos map[string]tlPoint) string {
	var rows strings.Builder
	n := 0
	var edges []ExportEdge
	for _, ed := range e.export.Edges {
		if ed.Rel == "supersedes" {
			edges = append(edges, ed)
		}
	}
	sortEdgesByTarget(edges, func(ed ExportEdge) string { return ed.From })
	for _, ed := range edges {
		target, ok := e.byID[ed.To]
		if !ok {
			continue
		}
		rows.WriteString(`<li><span class="id"><a href="/` + escapeAttr(e.routeOf(ed.From)) + `">` +
			escapeText(ed.From) + `</a> → <a href="/` + escapeAttr(RecordRoute(target)) + `">` +
			escapeText(ed.To) + `</a></span><span class="small">` + escapeText(shortTitle(target)) + `</span></li>`)
		n++
	}
	var stubs []ExportEdge
	for _, u := range e.export.Health.Unresolved {
		if u.Rel == "supersedes" {
			stubs = append(stubs, u)
		}
	}
	sortEdgesByTarget(stubs, func(ed ExportEdge) string { return ed.From })
	for _, u := range stubs {
		rows.WriteString(`<li><span class="id"><a href="/` + escapeAttr(e.routeOf(u.From)) + `">` +
			escapeText(u.From) + `</a> → <b class="stub">` + escapeText(u.To) + `</b></span>` +
			`<span class="small muted">` + escapeText(e.c.ui.Record.NotInTree) + `</span></li>`)
		n++
	}
	if n == 0 {
		return ""
	}
	return `<ul class="list">` + rows.String() + `</ul>`
}

// --- small numeric helpers -------------------------------------------------

// f1 formats a coordinate at one decimal place. Every number in the SVG goes
// through it, so two builds of one tree produce the same bytes.
func f1(v float64) string { return strconv.FormatFloat(v, 'f', 1, 64) }

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func min64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// civilFromDays is the inverse of dayNumber.
func civilFromDays(z int) (year, month, day int) {
	z += 719468
	era := z / 146097
	if z < 0 {
		era = (z - 146096) / 146097
	}
	doe := z - era*146097
	yoe := (doe - doe/1460 + doe/36524 - doe/146096) / 365
	y := yoe + era*400
	doy := doe - (365*yoe + yoe/4 - yoe/100)
	mp := (5*doy + 2) / 153
	d := doy - (153*mp+2)/5 + 1
	m := mp + 3
	if mp >= 10 {
		m = mp - 9
	}
	if m <= 2 {
		y++
	}
	return y, m, d
}

// isoDate renders a civil date as the record writes one.
func isoDate(y, m, d int) string {
	pad := func(n int) string {
		if n < 10 {
			return "0" + strconv.Itoa(n)
		}
		return strconv.Itoa(n)
	}
	return strconv.Itoa(y) + "-" + pad(m) + "-" + pad(d)
}
