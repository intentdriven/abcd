package lint

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// The index_drift region markers. A hand-written enumeration of a directory's
// contents is fenced by an opening marker naming the index id and a closing
// marker; everything between them is the enumeration the rule checks. Keying on
// an explicit marker rather than on list shape is what keeps the rule off every
// other list in the corpus: an unmarked paragraph is never parsed, so unrelated
// markdown cannot produce a finding.
var (
	indexOpenRe  = regexp.MustCompile(`<!--\s*index:\s*([A-Za-z0-9._-]+)\s*-->`)
	indexCloseRe = regexp.MustCompile(`<!--\s*/index\s*(?::\s*[A-Za-z0-9._-]+\s*)?-->`)
	// A backticked span is the candidate entry; the configured entry pattern
	// decides which candidates are enumeration members.
	indexTokenRe = regexp.MustCompile("`([^`\n]+)`")
)

// The two index_drift modes.
const (
	indexModeExact  = "exact"  // the listing and the directory must agree exactly
	indexModeAbsent = "absent" // every listed path must NOT exist
)

// checkIndexDrift is the derive-don't-store gate on hand-written directory
// indexes (adr-5, iss-38). A README that enumerates its sibling files by hand
// drifts silently the moment a file is added, moved, or shipped; the rule makes
// that drift fail the record gate instead.
//
// Each configured index names a marked region in a document and the directory
// the region describes, plus the regexp a backticked token must match to count
// as an entry — and, optionally, the mirror pattern reducing a directory entry's
// name to what the document enumerates, so a listing may name records by id
// rather than by filename. Two modes:
//
//   - exact: the enumerated set and the directory's contents must be equal —
//     a file added to the directory and a name listed for a file that is gone
//     are both findings.
//   - absent: every enumerated path must NOT exist. This is the shape a
//     "planned seams" list has — the drift is a seam that shipped while the
//     list still calls it planned.
//
// The rule fails closed: a configured index whose region is missing, or whose
// region yields no entries, is a finding rather than a silent pass, so deleting
// the enumeration means deleting its config entry too. A malformed index spec
// (no doc, no dir, no entry pattern, an uncompilable pattern, an unknown mode)
// is a configuration error, never a quiet no-op.
func checkIndexDrift(repoRoot string, cfg RuleConfig) ([]Finding, error) {
	var out []Finding
	for i, spec := range cfg.Indexes {
		fs, err := checkOneIndex(repoRoot, spec, i, cfg)
		if err != nil {
			return nil, err
		}
		out = append(out, fs...)
	}
	return out, nil
}

