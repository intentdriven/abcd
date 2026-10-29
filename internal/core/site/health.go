package site

// The health page — `/record/health/`.
//
// It collects every finding the record can be checked against ITSELF for. Each
// family is one panel: its count in the panel note, its findings in the body.
// A family with nothing to report renders `ui.Health.Clean` rather than being
// omitted, because a page that hides its empty checks cannot tell a reader the
// difference between a check that passed and a check that never ran.
//
// Nothing here is a judgement. Every family is a mechanical comparison of the
// record against itself — a reference against the tree, a record against the
// link graph, a trailer tally against a commit count.
//
// Every visible word is a count, an id, a title, or an interface label from
// `site-src/ui.json` (adr-47 decision 2). No person is named: the page names no
// author (adr-2610291037481588). Numbers are
// emitted in their own elements rather than glued into a phrase: a composed
// string is split on decorations only, so "8 commits" is neither a number nor an
// interface string.
//
// Generic side (itd-140): the inputs are the record format and git history. A
// repository with no authorship data renders the families that do have data, and
// one with no findings at all renders the page saying so. A project site reads
// no git history, so it renders the record's own families alone.

import (
	"strconv"
	"strings"
)

// isolatedListCap bounds the isolated-record list. A repository early in its
// life can have more unlinked records than linked ones, and a panel listing all
// of them is a wall rather than a finding. The panel note carries the TRUE
// total and the list says how many it did not draw, so the cap is visible
// arithmetic rather than a silent truncation.
const isolatedListCap = 20

// hasHealth reports whether the record has anything to check itself against.
// Without records and without a history there is no check to run, and the page
// and its navigation entry are omitted (itd-140: graceful absence).
func (e *explorer) hasHealth() bool {
	return e.pages.status && (len(e.export.Nodes) > 0 || e.export.authorship().Commits > 0)
}

// healthPage renders `/record/health/`: one panel per family of finding.
func (e *explorer) healthPage() (string, error) {
	var b strings.Builder
	// The page is the site's Status page (itd-2609061543533170 decision 7), so
	// it opens with the Now / Next / Later block (itd-2609212103568351), then
	// the checks the record is held to.
	b.WriteString(e.statusSection())
	// `reading` rather than the bare grid: the families are read one at a time
	// and their bodies are wildly different lengths, and a stretched row draws a
	// folded panel as a tall empty box, which reads as a rendering fault rather
	// than as a list a reader has not opened yet.
	// The counts first, as the dashboard states its stores: a reader sees what
	// each check found before opening anything, and a row of zeroes is itself
	// the answer. The lists below are folded, so the page opens as a summary.
	b.WriteString(`<div class="dash">`)
	ui := e.c.ui
	a := e.export.authorship()
	type count struct {
		n     int
		label string
	}
	tiles := []count{
		{len(e.export.Health.Unresolved), ui.Health.Unresolved},
		{len(e.isolatedRecords()), ui.Health.Isolated},
	}
	// The authorship families are answered by git history, which a project
	// site does not read (project.go): their tiles, the disclosure rate and
	// their panels are left out there, and the checks the record makes of
	// itself stay.
	gitFamilies := !e.c.project
	if gitFamilies {
		tiles = append(tiles,
			count{a.Undeclared, ui.Health.Undeclared},
			count{a.MultiTrailerCommits, ui.Health.MultiTrailer})
	}
	for _, t := range tiles {
		b.WriteString(tile(strconv.Itoa(t.n), t.label, nil))
	}
	if gitFamilies {
		b.WriteString(e.healthDisclosureRate())
	}
	b.WriteString(`</div>`)

	b.WriteString(`<div class="dash reading">`)
	b.WriteString(e.healthUnresolved())
	b.WriteString(e.healthIsolated())
	if gitFamilies {
		b.WriteString(e.healthUndeclared())
		b.WriteString(e.healthMultiTrailer())
	}
	b.WriteString(e.healthSupersedes())
	b.WriteString(`</div>`)
	return e.shell(routeHealth, e.c.ui.RecordNav.Health, "", b.String()), nil
}

