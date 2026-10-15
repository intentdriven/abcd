package lint

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
)

// Generated index regions (iss-2608220150157511). A register that enumerated a
// directory by hand made every branch that added a record co-edit the same
// lines, so two lanes landing a record each conflicted on the register even
// though neither touched the other's record. In generated mode the region is
// never edited by hand: RegenerateIndexes writes it from the directory, the
// index_drift rule compares it against the same rendering byte for byte, and a
// conflicted region is resolved by rerunning the generator over the merged tree.
// The rendering is a pure function of the directory, sorted by a stable key, so
// the order two branches landed in cannot change it. This is the surface
// appendix precedent (adr-2609231028044006) applied to the record's registers.

// The generated mode and its renderers.
const (
	indexModeGenerated = "generated"

	indexRenderNames    = "names"     // `a`, `b`, `c`. — wrapped
	indexRenderTitles   = "titles"    // - `itd-8` — <H1 title>
	indexRenderADRTable = "adr-table" // | [adr-N](file) | Title | Status | Date |
)

// IndexRegenerateCommand is the one command that rewrites every generated region;
// every finding and refusal names it.
const IndexRegenerateCommand = "go generate ./internal/surface/cli"

// indexNamesWidth is the column the names renderer wraps its list at.
const indexNamesWidth = 80

// IndexRewrite is one document holding generated regions: its bytes as
// committed and as the generator derives them. Equal means nothing to write.
type IndexRewrite struct {
	Doc       string // repo-relative, as the config names it
	Committed string
	Want      string
}

// RegenerateIndexes derives every generated region the config's index_drift
// rule declares and returns, per document, the committed bytes and the bytes
// with each region rewritten. It writes nothing: the generator writes, and the
// drift test compares, from the same result. A document whose region is
// missing or unclosed, or a directory a renderer cannot derive, is an error
// naming the document, because writing around it would silently drop the
// region. A disabled or absent rule yields nothing.
func RegenerateIndexes(repoRoot string, cfg Config) ([]IndexRewrite, error) {
	rc, ok := cfg.Rules["index_drift"]
	if !ok || !rc.Enabled {
		return nil, nil
	}
	var order []string
	docs := map[string]*IndexRewrite{}
	for i, spec := range rc.Indexes {
		if spec.Mode != indexModeGenerated {
			continue
		}
		if err := validateGeneratedSpec(spec, i); err != nil {
			return nil, err
		}
		rw, seen := docs[spec.Doc]
		if !seen {
			data, err := readRepoFile(repoRoot, spec.Doc, maxRepoFileBytes)
			if err != nil {
				return nil, fmt.Errorf("index %s: reading %s: %w", spec.ID, spec.Doc, err)
			}
			text := strings.ReplaceAll(string(data), "\r\n", "\n")
			rw = &IndexRewrite{Doc: spec.Doc, Committed: string(data), Want: text}
			docs[spec.Doc] = rw
			order = append(order, spec.Doc)
		}
		body, err := renderIndex(repoRoot, spec)
		if err != nil {
			return nil, fmt.Errorf("index %s in %s: %w", spec.ID, spec.Doc, err)
		}
		lines := strings.Split(rw.Want, "\n")
		open, end, err := generatedRegion(lines, spec.ID)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", spec.Doc, err)
		}
		out := append([]string{}, lines[:open+1]...)
		out = append(out, body...)
		out = append(out, lines[end:]...)
		rw.Want = strings.Join(out, "\n")
	}
	out := make([]IndexRewrite, 0, len(order))
	for _, d := range order {
		out = append(out, *docs[d])
	}
	return out, nil
}

