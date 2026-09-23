package release

// page.go — the release page: `RELEASE.md` at the repository root, composed at
// the cut from the press releases of the user-facing intents the cut shipped.
//
// It is the changelog slice's trust model applied to a second document. The
// composer owns the WORDING of the headline paragraphs and which intents are
// told rather than listed; the core owns the SET the page must cite
// (changelog.RecordSet.PressReleaseRequired, carried per entry as
// in_press_release), the heading, the name list's text (each listed intent's
// own title), the citation suffixes, and the check that every quote is carried
// word for word from the intent it is attributed to. A page that fails any of
// it is refused whole, before anything is written.

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/termsafe"
)

// PageFile is the release page, at the repository root beside CHANGELOG.md.
const PageFile = "RELEASE.md"

// ArchiveDir is where a replaced page moves, named after the release its heading
// names. It is the permanent-record tier: the pages are written by the cut,
// never by hand.
const ArchiveDir = ".abcd/development/releases"

const (
	// maxHeadlines and maxQuotes bound the page's lists. A release is scoped at
	// about twenty intents; fifty of either is not a page, it is a hostile payload.
	maxHeadlines = 50
	maxQuotes    = 50
	// maxPageBytes bounds the read of an outgoing page.
	maxPageBytes = 1 << 20
)

// PressReleasePayload is the page half of the composed payload. It is absent
// (or null) exactly when the cut's press-release set is empty.
type PressReleasePayload struct {
	// Headlines are the intents told as prose, one paragraph each.
	Headlines []Headline `json:"headlines"`
	// Listed are the ids of every other intent in the set. The binary renders
	// each as its record's title, so the name list carries no composer prose.
	Listed []string `json:"listed"`
	// Quotes are persona quotes carried word for word from the press release of
	// an intent a headline tells.
	Quotes []Quote `json:"quotes"`
}

// Headline is one prose paragraph and the intents it tells.
type Headline struct {
	Records []string `json:"records"`
	// Text is the wording only; the citation suffix is the core's.
	Text string `json:"text"`
}

// Quote is one persona quote, verified against its source before it is written.
type Quote struct {
	// Record is the intent whose press release carries the quote; it must be a
	// headline record.
	Record string `json:"record"`
	// Text is the quote as the source has it, attribution included.
	Text string `json:"text"`
	// Attribution is the speaker as the source names them; it must appear in
	// Text, so it is carried exactly as the source has it.
	Attribution string `json:"attribution"`
}

// PageResult reports what the cut did with the release page.
type PageResult struct {
	Written bool   `json:"written"`
	Path    string `json:"path,omitempty"`
	Heading string `json:"heading,omitempty"`
	// Archived is the path the outgoing page moved to; empty when nothing moved.
	Archived  string `json:"archived,omitempty"`
	Headlines int    `json:"headlines"`
	Listed    int    `json:"listed"`
	Quotes    int    `json:"quotes"`
	// Reason says why no page was written, empty when one was.
	Reason string `json:"reason,omitempty"`
}

// pageHeadingRe is the one reader of a page heading: it names the archive file
// of an outgoing page, and it proves a rendered page names the cut's version.
var pageHeadingRe = regexp.MustCompile(`^# Release (\d+\.\d+\.\d+) \((\d{4}-\d{2}-\d{2})\)$`)

// fenceRe matches a code fence opener anywhere in a text.
var fenceRe = regexp.MustCompile("```|~~~")

// pageHeading renders the heading: the bare version, as the changelog heading
// carries it, and the same clock's date in UTC.
func pageHeading(nextTag string, at time.Time) string {
	return fmt.Sprintf("# Release %s (%s)", strings.TrimPrefix(nextTag, "v"), at.UTC().Format("2006-01-02"))
}

// parsePageHeading reads the version and date off a page's first non-blank line.
func parsePageHeading(page string) (version, date string, ok bool) {
	for _, line := range strings.Split(page, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		m := pageHeadingRe.FindStringSubmatch(line)
		if m == nil {
			return "", "", false
		}
		return m[1], m[2], true
	}
	return "", "", false
}

// validatedPage is a page that passed every check, ready to render.
type validatedPage struct {
	headlines []Headline
	quotes    []Quote
	// listed are the listed entries, in payload order.
	listed []Entry
}

// pageSet returns the cut's press-release set, keyed by id, in cut order.
func pageSet(cut Cut) ([]Entry, map[string]Entry) {
	var order []Entry
	byID := map[string]Entry{}
	for _, e := range cut.Added {
		if e.InPressRelease {
			order = append(order, e)
			byID[e.ID] = e
		}
	}
	return order, byID
}