// healthDisclosureRate is the share of authored commits that disclose AI
// assistance, the last tile in the row of counts. The contributors page carries
// the authors and the trailers alone; the rate sits here, with the findings the
// trailer tally is checked against (iss-2608231008315498).
//
// The rate is COMMITS that disclose over commits a person WROTE. Merges are in
// neither: the forge writes them, no convention asks them to declare anything,
// and leaving them in the denominator understated this rate by more than twenty
// points. The excluded count is shown, not assumed. A history with no authored
// commit has no rate, and renders no tile.
func (e *explorer) healthDisclosureRate() string {
	ui := e.c.ui
	a := e.export.authorship()
	if a.Authored == 0 {
		return ""
	}
	share := strconv.Itoa(a.AssistedCommits*100/a.Authored) + "%"
	// The count is its own element rather than part of the sentence: the
	// provenance walk splits composed text on decorations only, and a number
	// glued to a phrase is neither a number nor an interface string.
	var excl string
	if a.Merges > 0 {
		excl = `<span class="s"><b class="tnum">` + strconv.Itoa(a.Merges) + `</b> ` +
			escapeText(ui.Contributors.MergesExcluded) + `</span>`
	}
	return tileExtra(share, ui.Contributors.Assisted,
		[]string{strconv.Itoa(a.AssistedCommits) + " / " + strconv.Itoa(a.Authored)}, excl)
}

// healthSupersedes reads the supersession edges as text: which record replaced
// which, and which replacement points at something the tree no longer holds.
//
// It sits LAST because it is the one family that is not a fault — a
// supersession is the record working — and it carries a line saying so, for the
// same reason the multi-trailer family does: a panel among findings that says
// nothing about itself is read as another finding.
func (e *explorer) healthSupersedes() string {
	ui := e.c.ui
	body, n := e.supersessions()
	lead := `<p class="small muted foldlead">` + escapeText(ui.Health.SupersedesLead) + `</p>`
	if n == 0 {
		return panelDisclosure("c12", relationWord("supersedes"), "", "0", lead+e.healthClean())
	}
	return panelDisclosure("c12", relationWord("supersedes"), "", strconv.Itoa(n), lead+body)
}

// healthClean is the body of a family with nothing to report.
func (e *explorer) healthClean() string {
	return `<div class="health"><div class="hsum">` + escapeText(e.c.ui.Health.Clean) + `</div></div>`
}

// --- 1. unresolved typed references ---------------------------------------

// healthUnresolved lists every typed reference no file in the tree answers to.
// It is the dashboard's own finding at full length, in the same two-line shape:
// the fact, then the explanation beneath it.
func (e *explorer) healthUnresolved() string {
	ui := e.c.ui
	found := e.export.Health.Unresolved
	if len(found) == 0 {
		return panelDisclosure("c12", ui.Health.Unresolved, "", "0", e.healthClean())
	}
	var b strings.Builder
	b.WriteString(`<div class="health">`)
	for _, ed := range found {
		b.WriteString(`<div class="hitem"><span class="hfact"><span class="w">!</span> <a href="` +
			escapeAttr(e.c.at("/"+e.routeOf(ed.From))) + `">` + escapeText(ed.From) + `</a> → <b class="stub">` +
			escapeText(ed.To) + `</b></span><span class="hwhy">` +
			escapeText(relationWord(ed.Rel)) + ` ` + escapeText(ui.Record.NotInTree) + `</span></div>`)
	}
	b.WriteString(`</div>`)
	return panelDisclosure("c12", ui.Health.Unresolved, "", strconv.Itoa(len(found)), b.String())
}

// --- 2. isolated records ---------------------------------------------------

// isolatedRecords is every record nothing reaches and which reaches nothing.
//
// It asks the SAME question the chart does. The export publishes a count
// (`Layout.Isolated`) taken from a bubble's weighted degree, which body
// mentions contribute to; counting typed links alone here would put a different
// number under near-identical wording on two pages of one site, which is the
// kind of quiet disagreement this page exists to find. A record named in
// another's prose is reached by it, so a mention counts — and the records
// listed here are exactly the ones drawn smallest on the relationship chart.
//
// The walk is over `e.export.Nodes` rather than over `e.byID`: they hold the
// same records, and only the slice has an order a golden file can rest on.
func (e *explorer) isolatedRecords() []ExportNode {
	out := make([]ExportNode, 0, len(e.export.Nodes))
	for _, n := range e.export.Nodes {
		if len(e.out[n.ID]) == 0 && len(e.in[n.ID]) == 0 && len(e.mentions[n.ID]) == 0 {
			out = append(out, n)
		}
	}
	return out
}