// validateGeneratedSpec is the configuration check for a generated index: it
// needs a document, a directory and a known renderer, and any dir_entry must
// compile. The entry pattern is not read — nothing in the region is parsed.
func validateGeneratedSpec(spec IndexSpec, i int) error {
	who := spec.ID
	if who == "" {
		who = "index " + fmt.Sprint(i)
	}
	if spec.ID == "" || spec.Doc == "" || spec.Dir == "" || spec.Render == "" {
		return &configError{"index_drift entry " + who + " in " + indexModeGenerated + " mode needs id, doc, dir, and render"}
	}
	switch spec.Render {
	case indexRenderNames, indexRenderTitles, indexRenderADRTable:
	default:
		return &configError{"index_drift entry " + who + " has unknown render " + spec.Render +
			" (want " + indexRenderNames + "|" + indexRenderTitles + "|" + indexRenderADRTable + ")"}
	}
	if spec.DirEntry != "" {
		if _, err := regexp.Compile(spec.DirEntry); err != nil {
			return &configError{"index_drift entry " + who + " has an uncompilable dir_entry pattern: " + err.Error()}
		}
	}
	return nil
}

// generatedRegion's refusals.
var (
	errNoRegion        = errors.New("no region")
	errUnclosedRegion  = errors.New("unclosed region")
	errDuplicateRegion = errors.New("duplicate region")
)

// generatedRegion locates a generated region: the index of the line holding
// only the opening marker for id, and of the line holding only the closing
// marker after it. A generated region's markers stand on lines of their own, so
// the generator can replace everything between them without touching a byte of
// the prose around them. A second opening marker for the same id is refused,
// naming both lines: taking the first and ignoring the rest would leave a
// listing in the document that no gate reads and no regeneration rewrites. A
// marker inside a fenced code block is quoted material — a page showing what a
// region looks like — and is never read as one.
func generatedRegion(lines []string, id string) (open, end int, err error) {
	var opens []int
	end = -1
	fenced := fenceMask(lines)
	for i, line := range lines {
		if fenced[i] {
			continue
		}
		t := strings.TrimSpace(line)
		if m := indexOpenRe.FindStringSubmatch(t); m != nil && m[0] == t && m[1] == id {
			opens = append(opens, i)
			continue
		}
		if len(opens) == 1 && end < 0 {
			if loc := indexCloseRe.FindString(t); loc != "" && loc == t {
				end = i
			}
		}
	}
	switch {
	case len(opens) == 0:
		return -1, -1, fmt.Errorf("%w: no line holding only <!-- index: %s --> (add the two marker lines, then run %s)", errNoRegion, id, IndexRegenerateCommand)
	case len(opens) > 1:
		return -1, -1, fmt.Errorf("%w: <!-- index: %s --> stands on lines %d and %d; keep one region per id (delete the other, markers included), then run %s",
			errDuplicateRegion, id, opens[0]+1, opens[1]+1, IndexRegenerateCommand)
	case end < 0:
		return -1, -1, fmt.Errorf("%w: <!-- index: %s --> on line %d has no <!-- /index --> line after it; "+
			"add a line holding only <!-- /index --> where the region ends, then run %s", errUnclosedRegion, id, opens[0]+1, IndexRegenerateCommand)
	}
	return opens[0], end, nil
}

