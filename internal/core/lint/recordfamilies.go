package lint

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// The record-families pair (itd-2609211913453478, acceptance criterion 4) holds
// the glossary and the record stores to the one page that maps abcd's record
// families, `glossary/core/record-families.md`. The page's table is the closed
// set of families: its first column, one bold name per row. Neither rule keeps a
// word list of its own — the page says what a family is, and the glossary says
// which other words are grouping words.
//
//   - glossary_family_pointer REFUSES a glossary entry whose
//     not_to_be_confused_with names nothing on the page. Naming a family row, or
//     the page itself, anywhere in the value is how an entry points at the map;
//     a retired word, a term with no row, null and an absent key point nowhere.
//   - record_family_key REPORTS a record frontmatter key naming a family the
//     page does not define. It never refuses: its severity is pinned to warn in
//     code, whatever the config asks, because the criterion says "reported".
const (
	ruleGlossaryFamilyPointer = "glossary_family_pointer"
	ruleRecordFamilyKey       = "record_family_key"

	defaultGlossaryDir = ".abcd/development/brief/glossary"
	familiesPageName   = "record-families.md"
	familyIntentID     = "itd-2609211913453478"
)

// familyMap is the record-families page as the two rules read it.
type familyMap struct {
	rel      string          // repo-relative, slash-separated
	term     string          // the page's own term (record-families)
	families []string        // the table's first column, in page order
	isFamily map[string]bool // the same, as a set
}

// glossaryEntry is one term file inside a bounded-context directory.
type glossaryEntry struct {
	rel    string
	fields map[string]fmField
}

func (e glossaryEntry) value(key string) string { return strings.TrimSpace(e.fields[key].value) }

// familyRuleDirs resolves a rule's glossary directory and page path, both
// repo-relative, from its config (the registry is the page).
func familyRuleDirs(cfg RuleConfig) (glossaryDir, pageRel string) {
	glossaryDir = cfg.GlossaryDir
	if glossaryDir == "" {
		glossaryDir = defaultGlossaryDir
	}
	pageRel = cfg.Registry
	if pageRel == "" {
		pageRel = path.Join(filepath.ToSlash(glossaryDir), "core", familiesPageName)
	}
	return glossaryDir, pageRel
}

// loadFamiliesPage reads the page and parses its family table. A missing page,
// or one whose table yields no family, is a configuration error rather than a
// clean tree: an armed rule reading an empty family set would refuse every entry
// or, worse, a rule that skipped would go green over a map that is not there.
func loadFamiliesPage(repoRoot, pageRel string) (familyMap, error) {
	lines, err := readContained(repoRoot, pageRel, "record-families page")
	if err != nil {
		return familyMap{}, err
	}
	if lines == nil {
		return familyMap{}, &configError{"record-families page " + quote(pageRel) +
			" does not exist; the families rules read the page's table as the closed set of families, and an absent page would disarm them"}
	}
	p := familyMap{rel: pageRel, isFamily: map[string]bool{}}
	p.term = strings.TrimSuffix(path.Base(pageRel), ".md")
	if t := strings.TrimSpace(frontmatterFields(lines)["term"].value); t != "" {
		p.term = strings.ToLower(t)
	}
	inTable := false
	for _, l := range lines {
		var cells []string
		if strings.HasPrefix(strings.TrimSpace(l), "|") {
			cells = tableCells(l)
		}
		switch {
		case len(cells) == 0:
			inTable = false
		case strings.EqualFold(cells[0], "Family"):
			inTable = true
		case inTable && strings.HasPrefix(cells[0], "**") && strings.HasSuffix(cells[0], "**"):
			name := strings.ToLower(strings.TrimSpace(strings.Trim(cells[0], "*")))
			if name != "" && !p.isFamily[name] {
				p.families = append(p.families, name)
				p.isFamily[name] = true
			}
		}
	}
	if len(p.families) == 0 {
		return familyMap{}, &configError{"record-families page " + quote(pageRel) +
			" has no family table (a `| Family | ...` table whose rows open with a bold name); the families rules have nothing to hold the glossary to"}
	}
	return p, nil
}

