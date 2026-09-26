package site

// A small CSL-JSON formatter, stdlib only.
//
// It exists because adr-47 rules out both a Node toolchain and committed HTML,
// which is what every off-the-shelf renderer needs. What it produces is the
// author-date-title-container shape the record's own bibliography already
// carries, with DOIs and URLs linked — enough to publish the sources honestly,
// and no more.
//
// The load-bearing part is not the formatting, it is the NUMBERING. The record
// cites `[7]` in prose against `ACKNOWLEDGEMENTS.md`'s numbered list, so a page
// that numbered the same sources differently would silently point every citation
// at the wrong source. The build therefore checks the two agree, entry by entry,
// and fails loudly when they do not.

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// CSLRelPath is where the record keeps its bibliography.
const CSLRelPath = ".abcd/development/research/references.csl.json"

// AcknowledgementsRelPath is the file whose numbering the page must agree with.
const AcknowledgementsRelPath = "ACKNOWLEDGEMENTS.md"

// The two headings the references page takes its own titles from.
//
// These are SEARCH KEYS, never text to render: the page prints the heading as
// the acknowledgement file spells it, so the two cannot drift apart in what they
// call things, and a repository that spells them differently gets a refusal
// rather than a page titled with a literal from in here.
const (
	referencesHeading   = "References & sources"
	inspirationsHeading = "Inspirations"
)

const maxBibliographyBytes = 4 << 20

// numberedItemRe matches one entry of the acknowledgement file's numbered list.
//
// It is anchored at COLUMN ZERO on purpose. A wrapped entry is indented under
// its own number, and a continuation line that happens to begin with a year —
// "2025. Is vibe coding safe?" — is otherwise read as item 2025, which turns a
// correct file into a count that does not match anything.
var numberedItemRe = regexp.MustCompile(`^(\d+)\.\s+(.*)$`)

// CSLEntry is one bibliography record, in the CSL-JSON fields the record uses.
type CSLEntry struct {
	ID             string     `json:"id"`
	Type           string     `json:"type"`
	Author         []CSLName  `json:"author"`
	Title          string     `json:"title"`
	ContainerTitle string     `json:"container-title"`
	Volume         string     `json:"volume"`
	Issue          string     `json:"issue"`
	Page           string     `json:"page"`
	Publisher      string     `json:"publisher"`
	PublisherPlace string     `json:"publisher-place"`
	Issued         *CSLIssued `json:"issued"`
	DOI            string     `json:"DOI"`
	URL            string     `json:"URL"`
}

// CSLName is one author.
type CSLName struct {
	Family  string `json:"family"`
	Given   string `json:"given"`
	Literal string `json:"literal"`
}

// CSLIssued is a CSL date, of which only the year is used.
type CSLIssued struct {
	DateParts [][]json.Number `json:"date-parts"`
}

// Year is the entry's year, or "" where it carries none.
func (d *CSLIssued) Year() string {
	if d == nil || len(d.DateParts) == 0 || len(d.DateParts[0]) == 0 {
		return ""
	}
	return d.DateParts[0][0].String()
}

// Bibliography is the rendered bibliography plus the credited inspirations
// beside it.
type Bibliography struct {
	// Entries are the sources, in the file's own order — which is the order
	// their numbers are cited in.
	Entries []CSLEntry
	// Path is the CSL file the entries came from.
	Path string
	// Inspirations are the acknowledgement file's credited ideas, as markdown
	// list items, and Heading its own heading for them.
	Inspirations     []Block
	InspirationsLead []Block
	Heading          string
	RefsHeading      string
	// Source is the acknowledgement file, RefsAnchor and InspAnchor the headings
	// the numbered sources and the credited ideas sit under, and AckLinkDefs its
	// link reference definitions — so every block rendered from that file can
	// carry the span it came from.
	Source      string
	RefsAnchor  string
	InspAnchor  string
	AckLinkDefs map[string]string
}

