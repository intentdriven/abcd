// Package decisions owns the decision records' folder: one flat folder,
// `.abcd/work/decisions/`, holding one `dcn-<id>-<slug>.md` file per decision
// (adr-2610110925465593, itd-2609151138388536). A decision is a record like any
// other: its id comes from the adr-45 mint (internal/core/recordid), its slug is
// cut at recordid.MaxSlugLen, and recording one writes a new file and appends to
// nothing, so two branches that each record a decision merge with nothing to
// reconcile.
//
// This file is the READ side: the folder's path, the parse of one record, and
// the listing of every record in id order. The writer arrives with the verb
// (spc-2610110931381316, step 2). The package is transport-agnostic: it never
// writes to stdout and knows no front door.
//
// A record is:
//
//	---
//	id: dcn-<yymmddHHMMSS><rrrr>
//	slug: <at most 40 characters>
//	date: <YYYY-MM-DD>
//	---
//	<YYYY-MM-DD> — <the decision, in the words it was recorded in>
//
// The date is the id's stamp, never a second clock reading, and the body opens
// with it, as every entry of the retired single log did.
package decisions

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"syscall"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// RelDir is the decisions folder, repo-relative and slash-separated. It is
// recordid's constant, re-exported so a caller that reaches the folder through
// this package names it here.
const RelDir = recordid.DecisionsRelDir

// Family is the decision records' id prefix. It is `dcn`, not `dec`: the
// lifeboat keys a line of a frozen decisions log as `dec-L<line>`, and the
// distinct prefix keeps the two apart (spc-2610110931381316, decision 16).
const Family = "dcn"

// Required is the frontmatter every decision record carries. record-lint's dcn
// store declares the same list, and both refuse a record without one of them.
var Required = []string{"id", "slug", "date"}

// readLimit caps one record's read. A decision is a paragraph; anything near
// this is not one.
const readLimit = 1 << 20

// idRe is a decision id's grammar: the family, a hyphen, and the digits.
var idRe = regexp.MustCompile(`^` + Family + `-([0-9]+)$`)

// bodyDateRe reads the date a record's body opens with.
var bodyDateRe = regexp.MustCompile(`^([0-9]{4}-[0-9]{2}-[0-9]{2})`)

// Decision is one decision record as read from its file.
type Decision struct {
	// ID is the record's handle, `dcn-<digits>`.
	ID string `json:"id"`
	// Slug is the kebab-case slug its filename and frontmatter both carry.
	Slug string `json:"slug"`
	// Date is the ISO date the id's stamp carries.
	Date string `json:"date"`
	// Path is the record's file, repo-relative and slash-separated.
	Path string `json:"path"`
	// Text is the record's body as written, opening with its date, with the
	// blank lines around it trimmed.
	Text string `json:"text"`
}

// Skipped is a file in the folder that List could not read as a record, and
// why. The reader validates before it reads, so a malformed record is skipped,
// and naming it here is what keeps the skip from being silent.
type Skipped struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// Parse reads one decision record. rel is the file's repo-relative path; its
// base name must be the record's filename, `dcn-<id>-<slug>.md`, because the
// filename and the frontmatter state the id and the slug twice and must agree.
//
// It refuses, naming the fault, a record that has no frontmatter, states a key
// twice, omits or blanks a required property, carries an id that is not the dcn
// grammar or a date its id's stamp does not carry, disagrees with its filename,
// or has a body that does not open with its date.
func Parse(rel string, content []byte) (Decision, error) {
	rel = filepath.ToSlash(rel)
	text := strings.ReplaceAll(string(content), "\r\n", "\n")
	lines := strings.Split(text, "\n")
	if frontmatter.Close(lines) < 0 {
		return Decision{}, fmt.Errorf("decisions: %s has no frontmatter block", rel)
	}
	if dups := frontmatter.Duplicates(lines); len(dups) > 0 {
		return Decision{}, fmt.Errorf("decisions: %s states '%s' twice (line %d)", rel, dups[0].Key, dups[0].Line)
	}
	fields := frontmatter.Fields(lines)
	values := map[string]string{}
	for _, key := range Required {
		v, ok := frontmatter.ScalarString(fields[key].Value)
		if !ok || strings.TrimSpace(v) == "" {
			return Decision{}, fmt.Errorf("decisions: %s is missing required property '%s'", rel, key)
		}
		values[key] = v
	}
	d := Decision{ID: values["id"], Slug: values["slug"], Date: values["date"], Path: rel}

	m := idRe.FindStringSubmatch(d.ID)
	if m == nil {
		return Decision{}, fmt.Errorf("decisions: %s: id %q is not %s-<digits>", rel, d.ID, Family)
	}
	stampDate := recordid.DateFromStamp(m[1])
	if stampDate == "" {
		return Decision{}, fmt.Errorf("decisions: %s: id %s carries no mint stamp to date it by", rel, d.ID)
	}
	if d.Date != stampDate {
		return Decision{}, fmt.Errorf("decisions: %s: date %s disagrees with its id's stamp, which carries %s", rel, d.Date, stampDate)
	}

	fileID, fileSlug, ok := recordid.SplitRecordFilename(Family, path.Base(rel))
	if !ok {
		return Decision{}, fmt.Errorf("decisions: %s is not a decision filename (%s-<id>-<slug>.md)", rel, Family)
	}
	if fileID != d.ID {
		return Decision{}, fmt.Errorf("decisions: %s: filename claims %s but frontmatter declares %s", rel, fileID, d.ID)
	}
	if fileSlug != d.Slug {
		return Decision{}, fmt.Errorf("decisions: %s: filename carries slug %q but frontmatter declares %q", rel, fileSlug, d.Slug)
	}

	_, body := frontmatter.Split(text)
	d.Text = strings.Trim(body, "\n")
	bm := bodyDateRe.FindStringSubmatch(d.Text)
	if bm == nil {
		return Decision{}, fmt.Errorf("decisions: %s: the body must open with its date, %s", rel, d.Date)
	}
	if bm[1] != d.Date {
		return Decision{}, fmt.Errorf("decisions: %s: the body opens with %s but the record's date is %s", rel, bm[1], d.Date)
	}
	return d, nil
}