// readContained reads one repo-relative, cloned-repo-controlled path the way
// every lint walk does — contained, symlink-resolved inside the repository, and
// size-guarded. An absent file yields nil lines and no error.
func readContained(repoRoot, rel, what string) ([]string, error) {
	if err := containedRepoPath(rel); err != nil {
		return nil, &configError{what + " " + quote(rel) + " " + err.Error() + "; the lint reads only inside the repository"}
	}
	abs := filepath.Join(repoRoot, filepath.FromSlash(rel))
	if _, err := os.Lstat(abs); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	real, err := containedRealPath(repoRoot, abs)
	if err != nil {
		return nil, &configError{what + " " + quote(rel) + " " + err.Error() + "; the lint reads only inside the repository"}
	}
	content, err := fsutil.ReadGuarded(real, citationPageSizeLimit)
	if err != nil {
		return nil, err
	}
	return strings.Split(string(content), "\n"), nil
}

// loadGlossaryEntries reads every term file that sits in a bounded-context
// directory (one level below the glossary root). The root's own files — the
// README and the term template — are not entries, and neither is a context
// README.
func loadGlossaryEntries(repoRoot, glossaryDir string) ([]glossaryEntry, error) {
	if err := containedRepoPath(glossaryDir); err != nil {
		return nil, &configError{"glossary_dir " + quote(glossaryDir) + " " + err.Error() + "; the lint reads only inside the repository"}
	}
	dirAbs := filepath.Join(repoRoot, glossaryDir)
	if err := resolvedInsideRoot(repoRoot, dirAbs); err != nil {
		return nil, &configError{"glossary_dir " + quote(glossaryDir) + " " + err.Error() + "; the lint reads only inside the repository"}
	}
	files, err := markdownFiles(dirAbs)
	if err != nil {
		return nil, err
	}
	var out []glossaryEntry
	for _, fileAbs := range files {
		inner, err := filepath.Rel(dirAbs, fileAbs)
		if err != nil {
			return nil, err
		}
		if len(strings.Split(filepath.ToSlash(inner), "/")) != 2 || strings.EqualFold(filepath.Base(inner), "README.md") {
			continue
		}
		rel := filepath.ToSlash(repoRel(repoRoot, fileAbs))
		lines, err := readContained(repoRoot, rel, "file")
		if err != nil {
			return nil, err
		}
		fields := frontmatterFields(lines)
		if strings.TrimSpace(fields["term"].value) == "" {
			continue
		}
		out = append(out, glossaryEntry{rel: rel, fields: fields})
	}
	return out, nil
}

// namedTerm is the term a not_to_be_confused_with member names: the segment
// after the bounded context (`core/spec` names spec), lower-cased.
func namedTerm(member string) string {
	m := strings.ToLower(strings.TrimSpace(member))
	if i := strings.LastIndex(m, "/"); i >= 0 {
		m = m[i+1:]
	}
	return strings.TrimSuffix(m, ".md")
}

// checkGlossaryFamilyPointer implements glossary_family_pointer.
func checkGlossaryFamilyPointer(repoRoot string, cfg RuleConfig) ([]Finding, error) {
	glossaryDir, pageRel := familyRuleDirs(cfg)
	page, err := loadFamiliesPage(repoRoot, pageRel)
	if err != nil {
		return nil, err
	}
	entries, err := loadGlossaryEntries(repoRoot, glossaryDir)
	if err != nil {
		return nil, err
	}
	onPage := func(member string) bool {
		t := namedTerm(member)
		return page.isFamily[t] || t == page.term
	}
	families := strings.Join(page.families, ", ")
	var out []Finding
	for _, e := range entries {
		term := strings.ToLower(e.value("term"))
		if e.rel == page.rel || term == page.term {
			continue
		}
		f, declared := e.fields["not_to_be_confused_with"]
		line := f.line
		named := false
		if declared && !isNull(strings.TrimSpace(f.value)) {
			for _, m := range parseYAMLStringList(f.value) {
				if onPage(m) {
					named = true
					break
				}
			}
		}
		if named {
			continue
		}
		what := "names nothing on the record-families page"
		if !declared {
			what = "is absent, so the entry names nothing on the record-families page"
			line = e.fields["term"].line
		}
		out = append(out, Finding{
			File: filepath.FromSlash(e.rel), Line: line, RuleID: ruleGlossaryFamilyPointer, Severity: cfg.Severity,
			Message: "glossary entry '" + term + "': not_to_be_confused_with " + what + " (" + page.rel +
				"); name the family it is confused with (" + families + "), or the page itself as core/" + page.term +
				", beside any other term it names — every entry points at the one map (" + familyIntentID + ")",
		})
	}
	return out, nil
}