// LoadBibliography reads the CSL file and the acknowledgement file, checks that
// they number the same sources the same way, and returns what the page renders.
//
// An ABSENT CSL file is a state, not a fault: the page and its navigation entry
// are omitted and the build succeeds. A PRESENT file whose numbering disagrees
// with the acknowledgement list is a fault, and it fails the build naming the
// entry — the citations in the record's prose are numbers, and a number pointing
// at the wrong source is worse than no page at all.
func LoadBibliography(repoRoot, cslRel, ackRel string) (*Bibliography, error) {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	data, err := fsutil.ReadGuardedInRoot(root, cslRel, maxBibliographyBytes)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var entries []CSLEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("site: %s: %w", cslRel, err)
	}
	if len(entries) == 0 {
		return nil, nil
	}
	// A bibliography is repository text like any other, and its addresses become
	// hrefs on a published page. Nothing else in the tree validates this file, so
	// the same scheme check the markdown path applies to every link runs here:
	// escaping an attribute is no defence against a well-formed `javascript:`.
	for i, e := range entries {
		for _, u := range []string{e.URL, e.DOI} {
			if scheme, bad := executableScheme(u); bad {
				return nil, fmt.Errorf("site: %s entry %d (%s) addresses %q, which runs code in the reader's browser; a bibliography links to sources",
					cslRel, i+1, e.ID, scheme)
			}
		}
	}
	b := &Bibliography{Entries: entries, Path: cslRel}

	// Every heading this page shows comes OUT OF the acknowledgement file. The
	// generator holds the two names only as search keys and never as text to
	// render (adr-47 decision 2): a page that fell back to its own literal would
	// print a heading the repository never wrote, under which it would then
	// publish that repository's sources.
	ack, err := fsutil.ReadGuardedInRoot(root, ackRel, maxBibliographyBytes)
	if err != nil {
		if os.IsNotExist(err) {
			// No file, no heading, no page. The bibliography is still exported;
			// it is the rendered page that has nothing to call itself.
			return nil, nil
		}
		return nil, err
	}
	text, consumed := StripFrontmatter(string(ack))
	secs, err := Sections(ackRel, text, consumed)
	if err != nil {
		return nil, err
	}
	b.Source = ackRel
	b.AckLinkDefs = LinkDefinitions(text)
	var numbered []string
	for _, s := range secs {
		switch {
		case strings.EqualFold(s.Title, referencesHeading):
			b.RefsHeading, b.RefsAnchor = s.Title, s.Anchor
			numbered = numberedItems(s.Body)
		case strings.EqualFold(s.Title, inspirationsHeading):
			b.Heading, b.InspAnchor = s.Title, s.Anchor
			for _, blk := range Blocks(s.Body, s.BodyLine) {
				if isUnorderedItem(strings.TrimLeft(blk.Text, " \t")) {
					b.Inspirations = append(b.Inspirations, blk)
					continue
				}
				b.InspirationsLead = append(b.InspirationsLead, blk)
			}
		}
	}
	// The file is HERE and it does not carry the heading the sources go under.
	// That is a fault, not an absence: the bibliography exists, the page has to
	// title it, and skipping quietly would leave the sources unpublished with
	// nothing said about why.
	if b.RefsHeading == "" {
		return nil, missingHeading(ackRel, referencesHeading, cslRel)
	}
	if len(numbered) == 0 {
		return nil, fmt.Errorf("site: %s § %s carries no numbered list, and %s lists %d sources — the record cites these by number, so the numbering has to exist to be agreed with",
			ackRel, b.RefsHeading, cslRel, len(entries))
	}
	if err := checkNumbering(cslRel, ackRel, entries, numbered); err != nil {
		return nil, err
	}
	return b, nil
}

// missingHeading is the refusal for an acknowledgement file that is present and
// does not carry a heading the page has to render.
func missingHeading(ackRel, heading, cslRel string) error {
	return fmt.Errorf("site: %s carries no '## %s' heading, and %s lists sources to publish under it — the site renders the file's own headings and never a literal of its own",
		ackRel, heading, cslRel)
}