// healthIsolated lists the records nothing links to and which link to nothing.
// The list folds away: it is the one family whose length is a function of how
// young the record is rather than of how wrong it is.
func (e *explorer) healthIsolated() string {
	ui := e.c.ui
	iso := e.isolatedRecords()
	if len(iso) == 0 {
		return panelDisclosure("c12", ui.Health.Isolated, "", "0", e.healthClean())
	}
	shown := iso
	if len(shown) > isolatedListCap {
		shown = shown[:isolatedListCap]
	}
	var b strings.Builder
	b.WriteString(`<ul class="list">`)
	for _, n := range shown {
		b.WriteString(`<li><span class="id"><a href="` + escapeAttr(e.c.at("/"+RecordRoute(n))) + `">` +
			escapeText(n.ID) + `</a></span><span>` + escapeText(shortTitle(n)) + `</span></li>`)
	}
	b.WriteString(`</ul>`)
	// The remainder is stated rather than dropped, and its number is its own
	// element so the count and the word are never one composed phrase.
	if rest := len(iso) - len(shown); rest > 0 {
		b.WriteString(`<div class="health"><div class="hsum"><b class="tnum">` + strconv.Itoa(rest) +
			`</b> ` + escapeText(ui.More) + `</div></div>`)
	}
	return panelDisclosure("c12", ui.Health.Isolated, "", strconv.Itoa(len(iso)), b.String())
}

// --- 3. author identity candidates (retired) -----------------------------
//
// The page once raised two author names the evidence said were one person,
// with the `.mailmap` line that would fold them. Both names, the profile each
// linked and the forge login in the suggested line are a person named on a
// published page, which the site no longer does (adr-2610291037481588). The
// fold the panel proposed is now applied where the people are counted
// (humanKey), so nothing is left for it to report. Its two interface strings
// stay declared in ui.json, and so in HealthUI, because the file is decoded
// with unknown fields refused and a managed repository's ui.json that declares
// them must still build.

// --- 4. authored commits declaring nothing --------------------------------

// healthUndeclared states how many authored commits carry no `Assisted-by:`
// trailer at all, against the number that could have carried one.
//
// It is a COUNT and not a list. The individual commits are not in anything this
// build has already loaded: `LoadAuthorship` folds the trailer walk into tallies
// as it reads and keeps no per-commit rows, so listing them would mean a second
// `git log` — a new invocation on every build of every managed repository, for a
// list whose length is the whole point of the number. The count is the finding.
func (e *explorer) healthUndeclared() string {
	ui := e.c.ui
	a := e.export.authorship()
	if a.Undeclared == 0 {
		return panelDisclosure("c12", ui.Health.Undeclared, "", "0", e.healthClean())
	}
	var b strings.Builder
	b.WriteString(`<div class="health"><div class="hsum">`)
	b.WriteString(`<b class="tnum">` + strconv.Itoa(a.Undeclared) + `</b> ` +
		escapeText(ui.Contributors.Undeclared))
	b.WriteString(` · <b class="tnum">` + strconv.Itoa(a.Authored) + `</b> ` +
		escapeText(ui.Tiles.Commits))
	// Merges are already out of the numerator and the denominator both; the page
	// says how many, so the reader never has to assume which.
	if a.Merges > 0 {
		b.WriteString(` · <b class="tnum">` + strconv.Itoa(a.Merges) + `</b> ` +
			escapeText(ui.Contributors.MergesExcluded))
	}
	b.WriteString(`</div></div>`)
	return panelDisclosure("c12", ui.Health.Undeclared, "", strconv.Itoa(a.Undeclared), b.String())
}

// --- 5. commits declaring more than one model -----------------------------

// healthMultiTrailer states the gap between the trailer OCCURRENCES and the
// commits that carry them.
//
// This family is NOT a defect and the panel says so in its own rendered text: a
// commit may legitimately declare two assisting models, and each such commit is
// counted twice in one tally and once in the other. The gap is published because
// two published numbers that differ with no stated reason read as a fault in one
// of them.
func (e *explorer) healthMultiTrailer() string {
	ui := e.c.ui
	a := e.export.authorship()
	// The COMMIT count decides whether the family has anything to report; the
	// trailer and commit totals below are the two figures that differ because
	// of it, and the separator between them goes the way the others did.
	n := a.MultiTrailerCommits
	var b strings.Builder
	b.WriteString(`<div class="health">`)
	if n == 0 {
		b.WriteString(`<div class="hsum">` + escapeText(ui.Health.Clean) + `</div>`)
	} else {
		b.WriteString(`<div class="hsum"><b class="tnum">` + strconv.Itoa(a.Assisted) + `</b> ` +
			escapeText(ui.Contributors.Trailers) + ` <b class="tnum">` +
			strconv.Itoa(a.AssistedCommits) + `</b> ` + escapeText(ui.Tiles.Commits) + `</div>`)
	}
	b.WriteString(`<div class="hwhy">` + escapeText(ui.Health.NotADefect) + `</div>`)
	b.WriteString(`</div>`)
	return panelDisclosure("c12", ui.Health.MultiTrailer, "", strconv.Itoa(n), b.String())
}
