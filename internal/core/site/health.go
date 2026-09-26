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
// link graph, an author against another author's own forge profile, a trailer
// tally against a commit count — and the one family that reads as an opinion
// (the identity candidates) is rendered as a CANDIDATE a human confirms, never
// as an assertion the page makes.
//
// Every visible word is a count, an id, a title, an author name git carries, or
// an interface label from `site-src/ui.json` (adr-47 decision 2). Numbers are
// emitted in their own elements rather than glued into a phrase: a composed
// string is split on decorations only, so "8 commits" is neither a number nor an
// interface string.
//
// Generic side (itd-140): the inputs are the record format and git history. A
// repository with no authorship data renders the families that do have data, and
// one with no findings at all renders the page saying so.

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
	return e.pages.status && (len(e.export.Nodes) > 0 || e.export.Authorship.Commits > 0)
}

// healthPage renders `/record/health/`: one panel per family of finding.
func (e *explorer) healthPage() (string, error) {
	var b strings.Builder
	// `reading` rather than the bare grid: the families are read one at a time
	// and their bodies are wildly different lengths, and a stretched row draws a
	// folded panel as a tall empty box, which reads as a rendering fault rather
	// than as a list a reader has not opened yet.
	// The counts first, as the dashboard states its stores: a reader sees what
	// each check found before opening anything, and a row of zeroes is itself
	// the answer. The lists below are folded, so the page opens as a summary.
	b.WriteString(`<div class="dash">`)
	ui := e.c.ui
	a := e.export.Authorship
	tiles := []struct {
		n     int
		label string
	}{
		{len(e.export.Health.Unresolved), ui.Health.Unresolved},
		{len(e.isolatedRecords()), ui.Health.Isolated},
		{len(e.sameAuthorCandidates()), ui.Health.SameAuthor},
		{a.Undeclared, ui.Health.Undeclared},
		{a.MultiTrailerCommits, ui.Health.MultiTrailer},
	}
	for _, t := range tiles {
		b.WriteString(tile(strconv.Itoa(t.n), t.label, nil))
	}
	b.WriteString(`</div>`)

	b.WriteString(`<div class="dash reading">`)
	b.WriteString(e.healthUnresolved())
	b.WriteString(e.healthIsolated())
	b.WriteString(e.healthSameAuthor())
	b.WriteString(e.healthUndeclared())
	b.WriteString(e.healthMultiTrailer())
	b.WriteString(e.healthSupersedes())
	b.WriteString(`</div>`)
	return e.shell(routeHealth, e.c.ui.RecordNav.Health, "", b.String()), nil
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
		b.WriteString(`<div class="hitem"><span class="hfact"><span class="w">!</span> <a href="/` +
			escapeAttr(e.routeOf(ed.From)) + `">` + escapeText(ed.From) + `</a> → <b class="stub">` +
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
		b.WriteString(`<li><span class="id"><a href="/` + escapeAttr(RecordRoute(n)) + `">` +
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

// --- 3. author identity candidates ----------------------------------------

// authorPair is one CANDIDATE identity fold: two distinct author names that the
// evidence says are one contributor.
//
// It is a candidate and never a finding of fact. The page proposes the `.mailmap`
// line that would fold the pair and stops there: only the contributor knows
// whether two names are one person, and a site that folded them on its own
// evidence would be rewriting the authorship of record from a URL.
type authorPair struct {
	// canonical is the name the fold would keep; alias the name it would retire.
	canonical, alias Author
}

// mailmapLine is the line that would fold the pair — gitmailmap(5)'s first form,
// which rewrites the name recorded against one commit address.
//
// The address it names is always a forge NOREPLY, which carries no mailbox and
// only the public username the profile URL is already derived from. That is what
// keeps this line publishable: the rule the `Author` type states is that a
// contributor's real address stays in git, and a suggestion that broke it would
// republish the very thing the export withholds.
func (p authorPair) mailmapLine() string {
	return p.canonical.Name + " <" + p.alias.email + ">"
}

// sameContributor reports whether the evidence says two author names are one
// person: they resolve to the SAME forge profile, or one author's name is the
// username segment of the other's profile URL.
func sameContributor(a, b Author) bool {
	if a.Profile != "" && a.Profile == b.Profile {
		return true
	}
	return nameIsProfileUser(a.Name, b.Profile) || nameIsProfileUser(b.Name, a.Profile)
}

// nameIsProfileUser reports whether a name IS the username a profile URL names.
// The comparison is case-insensitive: a forge username is, and an author who
// signs one commit `fixture` and another `Fixture` is one person either way.
func nameIsProfileUser(name, profile string) bool {
	if name == "" || profile == "" {
		return false
	}
	user := profile
	if i := strings.LastIndex(profile, "/"); i >= 0 {
		user = profile[i+1:]
	}
	return user != "" && strings.EqualFold(strings.TrimSpace(name), user)
}

// sameAuthorCandidates pairs the author rows the evidence says are one person.
//
// `Humans` is already ordered most commits first, so the earlier row of a pair
// is the one with the stronger claim to be the canonical name. The ALIAS side is
// whichever of the two carries a forge noreply address, because that is the
// address the suggested `.mailmap` line may name; where both do, the alias is
// the row with fewer commits.
func (e *explorer) sameAuthorCandidates() []authorPair {
	humans := e.export.Authorship.Humans
	var out []authorPair
	for i := range humans {
		for j := i + 1; j < len(humans); j++ {
			if !sameContributor(humans[i], humans[j]) {
				continue
			}
			canonical, alias := humans[i], humans[j]
			if profileURL(alias.email) == "" {
				canonical, alias = alias, canonical
			}
			if profileURL(alias.email) == "" {
				// Neither side can be named in a `.mailmap` line without
				// publishing a real address, so there is no suggestion to make
				// and the pair is not raised as one.
				continue
			}
			out = append(out, authorPair{canonical: canonical, alias: alias})
		}
	}
	return out
}

// healthAuthorName renders one side of a candidate pair, linked to the forge
// profile where the address derived one (itd-140: graceful absence otherwise).
func healthAuthorName(a Author) string {
	if a.Profile == "" {
		return escapeText(a.Name)
	}
	return `<a href="` + escapeAttr(a.Profile) + `">` + escapeText(a.Name) + `</a>`
}

// healthSameAuthor raises each candidate fold with the line that would apply it.
func (e *explorer) healthSameAuthor() string {
	ui := e.c.ui
	pairs := e.sameAuthorCandidates()
	if len(pairs) == 0 {
		return panelDisclosure("c12", ui.Health.SameAuthor, "", "0", e.healthClean())
	}
	var b strings.Builder
	b.WriteString(`<div class="health">`)
	for _, p := range pairs {
		b.WriteString(`<div class="hitem"><span class="hfact">` +
			healthAuthorName(p.canonical) + ` · ` + healthAuthorName(p.alias) +
			`</span><span class="hwhy">` + escapeText(ui.Health.Suggestion) + ` <code>` +
			escapeText(p.mailmapLine()) + `</code></span></div>`)
	}
	b.WriteString(`</div>`)
	return panelDisclosure("c12", ui.Health.SameAuthor, "", strconv.Itoa(len(pairs)), b.String())
}

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
	a := e.export.Authorship
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
	a := e.export.Authorship
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