// validatePage checks the page half of a payload against the cut, collecting
// every fault it finds.
func validatePage(cut Cut, p *PressReleasePayload, rs *reasons) validatedPage {
	set, inSet := pageSet(cut)
	if len(set) == 0 {
		if p != nil {
			rs.add(ReasonPageForEmptySet, "press_release",
				"no user-facing intent shipped in this cut, so there is no release page; send press_release as null")
		}
		return validatedPage{}
	}
	if p == nil {
		p = &PressReleasePayload{}
	}

	var out validatedPage
	cited := map[string]string{} // id -> the payload path that cited it
	headlineRecords := map[string]bool{}
	cite := func(id, at string) bool {
		if !checkID(id, at, rs) {
			return false
		}
		if first, dup := cited[id]; dup {
			rs.add(ReasonDuplicateCitation, at, "%s is already cited at %s; each intent is told or listed once", id, first)
			return false
		}
		cited[id] = at
		if _, ok := inSet[id]; !ok {
			rs.add(ReasonOutsideSet, at, "%s is not in the release page's set: %s", id, outsideCause(cut, id))
			return false
		}
		return true
	}

	if len(p.Headlines) == 0 {
		rs.add(ReasonNoHeadline, "press_release.headlines",
			"the set holds %d intent(s) and the page tells none of them; write at least one headline", len(set))
	}
	if len(p.Headlines) > maxHeadlines {
		rs.add(ReasonTextOversize, "press_release.headlines", "%d headlines (max %d)", len(p.Headlines), maxHeadlines)
	}
	for i, h := range p.Headlines {
		at := fmt.Sprintf("press_release.headlines[%d]", i)
		if len(h.Records) == 0 {
			rs.add(ReasonNoCitation, at+".records", "the headline cites no intent")
		}
		if len(h.Records) > maxRecordsPerEntry {
			rs.add(ReasonTextOversize, at+".records", "%d records on one headline (max %d)", len(h.Records), maxRecordsPerEntry)
		}
		var ids []string
		for j, id := range h.Records {
			if cite(id, fmt.Sprintf("%s.records[%d]", at, j)) {
				ids = append(ids, id)
				headlineRecords[id] = true
			}
		}
		text, ok := checkProse(h.Text, at+".text", rs)
		if ok && text == "" {
			rs.add(ReasonEmptyProse, at+".text", "the headline citing %s has no prose", strings.Join(ids, ", "))
		}
		out.headlines = append(out.headlines, Headline{Records: ids, Text: text})
	}
	for i, id := range p.Listed {
		if cite(id, fmt.Sprintf("press_release.listed[%d]", i)) {
			out.listed = append(out.listed, inSet[id])
		}
	}
	for _, e := range set {
		if _, ok := cited[e.ID]; !ok {
			rs.add(ReasonMissing, "press_release", "%s shipped in this cut and the page neither tells nor lists it", e.ID)
		}
	}

	if len(p.Quotes) > maxQuotes {
		rs.add(ReasonTextOversize, "press_release.quotes", "%d quotes (max %d)", len(p.Quotes), maxQuotes)
	}
	for i, q := range p.Quotes {
		at := fmt.Sprintf("press_release.quotes[%d]", i)
		if !checkID(q.Record, at+".record", rs) {
			continue
		}
		textOK := len(q.Text) <= maxEntryProseBytes
		if !textOK {
			rs.add(ReasonTextOversize, at+".text", "a %d-byte quote (max %d)", len(q.Text), maxEntryProseBytes)
		}
		if len(q.Attribution) > maxEntryProseBytes {
			rs.add(ReasonTextOversize, at+".attribution", "a %d-byte attribution (max %d)", len(q.Attribution), maxEntryProseBytes)
			textOK = false
		}
		if !headlineRecords[q.Record] {
			rs.add(ReasonQuoteSource, at+".record",
				"%s is not told by a headline; a quote is carried only from an intent the page tells", q.Record)
			continue
		}
		if !textOK {
			continue
		}
		if !structureSafe(q.Text, at+".text", rs) {
			continue
		}
		if why := verbatim(inSet[q.Record].pressRelease, q); why != "" {
			rs.add(ReasonQuoteNotVerbatim, at, "the quote attributed to %s is not carried word for word from its press release: %s",
				q.Record, why)
			continue
		}
		out.quotes = append(out.quotes, Quote{Record: q.Record, Text: collapse(q.Text), Attribution: collapse(q.Attribution)})
	}
	return out
}

// checkID bounds and matches one cited id, reporting a fault.
func checkID(id, at string, rs *reasons) bool {
	if len(id) > maxRecordIDBytes {
		rs.add(ReasonMalformedID, at, "a %d-byte record id (max %d)", len(id), maxRecordIDBytes)
		return false
	}
	if !payloadRecordIDRe.MatchString(id) {
		rs.add(ReasonMalformedID, at, "%q is not a record id (want itd-N or iss-N)", termsafe.Sanitize(id))
		return false
	}
	return true
}

