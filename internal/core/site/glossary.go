package site

// The glossary page set, and the term links that reach it.
//
// The glossary under `.abcd/development/brief/glossary/` names every sense of a
// word the record uses in more than one (iss-2609012245352480). Until this, the
// site read none of it: a rendered page showed "phase", "spec" or "surface" as
// plain words, and a reader who did not already know which sense was meant had
// nowhere on the site to find out (iss-2609020922150364).
//
// Two halves, and both are needed. The page set publishes the entries — an
// index and one page per term — so an entry HAS a URL; the term links put that
// URL where the word is, so a reader meets it at the moment of doubt rather
// than having to go looking. A page set nobody links is a glossary in a drawer.
//
// # Where the pages live, and why under /record/
//
// A term file is durable record prose, rendered verbatim exactly as an ADR's
// body is, so the entries sit under `/record/glossary/` with the rest of the
// verbatim rendering — which is where adr-47 decision 3 already places the
// single-source exemption these pages need, and which the `/record/*` header
// block already covers. Nothing here is generic-side special pleading: a
// repository with no glossary directory gets no pages, no navigation entry and
// no term links, and the build succeeds (itd-140).
//
// # What the linker will not touch
//
// The first occurrence of each ENTRY on a record page becomes a link, and only
// the first: a page where every "phase" is blue is a page nobody reads, and a
// term's aliases are spellings of the same entry rather than terms of their
// own, so "user" and "plugin consumer" together earn one link, not two. Four
// places are left exactly as the record wrote them — inside a code span, inside
// a heading, inside a link that is already there, and on a term's own entry
// page, where the link would lead the reader to the page they are on. The scan
// runs over the generator's OWN rendered HTML, where text is already escaped and
// tags are already balanced, so it inserts an anchor around text rather than
// re-parsing markdown, and a match is never made across an entity or inside one.

import (
	"os"
	"path"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/core/glossary"
)

// routeGlossary is the glossary index; an entry's route hangs beneath it.
const routeGlossary = "record/glossary/"

// glossaryEntry is one term as the site publishes it: what the term file says,
// the context it belongs to, the file it came from and the page it becomes.
type glossaryEntry struct {
	Term    glossary.Term
	Context string
	// Path is the term file, repo-relative — the span the page's prose is from.
	Path string
	// Route is the entry's page, e.g. `record/glossary/core/phase/`.
	Route string
}