// numberedItems collects the text of each entry of a numbered list, joined so a
// wrapped entry is one string.
func numberedItems(body string) []string {
	var out []string
	for _, line := range strings.Split(body, "\n") {
		if m := numberedItemRe.FindStringSubmatch(line); m != nil {
			out = append(out, m[2])
			continue
		}
		if len(out) > 0 && strings.TrimSpace(line) != "" {
			out[len(out)-1] += " " + strings.TrimSpace(line)
		}
	}
	return out
}

// checkNumbering asserts that entry N of the CSL file is entry N of the
// acknowledgement list.
//
// The comparison is on the two facts a citation is looked up by — the first
// author's family name and the year — rather than on the whole formatted string,
// because the two renderings legitimately differ in punctuation and this check
// is about ORDER, not about typography.
func checkNumbering(cslRel, ackRel string, entries []CSLEntry, numbered []string) error {
	if len(entries) != len(numbered) {
		return fmt.Errorf("site: %s lists %d sources and %s § %s numbers %d — the record cites these by number, so the two must agree entry for entry",
			cslRel, len(entries), ackRel, referencesHeading, len(numbered))
	}
	for i, e := range entries {
		key := e.firstFamily()
		year := e.Issued.Year()
		item := numbered[i]
		if key != "" && !strings.Contains(item, key) {
			return fmt.Errorf("site: %s entry %d is %q but %s § %s numbers %d as %q — a citation to [%d] would point at the wrong source",
				cslRel, i+1, e.ID, ackRel, referencesHeading, i+1, clip(item), i+1)
		}
		if year != "" && !strings.Contains(item, year) {
			return fmt.Errorf("site: %s entry %d is dated %s but %s § %s numbers %d as %q — a citation to [%d] would point at the wrong source",
				cslRel, i+1, year, ackRel, referencesHeading, i+1, clip(item), i+1)
		}
	}
	return nil
}

// firstFamily is the entry's first author's family name, which is how a reader
// looks a numbered citation up.
func (e CSLEntry) firstFamily() string {
	if len(e.Author) == 0 {
		return ""
	}
	if f := e.Author[0].Family; f != "" {
		return f
	}
	return e.Author[0].Literal
}

// Render formats one entry as HTML: authors, year, title, container, the
// publication details the type carries, and the DOI or URL as a link.
func (e CSLEntry) Render() string {
	var parts []string
	if a := e.authorList(); a != "" {
		parts = append(parts, a)
	}
	if y := e.Issued.Year(); y != "" {
		parts = append(parts, y)
	}
	title := escapeText(e.Title)
	// A work that stands alone is titled in italics; one inside a container is
	// titled plainly and its container is italicised. That is the whole of the
	// style, and it is the distinction a reader needs to find the thing.
	if e.ContainerTitle == "" && (e.Type == "book" || e.Type == "report") {
		title = "<em>" + title + "</em>"
	}
	parts = append(parts, title)
	if e.ContainerTitle != "" {
		c := "<em>" + escapeText(e.ContainerTitle) + "</em>"
		if v := e.volumeIssue(); v != "" {
			c += " " + escapeText(v)
		}
		parts = append(parts, c)
	}
	if e.Page != "" {
		parts = append(parts, escapeText(e.Page))
	}
	if e.Publisher != "" {
		p := escapeText(e.Publisher)
		if e.PublisherPlace != "" {
			p += ", " + escapeText(e.PublisherPlace)
		}
		parts = append(parts, p)
	}
	out := strings.Join(parts, ". ") + "."
	if link := e.link(); link != "" {
		out += " " + link
	}
	return out
}

// volumeIssue is the "19, 6" a journal entry carries.
func (e CSLEntry) volumeIssue() string {
	switch {
	case e.Volume != "" && e.Issue != "":
		return e.Volume + ", " + e.Issue
	case e.Volume != "":
		return e.Volume
	case e.Issue != "":
		return e.Issue
	}
	return ""
}