// checkProse bounds one untrusted text on its RAW length (the cleaner truncates
// silently, so measuring after it would let an oversize text through shortened),
// refuses page-breaking structure, and returns the cleaned line.
func checkProse(raw, at string, rs *reasons) (string, bool) {
	if len(raw) > maxEntryProseBytes {
		rs.add(ReasonTextOversize, at, "a %d-byte text (max %d)", len(raw), maxEntryProseBytes)
		return "", false
	}
	if !structureSafe(raw, at, rs) {
		return "", false
	}
	return termsafe.CleanProseLine(raw, maxEntryProseBytes), true
}

// structureSafe refuses a text that would break the page: one whose first
// non-space rune is `#` (a heading) or that carries a code fence.
func structureSafe(raw, at string, rs *reasons) bool {
	ok := true
	if strings.HasPrefix(strings.TrimSpace(raw), "#") {
		rs.add(ReasonHeading, at, "the text opens with `#`, which would render as a heading; the page carries one heading, the binary's")
		ok = false
	}
	if fenceRe.MatchString(raw) {
		rs.add(ReasonFence, at, "the text carries a code fence, which would swallow the rest of the page")
		ok = false
	}
	return ok
}

// outsideCause says why a cited id is not in the page's set, so the composer is
// told which rule it broke.
func outsideCause(cut Cut, id string) string {
	for _, e := range cut.Added {
		if e.ID != id {
			continue
		}
		switch {
		case strings.HasPrefix(id, "iss-"):
			return "it is an issue, and an issue carries no press release (its line is in the changelog)"
		default:
			return "it declares `impact: internal`, and an internal intent is not announced"
		}
	}
	for _, e := range cut.Removed {
		if e.ID == id {
			return "it was removed from shipped/ in this cut, and a removed intent is not announced"
		}
	}
	return "it is not in this cut (only intents that shipped since the last release are cited; nothing planned)"
}

// collapse joins whitespace runs into single spaces and trims.
func collapse(s string) string { return strings.Join(strings.Fields(s), " ") }

// sourceParagraphs splits a press-release section into paragraphs, each
// normalised the way the record's summary is: blockquote markers stripped and
// whitespace runs collapsed.
func sourceParagraphs(section string) []string {
	var out, para []string
	flush := func() {
		if len(para) > 0 {
			out = append(out, collapse(strings.Join(para, " ")))
			para = nil
		}
	}
	for _, raw := range strings.Split(section, "\n") {
		line := strings.TrimSpace(strings.TrimRight(raw, "\r"))
		for strings.HasPrefix(line, ">") {
			line = strings.TrimSpace(strings.TrimPrefix(line, ">"))
		}
		if line == "" {
			flush()
			continue
		}
		para = append(para, line)
	}
	flush()
	return out
}

// verbatim returns "" when the quote is carried word for word from source, or
// why it is not.
func verbatim(source string, q Quote) string {
	text := collapse(q.Text)
	attribution := collapse(q.Attribution)
	switch {
	case text == "":
		return "the quote is empty"
	case attribution == "":
		return "the quote carries no attribution"
	case !strings.Contains(text, attribution):
		return "the attribution does not appear in the quote as the source has it"
	case termsafe.CleanProseLine(q.Text, maxEntryProseBytes) != text:
		return "the quote carries text the page cannot render as written"
	}
	for _, para := range sourceParagraphs(source) {
		if strings.Contains(para, text) {
			return ""
		}
	}
	return "no paragraph of the intent's `## Press Release` section contains it"
}

// renderPage renders the page deterministically from a validated payload.
func renderPage(heading string, p validatedPage) string {
	lines := []string{heading, ""}
	for _, h := range p.headlines {
		lines = append(lines, fmt.Sprintf("%s (%s)", h.Text, strings.Join(h.Records, ", ")), "")
		told := map[string]bool{}
		for _, id := range h.Records {
			told[id] = true
		}
		for _, q := range p.quotes {
			if told[q.Record] {
				lines = append(lines, fmt.Sprintf("> %s (%s)", q.Text, q.Record), "")
			}
		}
	}
	if len(p.listed) > 0 {
		lines = append(lines, "Also in this release:", "")
		for _, e := range p.listed {
			lines = append(lines, fmt.Sprintf("- %s (%s)", termsafe.CleanProseLine(e.Title, maxEntryProseBytes), e.ID))
		}
		lines = append(lines, "")
	}
	lines = append(lines, "The line-by-line record of this release is its section in "+changelogFile+".")
	return strings.Join(lines, "\n") + "\n"
}
