package recordid

// resolve.go is the READ side of this package: given a repository, which record
// ids actually exist?
//
// The mint answers "what is a fresh id"; a resolver answers "does the id this
// payload cites name a real record". Both questions are about the same id space
// and the same filename grammar, so they share a home — a second resolver living
// next to a consumer (an ingest seam, a lint rule) would drift from the minting
// side the first time a family moved.
//
// Only the four ID-BEARING families are resolvable. The brief and the principles
// carry no per-entry id, so a citation of one cannot be validated by id at all;
// a caller that needs to reference them does so in prose, never as a citation.
//
// The scan is deliberately SHALLOW (a family root plus one bucket level) and
// symlink-refusing: the record families are flat-with-buckets by construction, an
// unbounded walk would let a pathological tree dominate the cost of an ingest,
// and following a link would let a planted symlink make a record appear to exist.

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
)

// CitedIDRe is the canonical grammar of a cited record id: one of the four
// id-bearing family prefixes, a hyphen, and a bare decimal N. It is exported
// because every ingest boundary that accepts a citation must bound and shape the
// string BEFORE it is looked up or echoed, and they must all agree on the shape.
var CitedIDRe = regexp.MustCompile(`^(?:adr|itd|iss|spc)-[0-9]+$`)

// adrFileRe matches an ADR filename NNNN-slug.md and captures the number. ADRs
// are the one family whose file does not carry its own id spelling, so the id is
// derived from the numeric prefix.
var adrFileRe = regexp.MustCompile(`^([0-9]+)-[^/]*\.md$`)

// adrHandleRe matches an ADR id in every spelling the record writes it in. It is
// case-insensitive and tolerant of padding because `adr-0012`, `adr-12` and
// `ADR-12` are one handle, not three records — the reading the record-lint gate,
// the citation resolver and the `abcd <id>` dispatch all already take.
var adrHandleRe = regexp.MustCompile(`(?i)^adr-([0-9]+)$`)

// CanonADRID canonicalises an ADR id to the one spelling every claimant of that
// decision resolves to: lower-case `adr-` and the number with its leading zeros
// trimmed. "" when the string is not an ADR id at all.
//
// ADRs are the one family carrying TWO id vintages: the hand-numbered ordinals
// 0001–0058, which keep their ids and filenames, and the minted
// `adr-<yymmddHHMMSS><rrrr>` form the 2026-09-01 ruling adopted. Both are the
// same `[0-9]+` grammar, so one canonicaliser serves both — which is the point of
// keeping it here rather than once per reader.
func CanonADRID(id string) string { return canonADRNum(adrHandleRe, id) }

// ADRFileID derives an ADR's canonical id from its filename: `0037-<slug>.md` →
// `adr-37`, `2609012206053814-<slug>.md` → `adr-2609012206053814`. "" when the
// name is not an ADR record's.
//
// It is exported because two sides must judge exactly the same files: the read-side
// resolver below, and the mint's presence check (core/decide), which redraws when a
// candidate id already names a record. A second copy of this derivation would let
// the mint re-issue an id the resolver already answers for.
func ADRFileID(name string) string { return canonADRNum(adrFileRe, name) }

// canonADRNum is the shared body of the two above: match, trim the leading zeros,
// render the canonical handle.
//
// The trim is TEXTUAL, never an integer parse. A minted id is 16 digits and an
// ordinal is 4, but neither the grammar nor any record bounds the width — and a
// parse that overflows returns "", which reads to every caller as "not a record"
// and drops a present decision in silence. Trimming text has no such edge and
// agrees with the parse for every number a parse can represent.
func canonADRNum(re *regexp.Regexp, s string) string {
	m := re.FindStringSubmatch(s)
	if m == nil {
		return ""
	}
	// An all-zero number is not an id: no adr-0 was ever issued, and admitting one
	// would let `0000-x.md` claim a handle no citation can name.
	trimmed := strings.TrimLeft(m[1], "0")
	if trimmed == "" {
		return ""
	}
	return "adr-" + trimmed
}

// maxScanEntries bounds how many directory entries one resolver may read across
// every family. A repository with more records than this is pathological or
// hostile; the scan refuses rather than growing unboundedly, and refusing is safe
// because the caller fails closed on the error.
const maxScanEntries = 20000

// familyRoots names each id-bearing family's store, repo-relative and
// slash-separated. Written once, in the order a reader expects them.
var familyRoots = []struct {
	prefix string
	dir    string
}{
	{"itd", ".abcd/development/intents"},
	{"spc", ".abcd/development/specs"},
	{"iss", ".abcd/work/issues"},
	{"adr", ".abcd/development/decisions/adrs"},
}