func checkOneIndex(repoRoot string, spec IndexSpec, i int, cfg RuleConfig) ([]Finding, error) {
	who := spec.ID
	if who == "" {
		who = "index " + fmt.Sprint(i)
	}
	if spec.Doc == "" || spec.Dir == "" || spec.Entry == "" {
		return nil, &configError{"index_drift entry " + who + " needs doc, dir, and entry"}
	}
	mode := spec.Mode
	if mode == "" {
		mode = indexModeExact
	}
	if mode != indexModeExact && mode != indexModeAbsent {
		return nil, &configError{"index_drift entry " + who + " has unknown mode " + mode + " (want " + indexModeExact + "|" + indexModeAbsent + ")"}
	}
	entryRe, err := regexp.Compile(spec.Entry)
	if err != nil {
		return nil, &configError{"index_drift entry " + who + " has an uncompilable entry pattern: " + err.Error()}
	}
	var dirEntryRe *regexp.Regexp
	if spec.DirEntry != "" {
		if mode == indexModeAbsent {
			return nil, &configError{"index_drift entry " + who + " sets dir_entry in " + indexModeAbsent + " mode, which resolves listed paths rather than reading the directory"}
		}
		dirEntryRe, err = regexp.Compile(spec.DirEntry)
		if err != nil {
			return nil, &configError{"index_drift entry " + who + " has an uncompilable dir_entry pattern: " + err.Error()}
		}
	}

	data, err := readRepoFile(repoRoot, spec.Doc, maxRepoFileBytes)
	if err != nil {
		return nil, &configError{"index_drift entry " + who + ": reading " + spec.Doc + ": " + err.Error()}
	}
	listed, start, found := indexRegion(strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n"), spec.ID, entryRe)

	finding := func(line int, msg string) Finding {
		return Finding{File: spec.Doc, Line: line, RuleID: "index_drift", Severity: cfg.Severity, Message: msg}
	}
	if !found {
		return []Finding{finding(0, "index '"+spec.ID+"' is configured but "+spec.Doc+" has no <!-- index: "+spec.ID+" --> region; delete the config entry when the enumeration goes")}, nil
	}
	if len(listed) == 0 {
		return []Finding{finding(start, "index '"+spec.ID+"' region lists no entries matching "+spec.Entry+"; an enumeration that parses to nothing gates nothing")}, nil
	}

	if mode == indexModeAbsent {
		return absentFindings(repoRoot, spec, listed, finding), nil
	}
	return exactFindings(repoRoot, spec, dirEntryRe, listed, start, finding)
}

// listedEntry is one enumerated token and the line it sits on.
type listedEntry struct {
	name string
	line int
}

// indexRegion returns the entries enumerated in the document's region for id,
// the region's opening line, and whether that region exists. A token counts as
// an entry when it matches the configured pattern; a trailing slash (the
// directory spelling) is trimmed so `oracle/` and `oracle` are one entry.
func indexRegion(lines []string, id string, entryRe *regexp.Regexp) ([]listedEntry, int, bool) {
	var entries []listedEntry
	start, open := 0, false
	for i, line := range lines {
		// scan is the slice of the line that is inside the region: from just
		// after an opening marker found on this line (a marker shares its
		// line with entries when the region opens mid-paragraph), up to just
		// before a closing marker found on this line (the mirror case). A
		// token outside that slice is outside the region and must not count.
		scan := line
		if !open {
			loc := indexOpenRe.FindStringSubmatchIndex(line)
			if loc == nil || line[loc[2]:loc[3]] != id {
				continue
			}
			open, start = true, i+1
			scan = line[loc[1]:]
		}
		closed := false
		if loc := indexCloseRe.FindStringIndex(scan); loc != nil {
			scan, closed = scan[:loc[0]], true
		}
		for _, m := range indexTokenRe.FindAllStringSubmatch(scan, -1) {
			tok := strings.TrimSpace(m[1])
			if !entryRe.MatchString(tok) {
				continue
			}
			entries = append(entries, listedEntry{name: strings.TrimSuffix(tok, "/"), line: i + 1})
		}
		if closed {
			return entries, start, true
		}
	}
	// An unclosed region is treated as running to the end of the document: the
	// entries it did enumerate are still checked, so a lost closing marker
	// cannot disarm the gate.
	return entries, start, open
}

// exactFindings compares the enumeration against the directory both ways.
func exactFindings(repoRoot string, spec IndexSpec, dirEntryRe *regexp.Regexp, listed []listedEntry, start int, finding func(int, string) Finding) ([]Finding, error) {
	actual, err := indexDirEntries(repoRoot, spec, dirEntryRe)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool, len(listed))
	var out []Finding
	for _, e := range listed {
		seen[e.name] = true
		if !actual[e.name] {
			out = append(out, finding(e.line, "index '"+spec.ID+"' lists `"+e.name+"` but "+spec.Dir+" has no such entry"))
		}
	}
	missing := make([]string, 0, len(actual))
	for name := range actual {
		if !seen[name] {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	for _, name := range missing {
		out = append(out, finding(start, "index '"+spec.ID+"' omits `"+name+"`, which exists in "+spec.Dir))
	}
	return out, nil
}

// absentFindings flags every enumerated path that exists on disk — the shape a
// planned-seams list drifts into when a seam ships.
func absentFindings(repoRoot string, spec IndexSpec, listed []listedEntry, finding func(int, string) Finding) []Finding {
	var out []Finding
	for _, e := range listed {
		rel := path.Join(spec.Dir, e.name)
		if _, err := os.Stat(filepath.Join(repoRoot, filepath.FromSlash(rel))); err != nil {
			continue
		}
		out = append(out, finding(e.line, "index '"+spec.ID+"' lists `"+e.name+"` as absent but "+rel+" exists"))
	}
	return out
}

// indexDirEntries enumerates the directory an exact index describes: the stems
// of files carrying the configured suffix, or the immediate subdirectories when
// no suffix is configured. A README is never an entry — it is the index itself.
// A missing directory contributes nothing, so every listed name fires rather
// than the rule passing on an empty comparison. A configured dir_entry reduces
// each stem to the part the document enumerates, and drops any stem the pattern
// does not describe.
func indexDirEntries(repoRoot string, spec IndexSpec, dirEntryRe *regexp.Regexp) (map[string]bool, error) {
	set := map[string]bool{}
	ents, err := os.ReadDir(filepath.Join(repoRoot, filepath.FromSlash(spec.Dir)))
	if err != nil {
		if os.IsNotExist(err) {
			return set, nil
		}
		return nil, err
	}
	for _, e := range ents {
		if spec.Suffix == "" {
			if name := indexDirName(e.Name(), dirEntryRe); e.IsDir() && name != "" {
				set[name] = true
			}
			continue
		}
		if e.IsDir() || !strings.HasSuffix(e.Name(), spec.Suffix) {
			continue
		}
		name := strings.TrimSuffix(e.Name(), spec.Suffix)
		if strings.EqualFold(name, "README") {
			continue
		}
		if name = indexDirName(name, dirEntryRe); name != "" {
			set[name] = true
		}
	}
	return set, nil
}

// indexDirName reduces a directory entry's name to the entry the document
// enumerates: submatch 1 of the dir_entry pattern (the whole match when the
// pattern has no group), or "" when the pattern does not match, which drops the
// file from the comparison. A nil pattern returns the name unchanged.
func indexDirName(name string, re *regexp.Regexp) string {
	if re == nil {
		return name
	}
	m := re.FindStringSubmatch(name)
	if m == nil {
		return ""
	}
	if len(m) > 1 {
		return m[1]
	}
	return m[0]
}
