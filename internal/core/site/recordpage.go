package site

// One page per record: `/record/<type>/<id>/`.
//
// The body is the record's own Markdown, rendered VERBATIM — the interview
// ruled full bodies rather than summaries, and adr-47 decision 3 exempts this
// rendering from the banned-token gate for exactly that reason: the record
// legitimately narrates change and names tools, and rewriting it to look better
// on the site is the thing that decision forbids.
//
// Around the body sit the facts the frontmatter already carries, the typed links
// phrased from THIS record's side, and the two forge links — the file, and its
// commit history, so an amendment is traceable from the date rather than merely
// visible as one.

import (
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// statusTone maps a lifecycle or status word onto the state palette the pills
// are drawn in: in play, done, draft, or set aside. It is the prototype's
// STATUS_TONE, and it is derived from the word rather than from the store, so a
// store the generator has never seen still gets the right colour.
func statusTone(s string) string {
	switch s {
	case "open", "planned", "proposed", "active":
		return "open"
	case "accepted", "shipped", "closed", "resolved":
		return "done"
	case "drafts", "draft":
		return "draft"
	case "superseded", "wontfix":
		return "notplanned"
	}
	return "plain"
}

// severityTone maps the record's severity ladder onto the palette's.
func severityTone(s string) string {
	switch s {
	case "critical":
		return "critical"
	case "major":
		return "high"
	case "minor":
		return "moderate"
	case "nitpick":
		return "low"
	}
	return "plain"
}

// recordPage renders one record.
func (e *explorer) recordPage(n ExportNode) (string, error) {
	ui := e.c.ui
	body, err := e.recordBody(n)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString(`<div class="pills recpills">`)
	// The pill's WORD and its COLOUR say the same thing. A discipline is filed
	// under intents by the record and drawn in its own colour by the chart, so a
	// pill reading "intent" in the discipline colour would have the label and
	// the swatch disagreeing on one badge.
	kind := n.Type
	if n.Lifecycle == disciplinesLifecycle {
		kind = disciplineWord
	}
	b.WriteString(`<span class="pill type" style="--c:var(` + typeColourToken(kind) + `)"><i></i>` +
		escapeText(kind) + `</span>`)
	if state := n.Lifecycle; state != "" {
		b.WriteString(`<span class="pill ` + statusTone(state) + `">` + escapeText(state) + `</span>`)
	}
	if n.Status != "" && n.Status != n.Lifecycle {
		b.WriteString(`<span class="pill ` + statusTone(n.Status) + `">` + escapeText(n.Status) + `</span>`)
	}
	if n.Severity != "" {
		b.WriteString(`<span class="pill ` + severityTone(n.Severity) + `">` + escapeText(n.Severity) + `</span>`)
	}
	if n.Kind != "" && n.Kind != "null" {
		b.WriteString(`<span class="pill plain">` + escapeText(n.Kind) + `</span>`)
	}
	b.WriteString(`<span class="mono id">` + escapeText(n.ID) + `</span>`)
	b.WriteString(`</div>`)

	b.WriteString(`<div class="dash recgrid">`)
	b.WriteString(`<div class="panel c8 recbody"` + srcAttr(n.Path, "") + `>` + body + `</div>`)
	b.WriteString(`<div class="c4 recside">`)
	if n.Derived {
		// The store declares no frontmatter, so there is no frontmatter panel to
		// render. Presenting this record's file name and git dates under that
		// heading would say the record stated them, which it did not; what it
		// has is a file, and the file is what the panel shows.
		b.WriteString(`<div class="panel recfile">` + e.fileLinks(n) + `</div>`)
	} else {
		b.WriteString(panel("", ui.Record.Frontmatter, "", e.frontmatterTable(n)))
	}
	if links := e.linkPanels(n); links != "" {
		b.WriteString(links)
	}
	b.WriteString(`</div>`)
	b.WriteString(`</div>`)

	return e.shell(RecordRoute(n), shortTitle(n), "", b.String()), nil
}

// recordBody renders the record's Markdown, with the first use of each glossary
// term linked to its entry.
func (e *explorer) recordBody(n ExportNode) (string, error) {
	body, err := e.renderMarkdownBody(n.Path)
	if err != nil {
		return "", err
	}
	return e.linkTerms(body, ""), nil
}

// renderMarkdownBody renders one repository Markdown file as page body.
//
// The H1 is dropped because the page already carries it as its heading; every
// other heading keeps its own level and its anchor, so a link into the middle of
// a record still lands where it points.
//
// A record's page and a glossary entry's page are the same rendering of the same
// kind of file, so they share it: a second copy would be a second answer to what
// a repo-relative link becomes and which images are assets.
func (e *explorer) renderMarkdownBody(rel string) (string, error) {
	data, err := fsutil.ReadGuardedInRoot(e.c.root, rel, maxRecordBodyBytes)
	if err != nil {
		return "", err
	}
	text, consumed := StripFrontmatter(string(data))
	secs, err := Sections(rel, text, consumed)
	if err != nil {
		return "", err
	}
	dir := path.Dir(rel)
	r := &Renderer{
		UI:    e.c.ui,
		Refs:  LinkDefinitions(text),
		Image: func(src, alt string, at Source) (string, error) { return e.c.assets.render(dir, src, alt, at) },
		Link:  func(href string, at Source) string { return e.href(rel, href) },
	}
	var b strings.Builder
	for _, s := range secs {
		if s.Level > 1 {
			level := s.Level
			if level > 6 {
				level = 6
			}
			tag := "h" + string(rune('0'+level))
			inner, err := r.inline(Source{Path: rel, Line: s.Line}, s.Title)
			if err != nil {
				return "", err
			}
			b.WriteString("<" + tag + ` id="` + escapeAttr(s.Anchor) + `">` + inner + "</" + tag + ">")
		}
		h, err := r.RenderBlocks(rel, Blocks(s.Body, s.BodyLine))
		if err != nil {
			return "", err
		}
		b.WriteString(h)
	}
	return b.String(), nil
}

// The record's own field names. They are the record format's words, not the
// generator's, so they are shared rather than repeated: the frontmatter table
// prints them and the chart's card names its dates by them, and the two cannot
// come to call the same fact by two different names.
const (
	fieldDate    = "date"
	fieldCreated = "created"
	fieldEntered = "entered"
	fieldTouched = "touched"
)

// frontmatterTable is the record's own fields, plus the three dates git carries
// for its file. Every cell is a value the record already holds.
func (e *explorer) frontmatterTable(n ExportNode) string {
	rows := [][2]string{
		{"id", n.ID},
		{"type", n.Type},
		{"lifecycle", n.Lifecycle},
		{"status", n.Status},
		{"kind", n.Kind},
		{"severity", n.Severity},
		{fieldDate, n.Date},
		{fieldCreated, n.Dates.Created},
		{fieldEntered, n.Dates.Entered},
		{fieldTouched, n.Dates.Touched},
	}
	var b strings.Builder
	b.WriteString(`<div class="tablewrap"><table class="fm"><tbody>`)
	for _, r := range rows {
		if r[1] == "" || r[1] == "null" {
			continue
		}
		b.WriteString(`<tr><th scope="row">` + escapeText(r[0]) + `</th><td>` + escapeText(r[1]) + `</td></tr>`)
	}
	b.WriteString(`</tbody></table></div>`)
	b.WriteString(e.fileLinks(n))
	return b.String()
}

// fileLinks names the record's file and offers the two views of it the forge
// serves: the file itself, and its commit history — which is what makes an
// amendment traceable from a date rather than merely visible as one.
//
// The path is a file name, which the generator may print; everything else here
// is a ui.json label.
func (e *explorer) fileLinks(n ExportNode) string {
	return e.forgeFileLinks(n.Path, n.Dates.Touched != "")
}

// forgeFileLinks is fileLinks for any repository file the site puts a page on —
// a record's, or a glossary entry's. history says whether the commit-history
// link is offered: a record whose file git has never touched has no history to
// show, and offering the link anyway would promise a page that is empty.
func (e *explorer) forgeFileLinks(rel string, history bool) string {
	blob := e.forgeBlob(rel)
	if blob == "" {
		return `<p class="reclinks mono">` + escapeText(rel) + `</p>`
	}
	var b strings.Builder
	b.WriteString(`<p class="reclinks"><span class="mono">` + escapeText(rel) + `</span><br>`)
	b.WriteString(`<a href="` + escapeAttr(blob) + `">` + escapeText(e.c.ui.Record.OpenOnForge) + ` ↗</a>`)
	if commits := e.forgeCommits(rel); commits != "" && history {
		b.WriteString(` · <a href="` + escapeAttr(commits) + `">` +
			escapeText(e.c.ui.Record.CommitHistory) + ` ↗</a>`)
	}
	b.WriteString(`</p>`)
	return b.String()
}

// linkPanels renders the record's cross-references, each phrased from THIS
// record's side: an outbound link says what this record does, an inbound one
// says what was done to it.
func (e *explorer) linkPanels(n ExportNode) string {
	ui := e.c.ui
	var out strings.Builder

	outbound := append([]ExportEdge{}, e.out[n.ID]...)
	sortEdgesByTarget(outbound, func(ed ExportEdge) string { return ed.To })
	stubs := append([]ExportEdge{}, e.stubs[n.ID]...)
	sortEdgesByTarget(stubs, func(ed ExportEdge) string { return ed.To })
	if len(outbound)+len(stubs) > 0 {
		var l strings.Builder
		l.WriteString(`<ul class="links">`)
		for _, ed := range outbound {
			l.WriteString(e.linkRow(ed.To, relationWord(ed.Rel)))
		}
		for _, ed := range stubs {
			// The target has left the tree. It is rendered as the absence it is
			// — dashed, unlinked, and labelled — never as a link to nothing.
			l.WriteString(`<li class="stubrow"><span class="id stub">` + escapeText(ed.To) + `</span>` +
				`<span class="muted">` + escapeText(ui.Record.NotInTree) + `</span>` +
				`<span class="rel">` + escapeText(relationWord(ed.Rel)) + `</span></li>`)
		}
		l.WriteString(`</ul>`)
		out.WriteString(panel("", ui.Record.Outbound, itoaLen(len(outbound)+len(stubs)), l.String()))
	}

	inbound := append([]ExportEdge{}, e.in[n.ID]...)
	sortEdgesByTarget(inbound, func(ed ExportEdge) string { return ed.From })
	if len(inbound) > 0 {
		var l strings.Builder
		l.WriteString(`<ul class="links">`)
		for _, ed := range inbound {
			l.WriteString(e.linkRow(ed.From, ui.Relations.Inverse(ed.Rel)))
		}
		l.WriteString(`</ul>`)
		out.WriteString(panel("", ui.Record.Inbound, itoaLen(len(inbound)), l.String()))
	}

	if ms := e.mentions[n.ID]; len(ms) > 0 {
		ids := append([]string{}, ms...)
		sort.SliceStable(ids, func(i, j int) bool { return lint.HandleLess(ids[i], ids[j]) })
		var l strings.Builder
		l.WriteString(`<ul class="links">`)
		for _, id := range ids {
			l.WriteString(e.linkRow(id, ""))
		}
		l.WriteString(`</ul>`)
		out.WriteString(panel("", ui.Record.Mentions, itoaLen(len(ids)), l.String()))
	}
	return out.String()
}

// linkRow is one cross-reference: the target's id and title, and the relation as
// this record reads it.
func (e *explorer) linkRow(id, rel string) string {
	target, ok := e.byID[id]
	if !ok {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<li><a class="id" href="/` + escapeAttr(RecordRoute(target)) + `">` + escapeText(id) + `</a>`)
	b.WriteString(`<span>` + escapeText(shortTitle(target)) + `</span>`)
	if rel != "" {
		b.WriteString(`<span class="rel">` + escapeText(rel) + `</span>`)
	}
	b.WriteString(`</li>`)
	return b.String()
}

// sortEdgesByTarget orders a record's links by the handle at the far end, so the
// page is a function of the record rather than of the walk's arrival order.
func sortEdgesByTarget(edges []ExportEdge, key func(ExportEdge) string) {
	sort.SliceStable(edges, func(i, j int) bool {
		a, b := key(edges[i]), key(edges[j])
		if a != b {
			return lint.HandleLess(a, b)
		}
		return edges[i].Rel < edges[j].Rel
	})
}

// typeColourToken is the stylesheet token a store's bubbles and pills take.
// Colour names only the durable types; everything else is neutral, because a
// fourth categorical hue stops being distinguishable under colour-vision
// deficiency in an all-pairs chart.
// disciplineWord is the singular a badge uses: the record's directory is
// plural, and a badge names one record.
const disciplineWord = "discipline"

func typeColourToken(typ string) string {
	switch typ {
	case disciplinesLifecycle, disciplineWord:
		// A discipline is filed under intents by the record and is its own kind
		// of thing to a reader, so it carries its own colour rather than an
		// intent's. The chart agrees: `record.js` reads the same rule.
		return "--s-discipline"
	case "adr":
		return "--s-adr"
	case "intent":
		return "--s-intent"
	case "spec":
		return "--s-spec"
	case "issue":
		return "--s-issue"
	}
	return "--s-neutral"
}

func itoaLen(n int) string {
	if n == 0 {
		return ""
	}
	return strconv.Itoa(n)
}