// loadGlossaryEntries reads the glossary through the repository root.
//
// An ABSENT glossary is not an error: it is a repository that keeps none, and
// the pages, the navigation entry and the links all go with it. An UNREADABLE
// term is an error, because glossary.Scan already refuses to let a term with no
// name, status or definition vanish silently from an index, and a page set that
// quietly dropped it would undo that.
func loadGlossaryEntries(root *os.Root) ([]glossaryEntry, error) {
	g, err := glossary.ScanInRoot(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []glossaryEntry
	for _, ctx := range g.Contexts {
		for _, t := range ctx.Terms {
			out = append(out, glossaryEntry{
				Term:    t,
				Context: ctx.Name,
				Path:    path.Join(glossary.DirRelPath, ctx.Name, t.File),
				Route:   routeGlossary + ctx.Name + "/" + strings.TrimSuffix(t.File, ".md") + "/",
			})
		}
	}
	return out, nil
}

// hasGlossary reports whether the repository declares any term. Without one the
// page set and its navigation entry are omitted.
func (e *explorer) hasGlossary() bool { return len(e.glossary) > 0 }

// glossaryIndexPage renders `/record/glossary/` — one panel per bounded
// context, each a list of that context's terms with the definition the term file
// states, in the shape the development page's lists take.
func (e *explorer) glossaryIndexPage() (string, error) {
	byContext := map[string][]glossaryEntry{}
	var contexts []string
	for _, en := range e.glossary {
		if _, seen := byContext[en.Context]; !seen {
			contexts = append(contexts, en.Context)
		}
		byContext[en.Context] = append(byContext[en.Context], en)
	}
	sort.Strings(contexts)

	var b strings.Builder
	b.WriteString(`<div class="dash">`)
	for _, ctx := range contexts {
		entries := byContext[ctx]
		var l strings.Builder
		l.WriteString(`<ul class="list reclist">`)
		for _, en := range entries {
			l.WriteString(`<li><span class="id"><a href="/` + escapeAttr(en.Route) + `">` +
				escapeText(en.Term.Name) + `</a>`)
			if en.Term.Status != "" {
				l.WriteString(`<span class="d">` + escapeText(en.Term.Status) + `</span>`)
			}
			l.WriteString(`</span><span` + srcAttr(en.Path, "") + `>` +
				escapeText(en.Term.Definition) + `</span></li>`)
		}
		l.WriteString(`</ul>`)
		// The context's name is a directory name, which the generator may print,
		// and it is the anchor its own terms sit under.
		b.WriteString(`<div id="` + escapeAttr(ctx) + `" class="c12">`)
		b.WriteString(panel("c12", ctx, itoaLen(len(entries)), l.String()))
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div>`)
	return e.shell(routeGlossary, e.c.ui.RecordNav.Glossary, "", b.String()), nil
}

// glossaryTermPage renders one entry — the term file's own body, around it the
// fields the file declares and the two forge links a record page also carries.
func (e *explorer) glossaryTermPage(en glossaryEntry) (string, error) {
	body, err := e.renderMarkdownBody(en.Path)
	if err != nil {
		return "", err
	}
	// A term never links its own word to its own page: the reader is there.
	body = e.linkTerms(body, en.Route)

	var b strings.Builder
	b.WriteString(`<div class="pills recpills">`)
	b.WriteString(`<span class="pill type" style="--c:var(--s-neutral)"><i></i>` +
		escapeText(en.Context) + `</span>`)
	if en.Term.Status != "" {
		b.WriteString(`<span class="pill ` + statusTone(en.Term.Status) + `">` +
			escapeText(en.Term.Status) + `</span>`)
	}
	b.WriteString(`<span class="mono id">` + escapeText(en.Term.Name) + `</span>`)
	b.WriteString(`</div>`)

	b.WriteString(`<div class="dash recgrid">`)
	b.WriteString(`<div class="panel c8 recbody"` + srcAttr(en.Path, "") + `>` + body + `</div>`)
	b.WriteString(`<div class="c4 recside">`)
	b.WriteString(panel("", e.c.ui.Record.Frontmatter, "", e.glossaryFieldTable(en)))
	b.WriteString(`</div>`)
	b.WriteString(`</div>`)

	return e.shell(en.Route, en.Term.Name, "", b.String()), nil
}

// glossaryFieldTable is the term file's own fields, named as the file names
// them — the same rule the record's frontmatter table follows.
func (e *explorer) glossaryFieldTable(en glossaryEntry) string {
	rows := [][2]string{
		{"term", en.Term.Name},
		{"bounded_context", en.Context},
		{"status", en.Term.Status},
		{"aliases", strings.Join(en.Term.Aliases, ", ")},
		{"definition", en.Term.Definition},
	}
	var b strings.Builder
	b.WriteString(`<div class="tablewrap"><table class="fm"><tbody>`)
	for _, r := range rows {
		if r[1] == "" {
			continue
		}
		b.WriteString(`<tr><th scope="row">` + escapeText(r[0]) + `</th><td` +
			srcAttr(en.Path, "") + `>` + escapeText(r[1]) + `</td></tr>`)
	}
	b.WriteString(`</tbody></table></div>`)
	b.WriteString(e.forgeFileLinks(en.Path, true))
	return b.String()
}

// --- the term linker --------------------------------------------------------

// gtermClass marks a link the generator added around a word the record wrote.
// It is a class rather than nothing so the stylesheet can draw it as the quiet
// cross-reference it is, and so a reader of the emitted page can tell a link
// the record wrote from one the glossary put there.
const gtermClass = "gterm"

// termLinker holds every spelling that reaches an entry, longest first.
type termLinker struct {
	// byName maps an ASCII-lowered term or alias to its entry's route.
	byName map[string]string
	// lengths are the distinct spelling lengths in bytes, longest first, so
	// "roadmap phase" is matched before the "phase" inside it.
	lengths []int
}

// newTermLinker indexes the entries. A spelling two entries claim goes to the
// first in the scan's own order, which is the directory's order: silently
// preferring one is better than linking a word to two places, and the glossary's
// own GL002 rule is what stops two terms claiming one word in the first place.
func newTermLinker(entries []glossaryEntry) *termLinker {
	l := &termLinker{byName: map[string]string{}}
	seen := map[int]bool{}
	add := func(name, route string) {
		name = strings.TrimSpace(name)
		if name == "" || !linkableSpelling(name) {
			return
		}
		key := asciiLower(name)
		if _, dup := l.byName[key]; dup {
			return
		}
		l.byName[key] = route
		if !seen[len(key)] {
			seen[len(key)] = true
			l.lengths = append(l.lengths, len(key))
		}
	}
	for _, en := range entries {
		add(en.Term.Name, en.Route)
		for _, a := range en.Term.Aliases {
			add(a, en.Route)
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(l.lengths)))
	return l
}

// linkableSpelling reports whether a spelling can be matched in rendered text at
// all. It must open and close on a word rune, so a match can be bounded on both
// sides, and it must carry no character the renderer would have escaped — a
// spelling containing `&`, `<` or `>` never appears verbatim in the output, and
// looking for one would either never match or match across an entity.
func linkableSpelling(s string) bool {
	if strings.ContainsAny(s, "&<>") {
		return false
	}
	first, _ := utf8.DecodeRuneInString(s)
	last, _ := utf8.DecodeLastRuneInString(s)
	return isWordRune(first) && isWordRune(last)
}

// isWordRune is what a term may not be matched across. The hyphen and the
// underscore count as word characters on purpose: "phase" inside
// "plumbing-phase" and inside `record_lint` is a different word, and linking it
// would point a reader at the entry for something else.
func isWordRune(r rune) bool {
	return r == '-' || r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

// asciiLower lowercases the ASCII letters and nothing else, so the folded form
// is the same length as what was folded and a match can be sliced back out of
// the text it was found in.
func asciiLower(s string) string {
	var b []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			if b == nil {
				b = []byte(s)
			}
			b[i] = c + ('a' - 'A')
		}
	}
	if b == nil {
		return s
	}
	return string(b)
}

// linkTerms wraps the first occurrence of each glossary term in the rendered
// HTML in a link to its entry. skipRoute is the page's own route, so an entry
// page never links its own word to itself; it is empty everywhere else.
func (e *explorer) linkTerms(html, skipRoute string) string {
	return e.terms.link(html, skipRoute)
}

// suppressedElement names the elements a term link must not be put inside: a
// link (nesting one anchor in another is not a document), a code span or a
// preformatted block (the record is quoting an identifier, not using a word),
// a heading (a linked heading reads as a section that goes somewhere else), and
// the two raw-text elements whose content is not prose at all.
func suppressedElement(name string) bool {
	switch name {
	case "a", "code", "pre", "script", "style",
		"h1", "h2", "h3", "h4", "h5", "h6":
		return true
	}
	return false
}

// link is the scan. It walks the generator's own output a tag and a text run at
// a time — never re-parsing markdown, never touching anything inside a tag —
// and rewrites only text that is not inside a suppressed element.
func (l *termLinker) link(html, skipRoute string) string {
	if l == nil || len(l.byName) == 0 {
		return html
	}
	// linked is keyed by ENTRY, not by spelling: a term and its aliases are
	// spellings of one thing, and a page that linked each of them would carry
	// three links to one entry.
	linked := map[string]bool{}
	var b strings.Builder
	b.Grow(len(html))
	suppress := 0
	for i := 0; i < len(html); {
		if html[i] == '<' {
			j := strings.IndexByte(html[i:], '>')
			if j < 0 {
				// Not markup this generator emits. Nothing after it can be read
				// with any confidence, so it is copied out untouched.
				b.WriteString(html[i:])
				break
			}
			tag := html[i : i+j+1]
			b.WriteString(tag)
			i += j + 1
			name, closing, self := tagParts(tag)
			if self || !suppressedElement(name) {
				continue
			}
			if closing {
				if suppress > 0 {
					suppress--
				}
				continue
			}
			suppress++
			continue
		}
		end := len(html)
		if j := strings.IndexByte(html[i:], '<'); j >= 0 {
			end = i + j
		}
		if suppress > 0 {
			b.WriteString(html[i:end])
		} else {
			l.linkRun(&b, html[i:end], linked, skipRoute)
		}
		i = end
	}
	return b.String()
}

// tagParts reads a tag's name and whether it closes or closes itself.
func tagParts(tag string) (name string, closing, self bool) {
	s := strings.TrimSuffix(strings.TrimPrefix(tag, "<"), ">")
	self = strings.HasSuffix(s, "/")
	s = strings.TrimSuffix(s, "/")
	if strings.HasPrefix(s, "/") {
		closing = true
		s = s[1:]
	}
	if strings.HasPrefix(s, "!") {
		return "", false, true
	}
	end := len(s)
	for i := 0; i < len(s); i++ {
		if c := s[i]; c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			end = i
			break
		}
	}
	return asciiLower(s[:end]), closing, self
}

// linkRun rewrites one run of text, which the renderer has already escaped.
func (l *termLinker) linkRun(b *strings.Builder, run string, linked map[string]bool, skipRoute string) {
	prevWord := false
	for i := 0; i < len(run); {
		r, size := utf8.DecodeRuneInString(run[i:])
		// An entity is one indivisible character as far as a reader is
		// concerned, so it is copied whole: a term is never matched across it,
		// and the letters spelling `&amp;` are never matched inside it.
		if r == '&' {
			if k := strings.IndexByte(run[i:], ';'); k > 1 && k <= 12 {
				b.WriteString(run[i : i+k+1])
				i += k + 1
				prevWord = false
				continue
			}
		}
		word := isWordRune(r)
		if !word || prevWord {
			b.WriteString(run[i : i+size])
			prevWord = word
			i += size
			continue
		}
		if n := l.matchAt(run, i, linked, skipRoute); n > 0 {
			match := run[i : i+n]
			route := l.byName[asciiLower(match)]
			b.WriteString(`<a class="` + gtermClass + `" href="/` +
				escapeAttr(route) + `">` + match + `</a>`)
			linked[route] = true
			i += n
			prevWord = true
			continue
		}
		b.WriteString(run[i : i+size])
		prevWord = true
		i += size
	}
}

// matchAt returns the byte length of the longest spelling that starts at i and
// whose entry is still available to link, or 0. The word must END on a boundary
// too: "spec" inside "specification" is not the term, it is the start of a
// longer word. `linked` is keyed by route, so a spelling whose entry is already
// linked is passed over and a SHORTER spelling of a different entry starting at
// the same byte is still found.
func (l *termLinker) matchAt(run string, i int, linked map[string]bool, skipRoute string) int {
	for _, n := range l.lengths {
		if i+n > len(run) {
			continue
		}
		if i+n < len(run) {
			if next, _ := utf8.DecodeRuneInString(run[i+n:]); isWordRune(next) {
				continue
			}
		}
		key := asciiLower(run[i : i+n])
		route, ok := l.byName[key]
		if !ok || linked[route] || route == skipRoute {
			continue
		}
		return n
	}
	return 0
}