// checkGeneratedIndex is index_drift's generated mode: the region must equal
// the rendering. One finding per index, on the opening marker's line, naming
// the first line that differs and the command that rewrites the region.
func checkGeneratedIndex(repoRoot string, spec IndexSpec, i int, cfg RuleConfig) ([]Finding, error) {
	if err := validateGeneratedSpec(spec, i); err != nil {
		return nil, err
	}
	data, err := readRepoFile(repoRoot, spec.Doc, maxRepoFileBytes)
	if err != nil {
		return nil, &configError{"index_drift entry " + spec.ID + ": reading " + spec.Doc + ": " + err.Error()}
	}
	finding := func(line int, msg string) []Finding {
		return []Finding{{File: spec.Doc, Line: line, RuleID: "index_drift", Severity: cfg.Severity, Message: msg}}
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	open, end, err := generatedRegion(lines, spec.ID)
	if err != nil {
		msg := "generated index '" + spec.ID + "': " + err.Error()
		if errors.Is(err, errNoRegion) {
			// Only a region that is gone altogether may mean the enumeration went.
			msg += "; delete the config entry when the enumeration goes"
		}
		return finding(0, msg), nil
	}
	want, err := renderIndex(repoRoot, spec)
	if err != nil {
		return finding(open+1, "generated index '"+spec.ID+"' cannot be derived from "+spec.Dir+": "+err.Error()), nil
	}
	got := lines[open+1 : end]
	for k := 0; k < len(got) || k < len(want); k++ {
		if k < len(got) && k < len(want) && got[k] == want[k] {
			continue
		}
		return finding(open+1, fmt.Sprintf("generated index '%s' differs from what %s derives (first at line %d); "+
			"never edit the region by hand, and a merge conflict in it is resolved the same way: run `%s`",
			spec.ID, spec.Dir, open+2+k, IndexRegenerateCommand)), nil
	}
	return nil, nil
}

// renderIndex composes a generated region's lines from the directory. A source
// directory that does not exist is a refusal naming it, never an empty
// listing: a moved or misnamed directory would otherwise empty the region and
// the region would still equal its rendering, so the gate would stay green over
// an index that lists nothing.
func renderIndex(repoRoot string, spec IndexSpec) ([]string, error) {
	if st, err := os.Stat(filepath.Join(repoRoot, filepath.FromSlash(spec.Dir))); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%s does not exist; point the index's dir at where the records now live, "+
				"or delete the config entry when the enumeration goes", spec.Dir)
		}
		return nil, err
	} else if !st.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", spec.Dir)
	}
	var dirEntryRe *regexp.Regexp
	if spec.DirEntry != "" {
		dirEntryRe = regexp.MustCompile(spec.DirEntry) // compiled once in validateGeneratedSpec
	}
	switch spec.Render {
	case indexRenderNames:
		set, err := indexDirEntries(repoRoot, spec, dirEntryRe)
		if err != nil {
			return nil, err
		}
		names := make([]string, 0, len(set))
		for n := range set {
			names = append(names, n)
		}
		sort.Strings(names)
		return renderNameList(names), nil
	case indexRenderTitles:
		recs, err := indexRecords(repoRoot, spec, dirEntryRe)
		if err != nil {
			return nil, err
		}
		out := make([]string, 0, len(recs))
		for _, r := range recs {
			out = append(out, "- `"+r.id+"` — "+r.title)
		}
		return out, nil
	case indexRenderADRTable:
		recs, err := indexRecords(repoRoot, spec, dirEntryRe)
		if err != nil {
			return nil, err
		}
		out := []string{"| ID | Title | Status | Date |", "|---|---|---|---|"}
		for _, r := range recs {
			title := adrTitlePrefixRe.ReplaceAllString(r.title, "")
			// A supersession the H1 already states is not repeated.
			var notes []string
			for _, n := range []struct{ key, verb string }{{"supersedes", "supersedes "}, {"superseded_by", "superseded by "}} {
				if s := fmList(r.fields, n.key); len(s) > 0 {
					if note := n.verb + strings.Join(s, ", "); !strings.Contains(title, note) {
						notes = append(notes, note)
					}
				}
			}
			if len(notes) > 0 {
				title += " (" + strings.Join(notes, "; ") + ")"
			}
			out = append(out, "| ["+r.id+"]("+r.file+") | "+tableCell(title)+" | "+
				tableCell(fmScalar(r.fields, "status"))+" | "+tableCell(fmScalar(r.fields, "date"))+" |")
		}
		return out, nil
	}
	return nil, fmt.Errorf("unknown render %q", spec.Render)
}

// adrTitlePrefixRe is the `ADR-<N>: ` an ADR's H1 opens with; the table's ID
// column already carries it.
var adrTitlePrefixRe = regexp.MustCompile(`^ADR-[0-9]+:\s*`)

// indexRecord is one record a titles or table renderer lists.
type indexRecord struct {
	id     string
	file   string
	title  string
	fields map[string]fmField
}