// List reads every decision record in repoRoot's folder and returns them in id
// order, oldest first: the id is the stamp, so id order is mint order. A folder
// that does not exist holds no decisions, and that is an empty list rather than
// an error, since a repository need not have recorded one.
//
// The folder's README, any file that is not markdown and any subdirectory are
// not records (record-lint reports a subdirectory of the flat store). Every
// other entry that cannot be read as one — a link, a file whose name is outside
// the grammar, a record Parse refuses — comes back in the skipped list with its
// reason. A folder that is itself a link, or
// that cannot be read, is an error: reporting no decisions when the truth is
// "the folder could not be read" would hide every one of them.
func List(repoRoot string) ([]Decision, []Skipped, error) {
	dirAbs := filepath.Join(repoRoot, filepath.FromSlash(RelDir))
	fi, err := os.Lstat(dirAbs)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("decisions: reading %s: %w", RelDir, err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return nil, nil, fmt.Errorf("decisions: %s is a link; the folder is read only where it is", RelDir)
	}
	if !fi.IsDir() {
		return nil, nil, fmt.Errorf("decisions: %s is not a directory", RelDir)
	}
	entries, err := os.ReadDir(dirAbs)
	if err != nil {
		return nil, nil, fmt.Errorf("decisions: reading %s: %w", RelDir, err)
	}
	var out []Decision
	var skipped []Skipped
	for _, e := range entries {
		name := e.Name()
		rel := RelDir + "/" + name
		if e.IsDir() || !strings.HasSuffix(name, ".md") || strings.EqualFold(name, "README.md") {
			continue
		}
		if e.Type()&os.ModeSymlink != 0 {
			skipped = append(skipped, Skipped{Path: rel, Reason: "a record is never a link, and the reader does not follow one"})
			continue
		}
		content, err := fsutil.ReadGuarded(filepath.Join(dirAbs, name), readLimit)
		if err != nil {
			skipped = append(skipped, Skipped{Path: rel, Reason: "cannot be read safely: " + err.Error()})
			continue
		}
		d, err := Parse(rel, content)
		if err != nil {
			skipped = append(skipped, Skipped{Path: rel, Reason: err.Error()})
			continue
		}
		out = append(out, d)
	}
	sort.SliceStable(out, func(i, j int) bool { return idLess(out[i].ID, out[j].ID) })
	return out, skipped, nil
}

// idLess orders two decision ids by their numbers. A minted id's number is
// sixteen digits, so the string order is the numeric one; the length compare
// keeps that true of a shorter number too.
func idLess(a, b string) bool {
	na := strings.TrimLeft(strings.TrimPrefix(a, Family+"-"), "0")
	nb := strings.TrimLeft(strings.TrimPrefix(b, Family+"-"), "0")
	if len(na) != len(nb) {
		return len(na) < len(nb)
	}
	return na < nb
}