// authorList renders the authors in reading order, the last joined with "and".
func (e CSLEntry) authorList() string {
	names := make([]string, 0, len(e.Author))
	for _, a := range e.Author {
		switch {
		case a.Literal != "":
			names = append(names, escapeText(a.Literal))
		case a.Given != "":
			names = append(names, escapeText(a.Given+" "+a.Family))
		default:
			names = append(names, escapeText(a.Family))
		}
	}
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	case 2:
		return names[0] + " and " + names[1]
	}
	return strings.Join(names[:len(names)-1], ", ") + ", and " + names[len(names)-1]
}

// link is the entry's resolvable address: its DOI where it has one, its URL
// otherwise. A source nobody can reach is a source nobody can check.
func (e CSLEntry) link() string {
	if e.DOI != "" {
		href := "https://doi.org/" + e.DOI
		return `<a href="` + escapeAttr(href) + `">doi:` + escapeText(e.DOI) + `</a>`
	}
	if e.URL != "" {
		return `<a href="` + escapeAttr(e.URL) + `">` + escapeText(e.URL) + `</a>`
	}
	return ""
}

// referencesPage renders `/references/`.
func (e *explorer) referencesPage() (string, error) {
	b := e.bib
	var out strings.Builder
	// Two bibliographies of very different length read badly as parallel
	// columns, and evening their heights only trades the ragged pair for a
	// screen of blank paper. They are stacked full-width instead, each folded
	// behind its own heading — the shape the relationship chart's list already
	// uses — so the page opens as two closed headings a reader chooses between.
	out.WriteString(`<div class="dash reading">`)

	// Every block on this page names the repository span it was lifted from, the
	// same way a composed chapter does. The sources are the CSL file's own
	// entries, so that file is their span; the heading above them is the
	// acknowledgement file's, so that section is its.
	var refs strings.Builder
	refs.WriteString(`<ol class="refs"` + srcAttr(b.Path, "") + `>`)
	for _, entry := range b.Entries {
		refs.WriteString(`<li>` + entry.Render() + `</li>`)
	}
	refs.WriteString(`</ol>`)
	if e.c.repo.Repository != "" {
		refs.WriteString(`<p class="small muted"` + srcAttr(b.Path, "") + `><a href="` +
			escapeAttr(forgeBlob(e.c.repo.Repository, b.Path)) + `">` + escapeText(b.Path) + `</a></p>`)
	}
	out.WriteString(panelDisclosure("c12", b.RefsHeading, b.Source+"#"+b.RefsAnchor,
		strconv.Itoa(len(b.Entries)), refs.String()))

	// Rendered only when the acknowledgement file supplied BOTH the heading and
	// the entries. Without the heading there is nothing to call the panel that
	// the repository wrote.
	if len(b.Inspirations) > 0 && b.Heading != "" {
		r := &Renderer{UI: e.c.ui, Refs: b.AckLinkDefs,
			Image: func(src, alt string, at Source) (string, error) { return e.c.assets.render(".", src, alt, at) },
			Link:  func(href string, at Source) string { return e.href(b.Source, href) }}
		lead, err := r.RenderBlocks(b.Source, b.InspirationsLead)
		if err != nil {
			return "", err
		}
		items, err := r.RenderBlocks(b.Source, b.Inspirations)
		if err != nil {
			return "", err
		}
		body := `<div class="prose small"` + srcAttr(b.Source, b.InspAnchor) + `>` + lead + `</div>` +
			`<div class="insp"` + srcAttr(b.Source, b.InspAnchor) + `>` + items + `</div>`
		out.WriteString(panelDisclosure("c12", b.Heading, b.Source+"#"+b.InspAnchor,
			strconv.Itoa(countListItems(b.Inspirations)), body))
	}
	out.WriteString(`</div>`)
	return e.shell(routeReferences, e.c.ui.NavReferences, "", out.String()), nil
}

// countListItems counts the bullets across a run of list blocks.
func countListItems(blocks []Block) int {
	n := 0
	for _, b := range blocks {
		for _, ln := range strings.Split(b.Text, "\n") {
			if isUnorderedItem(strings.TrimLeft(ln, " \t")) && indentOf(ln) == 0 {
				n++
			}
		}
	}
	return n
}