// Resolver is the live record-id set of one repository, read once. It is a
// snapshot: build it at the start of a validation pass and use it throughout, so
// every citation in one payload is judged against the same view of the record.
type Resolver struct {
	ids map[string]string
}

// NewResolver reads repoRoot's record families and returns the resolver over
// them.
//
// An ABSENT family is not an error — a repository need not carry every store, and
// its ids simply do not resolve. An unreadable family that IS present IS an
// error: reporting "this id names no record" when the truth is "the store could
// not be read" would refuse a correct citation, so the scan fails closed and lets
// the caller say what actually happened.
func NewResolver(repoRoot string) (*Resolver, error) {
	r := &Resolver{ids: map[string]string{}}
	budget := maxScanEntries
	for _, fam := range familyRoots {
		if err := r.scanFamily(repoRoot, fam.prefix, fam.dir, &budget); err != nil {
			return nil, err
		}
	}
	return r, nil
}

// Lookup returns the repo-relative path of the record id, and whether it exists.
// A malformed id simply does not resolve — callers that must distinguish
// "malformed" from "absent" check CitedIDRe first, which they do anyway to bound
// the string before echoing it.
func (r *Resolver) Lookup(id string) (string, bool) {
	p, ok := r.ids[id]
	return p, ok
}

// Len reports how many record ids resolved — the size of the live record.
func (r *Resolver) Len() int { return len(r.ids) }

// scanFamily reads one family root and, one level down, each of its buckets. It
// decrements the shared entry budget so the ceiling is over the whole scan rather
// than per family.
func (r *Resolver) scanFamily(repoRoot, prefix, relDir string, budget *int) error {
	entries, err := readDirIfPresent(filepath.Join(repoRoot, filepath.FromSlash(relDir)))
	if err != nil {
		return fmt.Errorf("recordid: reading %s: %w", relDir, err)
	}
	for _, e := range entries {
		if *budget--; *budget < 0 {
			return fmt.Errorf("recordid: more than %d record files; refusing to scan further", maxScanEntries)
		}
		// Never follow a link: a symlinked file could make a record appear to
		// exist, and a symlinked directory could walk the scan out of the repo.
		if e.Type()&os.ModeSymlink != 0 {
			continue
		}
		if !e.IsDir() {
			r.record(prefix, relDir, e.Name())
			continue
		}
		bucketRel := path.Join(relDir, e.Name())
		inner, err := readDirIfPresent(filepath.Join(repoRoot, filepath.FromSlash(bucketRel)))
		if err != nil {
			return fmt.Errorf("recordid: reading %s: %w", bucketRel, err)
		}
		for _, f := range inner {
			if *budget--; *budget < 0 {
				return fmt.Errorf("recordid: more than %d record files; refusing to scan further", maxScanEntries)
			}
			if f.IsDir() || f.Type()&os.ModeSymlink != 0 {
				continue
			}
			r.record(prefix, bucketRel, f.Name())
		}
	}
	return nil
}

// record derives name's id for its family and, when it has one, maps it to the
// file. First writer wins, so a duplicate id across buckets keeps the first path
// in scan order — the resolver answers "does it exist", and any duplicate is the
// record-lint uniqueness rules' business, not this one's.
func (r *Resolver) record(prefix, relDir, name string) {
	id := fileID(prefix, name)
	if id == "" {
		return
	}
	if _, seen := r.ids[id]; seen {
		return
	}
	r.ids[id] = path.Join(relDir, name)
}

// fileID derives a record id from a filename within its family. The three
// slug-prefixed families spell their own id (itd-104-<slug>.md); ADRs carry a
// bare number instead (0037-<slug>.md → adr-37), read through the one exported
// derivation the mint's presence check also calls.
func fileID(prefix, name string) string {
	if prefix == "adr" {
		return ADRFileID(name)
	}
	m := idRe(prefix).FindStringSubmatch(name)
	if m == nil {
		return ""
	}
	n, err := strconv.Atoi(m[1])
	if err != nil || n <= 0 {
		return ""
	}
	// Rebuilt from the parsed integer, never from the matched text, so a
	// zero-padded filename cannot mint a second spelling of the same id.
	return prefix + "-" + strconv.Itoa(n)
}

// readDirIfPresent reads a directory, treating "absent" and "not a directory" as
// an empty listing and refusing a symlinked root outright. Every other fault is
// returned so the caller fails closed.
func readDirIfPresent(dir string) ([]os.DirEntry, error) {
	fi, err := os.Lstat(dir)
	if err != nil {
		// Absent, or a prefix component that is not a directory: both mean "this
		// family is not here", which is a legitimate repository shape.
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
			return nil, nil
		}
		return nil, err
	}
	if !fi.IsDir() || fi.Mode()&os.ModeSymlink != 0 {
		return nil, nil
	}
	return os.ReadDir(dir)
}