// indexRecords reads every record in an index's directory — the files carrying
// its suffix, the README excepted — with its id, H1 title and frontmatter,
// sorted by id with the digits compared as numbers, so `itd-9` precedes
// `itd-10` and an ordinal ADR precedes a minted one. The id is the frontmatter
// `id`, else the dir_entry reduction of the file's stem, else the stem. A record
// with no title is an error naming it: a blank row is not a listing.
func indexRecords(repoRoot string, spec IndexSpec, dirEntryRe *regexp.Regexp) ([]indexRecord, error) {
	dir := filepath.Join(repoRoot, filepath.FromSlash(spec.Dir))
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var recs []indexRecord
	for _, e := range ents {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, spec.Suffix) {
			continue
		}
		stem := strings.TrimSuffix(name, spec.Suffix)
		if strings.EqualFold(stem, "README") {
			continue
		}
		id := indexDirName(stem, dirEntryRe)
		if id == "" {
			continue
		}
		data, err := readRepoAbs(repoRoot, filepath.Join(dir, name), maxRepoFileBytes)
		if err != nil {
			return nil, err
		}
		lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
		fields := frontmatterFields(lines)
		if v := fmScalar(fields, "id"); v != "" {
			id = v
		}
		title := recordTitle(lines)
		if title == "" {
			return nil, fmt.Errorf("%s/%s has no title to list", spec.Dir, name)
		}
		recs = append(recs, indexRecord{id: id, file: name, title: title, fields: fields})
	}
	sort.Slice(recs, func(i, j int) bool { return naturalLess(recs[i].id, recs[j].id) })
	return recs, nil
}

// fmScalar reads a frontmatter scalar, comment stripped and unquoted; "" when
// absent or null.
func fmScalar(fields map[string]fmField, key string) string {
	f, ok := fields[key]
	if !ok {
		return ""
	}
	raw := strings.TrimSpace(frontmatter.StripComment(f.value))
	if isNull(raw) {
		return ""
	}
	if v, ok := frontmatter.ScalarString(raw); ok {
		return v
	}
	return raw
}

// fmList reads a frontmatter field that is a flow sequence or a single scalar.
func fmList(fields map[string]fmField, key string) []string {
	f, ok := fields[key]
	if !ok {
		return nil
	}
	raw := strings.TrimSpace(frontmatter.StripComment(f.value))
	if isNull(raw) {
		return nil
	}
	return frontmatter.StringList(raw)
}

// tableCell keeps a value from opening a new table column.
func tableCell(s string) string { return strings.ReplaceAll(s, "|", `\|`) }

// naturalLess orders ids with each run of digits compared as a number, so the
// order is total and stable however the directory lists its files.
func naturalLess(a, b string) bool {
	for a != "" && b != "" {
		da, db := digitRun(a), digitRun(b)
		if da > 0 && db > 0 {
			na, nb := strings.TrimLeft(a[:da], "0"), strings.TrimLeft(b[:db], "0")
			if len(na) != len(nb) {
				return len(na) < len(nb)
			}
			if na != nb {
				return na < nb
			}
			a, b = a[da:], b[db:]
			continue
		}
		if a[0] != b[0] {
			return a[0] < b[0]
		}
		a, b = a[1:], b[1:]
	}
	return len(a) < len(b)
}

func digitRun(s string) int {
	n := 0
	for n < len(s) && s[n] >= '0' && s[n] <= '9' {
		n++
	}
	return n
}

// renderNameList renders names as one sentence of backticked entries, commas
// between and a full stop at the end, filled greedily to indexNamesWidth.
func renderNameList(names []string) []string {
	var out []string
	line := ""
	for i, n := range names {
		tok := "`" + n + "`"
		if i < len(names)-1 {
			tok += ","
		} else {
			tok += "."
		}
		switch {
		case line == "":
			line = tok
		case len(line)+1+len(tok) > indexNamesWidth:
			out = append(out, line)
			line = tok
		default:
			line += " " + tok
		}
	}
	if line != "" {
		out = append(out, line)
	}
	return out
}