// familyWord is one grouping word the glossary knows that the page has no row
// for, with the reason it counts as a family word.
type familyWord struct {
	toks []string
	why  string
}

// nonFamilyWords derives, from the glossary alone, the grouping words the page
// does not define: every superseded entry's term and aliases (the retired
// families), and every forbidden synonym a family entry declares (a second
// name for a family). A word that is itself a family on the page, or an alias
// of one, is never in the set.
func nonFamilyWords(entries []glossaryEntry, page familyMap) []familyWord {
	onPage := map[string]bool{}
	for _, f := range page.families {
		onPage[f] = true
	}
	for _, e := range entries {
		if page.isFamily[strings.ToLower(e.value("term"))] {
			for _, a := range parseYAMLStringList(e.value("aliases")) {
				onPage[normWord(a)] = true
			}
		}
	}
	seen := map[string]bool{}
	var out []familyWord
	add := func(w, why string) {
		n := normWord(w)
		if n == "" || onPage[n] || seen[n] {
			return
		}
		seen[n] = true
		out = append(out, familyWord{toks: strings.Split(n, "_"), why: why})
	}
	// Retired families first, so a word that is both (phase is superseded and a
	// forbidden synonym of bundle) is reported as what it most plainly is.
	for _, e := range entries {
		if term := strings.ToLower(e.value("term")); strings.EqualFold(e.value("status"), "superseded") {
			add(term, "the superseded glossary term '"+term+"'")
			for _, a := range parseYAMLStringList(e.value("aliases")) {
				add(a, "an alias of the superseded glossary term '"+term+"'")
			}
		}
	}
	for _, e := range entries {
		if term := strings.ToLower(e.value("term")); page.isFamily[term] {
			for _, s := range parseYAMLStringList(e.value("forbidden_synonyms")) {
				add(s, "a forbidden synonym of the family '"+term+"'")
			}
		}
	}
	return out
}

// normWord folds a word or phrase to underscore-joined lower-case tokens, the
// spelling a frontmatter key uses.
func normWord(w string) string {
	f := strings.FieldsFunc(strings.ToLower(w), func(r rune) bool { return r == ' ' || r == '-' || r == '_' })
	return strings.Join(f, "_")
}

// keyNames reports whether a frontmatter key's tokens hold the word's tokens
// contiguously, the last token also matching in its plural spelling.
func keyNames(keyToks, wordToks []string) bool {
	n := len(wordToks)
	for i := 0; i+n <= len(keyToks); i++ {
		ok := true
		for j := 0; j < n && ok; j++ {
			k, w := keyToks[i+j], wordToks[j]
			ok = k == w || (j == n-1 && (k == w+"s" || k == w+"es"))
		}
		if ok {
			return true
		}
	}
	return false
}

// checkRecordFamilyKey implements record_family_key.
func checkRecordFamilyKey(repoRoot string, cfg RuleConfig) ([]Finding, error) {
	glossaryDir, pageRel := familyRuleDirs(cfg)
	page, err := loadFamiliesPage(repoRoot, pageRel)
	if err != nil {
		return nil, err
	}
	entries, err := loadGlossaryEntries(repoRoot, glossaryDir)
	if err != nil {
		return nil, err
	}
	words := nonFamilyWords(entries, page)
	if len(words) == 0 {
		return nil, nil
	}
	// The store walk's own findings (a misplaced file, a malformed filename)
	// are record_schema's to report; this rule reads only the keys.
	records, _, err := scanRecordStores(repoRoot, RuleConfig{RecordStores: cfg.RecordStores, Severity: severityWarn})
	if err != nil {
		return nil, err
	}
	families := strings.Join(page.families, ", ")
	var out []Finding
	for _, r := range records {
		keys := make([]string, 0, len(r.fields))
		for k := range r.fields {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			keyToks := strings.Split(normWord(k), "_")
			for _, w := range words {
				if !keyNames(keyToks, w.toks) {
					continue
				}
				out = append(out, Finding{
					File: r.rel, Line: r.fields[k].line, RuleID: ruleRecordFamilyKey, Severity: severityWarn,
					Message: "frontmatter key '" + k + "' names '" + strings.Join(w.toks, " ") + "', " + w.why +
						", which the record-families page (" + page.rel + ") does not define; its families are " +
						families + " — reported, not refused (" + familyIntentID + ")",
				})
				break
			}
		}
	}
	return out, nil
}
