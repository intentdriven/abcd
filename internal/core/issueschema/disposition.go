package issueschema

// The ONE reader of "which disposition is in force".
//
// The question is asked by two packages — core/capture, which refuses a second
// answer that does not supersede the standing one, and core/lint, which reports
// what nobody has answered — and it cannot be asked by one, because core/lint
// may not import core/capture (capture's own tests import lint, so the edge back
// would be a cycle). Two readers were tolerable; two ANSWERS were not, and twice
// in review they diverged: first on a duplicated key, then on a file led by a
// comment or a blank line, where the lenient scanner read straight past the
// preamble to the supersession edge while the strict parser refused the file and
// left both records standing. The board then said "answered" and the verb said
// "two standing", over the same bytes.
//
// So the DECISION lives here, in the leaf both packages already read the ledger's
// schema from, and each package supplies only the directory walk. This file
// touches no filesystem — the walk is what differs between a writer holding a
// lock and a gate scanning a tree, and the judgement is what must not.

import (
	"sort"
	"strings"
)

// DispositionRecord is one disposition as the standing computation sees it: the
// fields that decide what is in force, plus whether the record is well-formed
// enough to be trusted with them.
type DispositionRecord struct {
	// ID is the record's id, taken from its filename.
	ID string
	// State is the disposition's state, or "" when the record does not carry one.
	State string
	// ExitCondition is what would end a held disposition.
	ExitCondition string
	// Supersedes is the disposition this one replaces, or "" when it replaces
	// none. It is EMPTY on a record that is not well-formed, whatever the bytes
	// say: a record no reader can read cannot be trusted to retire another.
	Supersedes string
	// WellFormed reports whether the frontmatter block is one every reader of
	// this ledger can read: opened on the first line, closed, and carrying no
	// duplicated top-level key. A record that is not well-formed still STANDS —
	// dropping it would let a malformed file silently remove the answer it claims
	// to replace — so it is the caller who is told there is something here to
	// deal with.
	WellFormed bool
}

// DispositionFileID returns the id a ledger filename claims and whether it
// claims one at all. A file naming nothing (a README, a stray note) is not an
// answer and contributes neither presence nor a supersession edge.
func DispositionFileID(name string) (string, bool) {
	id, ok := strings.CutSuffix(name, ".md")
	if !ok || !dispositionIDRe.MatchString(id) {
		return "", false
	}
	return id, true
}

// ParseDisposition reads one disposition's frontmatter. It is deliberately its
// own small scanner rather than either package's: the strict ledger parser
// refuses more than this question needs to know, and the lenient line scanner
// reads past shapes that make a record untrustworthy. What both sides need is
// exactly this — the three deciding fields, and an honest verdict on whether the
// block they came from can be read at all.
func ParseDisposition(id, content string) DispositionRecord {
	rec := DispositionRecord{ID: id}
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")

	// The block must OPEN on the first line. A comment, a blank line, or any
	// other preamble means the file is not the shape a record is written in, and
	// tolerating it is precisely where the two readers parted company.
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return rec
	}
	closeAt := -1
	for i := 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], " ") || strings.HasPrefix(lines[i], "\t") {
			continue
		}
		if strings.TrimSpace(lines[i]) == "---" {
			closeAt = i
			break
		}
	}
	if closeAt == -1 {
		return rec
	}

	seen := map[string]bool{}
	values := map[string]string{}
	for _, line := range lines[1:closeAt] {
		// Only top-level keys decide anything here; an indented line belongs to a
		// nested value this question does not read.
		if line == "" || strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		// A duplicated top-level key is malformed to every reader of this ledger:
		// the strict parser refuses the file outright while a lenient scanner keeps
		// only the first value, so a second line can hide the value the first
		// shows. Such a record decides nothing.
		if seen[key] {
			return DispositionRecord{ID: id}
		}
		seen[key] = true
		values[key] = strings.Trim(strings.TrimSpace(value), `"'`)
	}

	// A record citing ITSELF is not well-formed. Honouring the citation would
	// supersede the record out of the standing set, so an item plainly carrying an
	// answer would read as carrying none — the verb would accept a fresh uncited
	// answer and the board would report it outstanding. Nothing legitimate spells
	// it: a supersession names the answer being replaced, and no record replaces
	// itself.
	if s := values["supersedes_disposition"]; s == id {
		return DispositionRecord{ID: id}
	} else if dispositionIDRe.MatchString(s) {
		rec.Supersedes = s
	}
	rec.WellFormed = true
	rec.State = values["state"]
	rec.ExitCondition = values["exit_condition"]
	return rec
}

// StandingDispositionIDs returns the ids of the records no sibling supersedes,
// sorted — the answers currently in force, which is exactly one in a healthy
// ledger. None means the item is unanswered, which the outstanding report says
// out loud and no state names; more than one is a ledger fault the write path
// refuses, and no reader may pick a winner from it.
//
// The superseded set is built only from WELL-FORMED records, which is the whole
// convergence: a record neither reader can read retires nothing, and it does not
// vanish either.
func StandingDispositionIDs(records []DispositionRecord) []string {
	superseded := make(map[string]bool, len(records))
	for _, r := range records {
		if r.WellFormed && r.Supersedes != "" {
			superseded[r.Supersedes] = true
		}
	}
	standing := make([]string, 0, len(records))
	for _, r := range records {
		if !superseded[r.ID] {
			standing = append(standing, r.ID)
		}
	}
	sort.Strings(standing)
	return standing
}
