package site

// The decision log — `/record/decision-log/`.
//
// Every decision record on one page (itd-2609151138388536,
// spc-2610110931381316 step 5). A decision is one paragraph written once and
// never moved, and the record holds hundreds of them, so they are not dealt
// onto the development page or drawn on the relationship chart, both of which
// are built for the stores that move and which hundreds of one-line nodes
// would bury. They are read here instead, newest first: the record can hold
// years of decisions, and the recent ones are the ones a reader comes for
// (the product thinker's ruling of 2026-10-15).
//
// The page has no cap. The intent's criterion is that every entry is listed,
// so a list cut the way the development page cuts a bucket would fail it. The
// rows are grouped by the month of their date, each month a disclosure whose
// heading is the month and its count, and every row is in the HTML whether its
// month is open or not, so a reader's find-in-page reaches all of them. The
// newest month comes first and opens, as the place a reader arriving at the
// log looks first.
//
// A row is the record's id, linking the record's own page, its date, and its
// full body rendered from the record file, which is the record's words and
// carries that file as its source. The page exists only where the repository
// opts in to publishing the decisions (RecordOpts.Decisions) and holds at
// least one; otherwise it and its navigation entry are omitted (itd-140).

import (
	"strconv"
	"strings"
)

// hasDecisionLog reports whether the export carries any decision to list.
func (e *explorer) hasDecisionLog() bool { return len(e.export.Decisions) > 0 }

// decisionMonth is one month's decisions, newest first.
type decisionMonth struct {
	month string
	nodes []ExportNode
}

// decisionMonths groups the decisions by the month of their date, newest
// first. The export holds them oldest first, in id order, so walking it
// backwards puts the months and the rows within each in newest-first order.
func decisionMonths(nodes []ExportNode) []decisionMonth {
	var out []decisionMonth
	for i := len(nodes) - 1; i >= 0; i-- {
		n := nodes[i]
		m := n.Date
		if len(m) >= len("YYYY-MM") {
			m = m[:len("YYYY-MM")]
		}
		if len(out) == 0 || out[len(out)-1].month != m {
			out = append(out, decisionMonth{month: m})
		}
		out[len(out)-1].nodes = append(out[len(out)-1].nodes, n)
	}
	return out
}

// decisionLogPage renders `/record/decision-log/`.
func (e *explorer) decisionLogPage() (string, error) {
	months := decisionMonths(e.export.Decisions)
	var b strings.Builder
	b.WriteString(`<div class="dash">`)
	for i, m := range months {
		var l strings.Builder
		l.WriteString(`<ul class="list declog">`)
		for _, n := range m.nodes {
			body, err := e.renderMarkdownBody(n.Path)
			if err != nil {
				return "", err
			}
			l.WriteString(`<li><span class="id"><a href="/` + escapeAttr(RecordRoute(n)) + `">` +
				escapeText(n.ID) + `</a>`)
			if n.Date != "" {
				l.WriteString(`<span class="d">` + escapeText(n.Date) + `</span>`)
			}
			l.WriteString(`</span><div class="decbody"` + srcAttr(n.Path, "") + `>` + body + `</div></li>`)
		}
		l.WriteString(`</ul>`)
		b.WriteString(panelFold("c12", m.month, "", strconv.Itoa(len(m.nodes)), l.String(), i == 0))
	}
	b.WriteString(`</div>`)
	return e.shell(routeDecisionLog, e.c.ui.RecordNav.DecisionLog, "", b.String()), nil
}
