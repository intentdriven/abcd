package lint

// The prose-citation family (prose_citation_resolves): a record id written in a
// record's PROSE must name a record that exists. Prose is the whole document
// except the typed frontmatter fields and fenced code — see WHAT IS READ below.
//
// The ruling this implements: an id written in prose must resolve unless the
// author marks it as illustrative or forward-looking.
//
// record_schema already resolves the typed cross-references in FRONTMATTER, and
// the citation family already checks URLs. Between them sits the thing a session
// actually writes: a sentence. A spec composed during an autonomous run carried
// two invented ids in its body and was caught only by a human reading the
// document back afterwards — nothing in the gates reads a record's prose for the
// handles it names (iss-2609100518527863).
//
// There is exactly ONE resolver in this binary and this rule does not add a
// second: recordid.Resolver answers whether an id names a record, and
// recordid.CanonCitedID folds the spelling a sentence used into the spelling that
// resolver keys on. This file contributes the two things neither of them can
// know — where in a document a citation is, and which unresolvable ones the
// record has already ruled on.
//
// THREE ways an id can legitimately fail to resolve, and each has its own door:
//
//  1. A PLACEHOLDER is not a citation and needs nothing. `itd-N`, `spc-<id>`,
//     `adr-NNNN` are not the cited-id grammar, so they never reach the lookup.
//     The corpus already writes its placeholders this way, so the commonest
//     legitimate case costs an author nothing. A slug does NOT make a handle a
//     placeholder: `itd-160-dangling-...md` cites itd-160 and must resolve like
//     any other citation. Deferring that shape to links_resolve was the earlier
//     reading and it was wrong — links_resolve judges markdown link TARGETS,
//     `[..](..)`, so a bare filename-shaped handle in a sentence was judged by
//     nothing, and appending a slug to an invented id made it invisible.
//
//  2. A NUMERIC placeholder is byte-identical to a citation — `adr-999` in a
//     sentence describing what a gate refuses cannot be told from a citation of
//     a decision — and so is a forward reference to a record not yet minted. The
//     author says which, on the line, with `<!-- record-lint: illustrative -->`
//     or `<!-- record-lint: forward-looking -->`. The marker is scoped to its own
//     line, the way docs-lint's `<!-- docs-lint: allow -->` is: a marker that
//     carried past its line would silence a whole document from one sentence.
//
//  3. The corpus that predates this gate. 172 such mentions across 59 files were
//     measured when it landed — pruned decisions still narrated by their
//     successors, a predecessor implementation's numbering that was never
//     migrated, records that live on an unmerged design branch. Marking each in
//     place would be a 59-file edit asserting a ruling the record has not made
//     (iss-179 and iss-2608271804496010 both still owe it), so they are carried
//     in a committed BASELINE instead, per id, each entry saying which class it
//     is and why. The ratchet is the point: a new unresolvable id fails even in a
//     file the baseline already names.
//
//     A baseline entry is a GLOBAL licence: it excuses its id everywhere, for
//     good. That is the right instrument for an id whose ruling is genuinely
//     owed, and the wrong one for an id whose author already knows the answer —
//     so an id that CAN carry door 2's line marker takes the marker. The
//     corpus's three illustrative ids are marked at each of their sites for
//     exactly this reason; door 3 is for what the marker cannot say.
//
// The baseline is keyed on the ID, never on the file. An issue record moves from
// open/ to resolved/ and an intent from drafts/ to planned/ as a matter of
// routine, so a path-keyed entry would go stale on a move that changed nothing
// about the citation — a gate that fails on lifecycle transitions is a gate that
// gets disabled.
//
// The rule does NOT consult exempt_paths, for the reason record_schema and
// cross_store_id_claim do not: being historical is not a licence to name an id
// that answers to nothing. The two exemptions this rule HAS — the line marker and
// the baseline — are both visible in a diff, which a path exemption is not.
//
// WHAT IS READ, and what is not:
//
//   - The whole document, FRONTMATTER INCLUDED, minus the frontmatter lines whose
//     key is one of the typed cross-reference fields record_schema already
//     resolves (recordParsedFields, and the indented block below such a key).
//     Everything else in a frontmatter block is free text — `deferral_reason`,
//     `found_during`, `resolution`, a `kind_notes` sentence — and free text is
//     exactly where an unchecked id hides. Reading only the body left those
//     fields covered by neither gate.
//   - FENCED code is excluded, by fenceMask, which is mdrecord's fence rule:
//     backtick and tilde fences both count. Four-space indented code is not a
//     fence, and a line the fence rules disagree about is read as prose. An id
//     in either is read as prose and must resolve or carry a marker — which is
//     the safe direction to be wrong in, and is stated here so an author who
//     meets it knows why.
//   - Nine record stores are configured, but only FOUR families resolve
//     (recordid.familyRoots: adr, itd, iss, spc). The `rdi`, `dsp`, `rdg`, `adm`
//     and `srp` stores are scanned as FILES — their prose is read like any
//     other — but their own ids are not the cited-id grammar, so a mention of
//     `rdg-2609022207359936` is not a citation and is not checked by anything
//     here.

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"syscall"

	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
)

const (
	// ruleProseCitationResolves is the gate: an unresolvable id in a record body.
	ruleProseCitationResolves = "prose_citation_resolves"
	// ruleProseCitationBaselineStale is the ratchet's shrink half, reported under
	// its own id so it can never be mistaken for the gate. A baseline entry that
	// nothing needs any more is good news, not a fault.
	ruleProseCitationBaselineStale = "prose_citation_baseline_stale"
)

// DefaultProseBaselinePath is the committed baseline's repo-relative home. It
// sits beside the other committed machine records under .abcd/ — record-lint.json,
// citations-baseline.json — because it is a record the gate reads on every commit.
const DefaultProseBaselinePath = ".abcd/prose-citations-baseline.json"

// ProseBaselineSchemaVersion is the only baseline schema this build understands.
// A document declaring anything else is refused rather than best-effort parsed:
// a gate must never enforce a record whose meaning it is guessing at.
const ProseBaselineSchemaVersion = 1

// proseBaselineSizeLimit caps the baseline read. It is a committed file, but the
// guarded read is what keeps a hostile branch from handing the gate something
// other than a record.
const proseBaselineSizeLimit = 1 << 20 // 1 MiB

// proseCitationRe finds a candidate handle in a line. The LEADING boundary is
// \b, so a handle inside a word (`xitd-5`) never matches; the trailing boundary
// is applied in code because Go's regexp has no lookahead, and consuming the byte
// after the match would make two adjacent handles read as one.
var proseCitationRe = regexp.MustCompile(`(?i)\b(?:adr|itd|iss|spc)-[0-9]+`)

// proseEscapeRe matches the line-scoped exemption marker in both its spellings.
// The vocabulary is exactly the ruling's own two words, and there is no third,
// blanket spelling: a general `allow` would re-open the silent exemption this
// rule exists to close, since a reader could not tell an illustration from a
// citation nobody checked.
var proseEscapeRe = regexp.MustCompile(`(?i)<!--\s*record-lint:\s*(illustrative|forward-looking)\b`)

// proseTypedFrontmatterKeys are the frontmatter keys this rule does NOT read,
// because record_schema (and, for the intent<->spec pair, spec_links) already
// resolves their targets and says so in its own vocabulary. The set is taken
// from recordParsedFields rather than spelled a second time: a field added there
// must not silently become a field this rule double-reports.
//
// Every OTHER frontmatter key is free text and IS read. That is the point — a
// `deferral_reason:` sentence is prose that happens to sit above the `---`, and
// before this it was read by neither gate.
var proseTypedFrontmatterKeys = func() map[string]bool {
	m := make(map[string]bool, len(recordParsedFields))
	for _, f := range recordParsedFields {
		m[f] = true
	}
	return m
}()

// proseFrontmatterKeyRe captures the key of a `key: value` frontmatter line.
var proseFrontmatterKeyRe = regexp.MustCompile(`^([A-Za-z0-9_]+)\s*:`)

// proseFrontmatterSkip marks, over the frontmatter region only, the lines whose
// key is typed — and the indented or sequence lines that carry that key's block
// value, since `supersedes:` followed by `  - adr-12` holds its handles below the
// key rather than beside it.
//
// `end` is recordBodyStart's answer, so the region this walks is exactly the
// region the body scan does not: the two cannot disagree about where prose
// begins. Lines before the opening `---` (a leading attribution comment, blanks)
// carry no key and so are never skipped.
func proseFrontmatterSkip(lines []string, end int) []bool {
	skip := make([]bool, end)
	inTyped := false
	for i := 0; i < end && i < len(lines); i++ {
		line := lines[i]
		if m := proseFrontmatterKeyRe.FindStringSubmatch(line); m != nil {
			inTyped = proseTypedFrontmatterKeys[m[1]]
			skip[i] = inTyped
			continue
		}
		trimmed := strings.TrimLeft(line, " \t")
		continuation := len(trimmed) < len(line) || strings.HasPrefix(line, "- ")
		if continuation && strings.TrimSpace(line) != "" {
			skip[i] = inTyped
			continue
		}
		inTyped = false
	}
	return skip
}

// proseBaselineClasses are the ways a carried id can legitimately not resolve.
// They are the taxonomy the corpus measurement found, and declaring one is
// mandatory: an exemption that does not say WHAT it is exempting is an allowlist
// entry, and an allowlist grows silently.
var proseBaselineClasses = map[string]bool{
	// The record existed in this tree and was removed — a superseded ADR pruned
	// by its successor, an issue consumed by promotion.
	"pruned": true,
	// The id was used by a predecessor implementation's numbering, or released
	// without ever naming a file here.
	"never-minted": true,
	// A numeric id written to demonstrate behaviour rather than to cite, at a
	// site that CANNOT carry the line marker — the marker is the ordinary door
	// for this class, and a baseline entry is a GLOBAL licence: every future
	// mention of that id, anywhere, passes unmarked. No entry uses this class
	// today; the corpus's illustrative ids (itd-9999, itd-1990,
	// iss-2608201142077341) are each marked on their own line instead. It stays
	// as a class because one shape genuinely cannot be marked — a value a reader
	// consumes verbatim, such as the `slug:` field — and the alternative to a
	// named class there is an entry that lies about which door it used.
	"illustrative": true,
	// A record that is not in this tree yet — typically one living on an
	// unmerged branch.
	"forward-looking": true,
	// A probable invented or mistyped id: a defect to correct, carried so the
	// gate can land, and listed here so it is findable.
	"suspect": true,
}

// ProseBaselineEntry is one carried id and the ruling that carries it.
type ProseBaselineEntry struct {
	// ID is the canonical handle (lower-case family, unpadded number).
	ID string `json:"id"`
	// Class is one of proseBaselineClasses.
	Class string `json:"class"`
	// Note says why this id does not resolve, in the author's words. Required:
	// the baseline is a ruling, and a ruling with no reasoning is an allowlist.
	Note string `json:"note"`
}

// ProseBaseline is the committed record of ids the gate carries.
type ProseBaseline struct {
	SchemaVersion int                  `json:"schema_version"`
	IDs           []ProseBaselineEntry `json:"ids"`
}

// checkProseCitations implements the rule. It fails closed on the one way an
// armed gate could check nothing and still report clean: configured stores that
// yield no record files, which looks exactly like a corpus with no faults.
func checkProseCitations(repoRoot string, cfg RuleConfig) ([]Finding, error) {
	if len(cfg.RecordStores) == 0 {
		return nil, &configError{ruleProseCitationResolves +
			": no record_stores configured; the rule would read no prose and report clean"}
	}

	files, err := proseRecordFiles(repoRoot, cfg.RecordStores)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, &configError{ruleProseCitationResolves +
			": the configured record stores hold no record files; the gate would pass by not looking"}
	}

	resolver, err := recordid.NewResolver(repoRoot)
	if err != nil {
		return nil, &configError{ruleProseCitationResolves + ": " + err.Error()}
	}

	baselinePath := cfg.Baseline
	if baselinePath == "" {
		baselinePath = DefaultProseBaselinePath
	}
	baseline, err := loadProseBaseline(repoRoot, baselinePath)
	if err != nil {
		return nil, err
	}

	var out []Finding
	used := map[string]bool{}
	for _, abs := range files {
		fs, err := proseCitationsInFile(repoRoot, abs, resolver, baseline, used, cfg.Severity)
		if err != nil {
			return nil, err
		}
		out = append(out, fs...)
	}

	// The shrink half of the ratchet. An entry whose id now resolves, or that no
	// prose cites any more, has done its job; saying so is how the backlog is seen
	// to drain rather than quietly outliving the thing it excused.
	ids := make([]string, 0, len(baseline))
	for id := range baseline {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if used[id] {
			continue
		}
		reason := "no prose in the record stores cites it any more"
		if _, ok := resolver.Lookup(id); ok {
			reason = "it now resolves to " + mustLookup(resolver, id)
		}
		out = append(out, Finding{
			File: baselinePath, Line: 0, RuleID: ruleProseCitationBaselineStale, Severity: severityInfo,
			Message: "the carried id " + id + " is spent — " + reason +
				"; drop its entry so the baseline keeps shrinking",
		})
	}
	return out, nil
}

// mustLookup renders a resolved path for a message. The caller has already
// established the id resolves, so the miss branch is unreachable; it returns a
// literal rather than panicking, because a gate must not crash on a race between
// its own two reads.
func mustLookup(r *recordid.Resolver, id string) string {
	if p, ok := r.Lookup(id); ok {
		return p
	}
	return "a record"
}

// proseCitationsInFile reads one record and reports every unresolvable handle its
// BODY names. `used` accumulates which baseline entries were actually needed, so
// the caller can report the spent ones.
func proseCitationsInFile(repoRoot, abs string, resolver *recordid.Resolver, baseline map[string]ProseBaselineEntry, used map[string]bool, severity string) ([]Finding, error) {
	data, err := fsutil.ReadGuarded(abs, citationPageSizeLimit)
	if err != nil {
		// A record that vanished mid-walk (a checkout racing the gate) is not a
		// finding about the record; every other fault is returned so the caller
		// fails closed rather than reporting a file it never read as clean.
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, &configError{ruleProseCitationResolves + ": reading " + repoRel(repoRoot, abs) + ": " + err.Error()}
	}
	rel := repoRel(repoRoot, abs)
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	mask := fenceMask(lines)

	skip := proseFrontmatterSkip(lines, recordBodyStart(lines))

	var out []Finding
	for i := 0; i < len(lines); i++ {
		if mask[i] || (i < len(skip) && skip[i]) {
			continue
		}
		line := lines[i]
		escaped := proseEscapeRe.MatchString(line)
		seen := map[string]bool{}
		for _, id := range proseCitedIDs(line) {
			if seen[id] {
				// A line naming one absent record twice is one defect.
				continue
			}
			seen[id] = true
			if _, ok := resolver.Lookup(id); ok {
				continue
			}
			if _, carried := baseline[id]; carried {
				used[id] = true
				continue
			}
			if escaped {
				continue
			}
			out = append(out, Finding{
				File: rel, Line: i + 1, RuleID: ruleProseCitationResolves, Severity: severity,
				Message: proseCitationMessage(id),
			})
		}
	}
	return out, nil
}

// proseCitationMessage is the refusal, and it is where an author learns the
// convention: a gate whose message does not teach its own escape is a gate people
// route around.
func proseCitationMessage(id string) string {
	return "prose cites " + id + ", which names no record in this repository. " +
		"An id written in prose must resolve unless the author marks it: correct the id, " +
		"or put `<!-- record-lint: illustrative -->` (a number used to demonstrate behaviour) or " +
		"`<!-- record-lint: forward-looking -->` (a record not minted yet) on the same line. " +
		"A placeholder written with a letter — itd-N, spc-<id> — is not a citation and needs no marker, " +
		"but a slug does not stop an id being an id: itd-N-some-slug.md cites itd-N. " +
		"The baseline at " + DefaultProseBaselinePath + " carries the ids that predate this gate; it ratchets down, never up."
}

// proseCitedIDs returns the canonical ids a line cites, in order. A candidate is
// dropped when the byte on either side continues the token — `xitd-5` and
// `adr-4x` are words that happen to contain a handle's bytes, not handles.
//
// A HYPHEN does not continue a handle, on either side. It is the byte that joins
// a handle to its slug (`iss-2608231243286557-the-renderer-half.md`) and the byte
// that can precede one inside a compound (`pre-adr-999`), and reading either as
// "not a citation" is how an unresolvable id becomes invisible: appending a slug
// to an invented id was enough to silence the gate, and links_resolve does not
// cover the gap — it judges markdown link TARGETS, `[..](..)`, so a bare
// filename-shaped handle in a sentence was judged by nothing at all. A slug does
// not stop an id being an id: `adr-4-slug.md` cites adr-4 and resolves or fails
// like any other citation.
func proseCitedIDs(line string) []string {
	var out []string
	for _, loc := range proseCitationRe.FindAllStringIndex(line, -1) {
		if loc[0] > 0 && continuesHandle(line[loc[0]-1]) {
			continue
		}
		if loc[1] < len(line) && continuesHandle(line[loc[1]]) {
			continue
		}
		if id := recordid.CanonCitedID(line[loc[0]:loc[1]]); id != "" {
			out = append(out, id)
		}
	}
	return out
}

// continuesHandle reports whether b would make the adjacent text part of a longer
// WORD rather than a bare handle: letters, digits and underscore only. It is the
// same set on both sides, and on the left it agrees with the regexp's own leading
// \b — stated in code so the two boundaries cannot drift apart.
func continuesHandle(b byte) bool {
	switch {
	case b >= '0' && b <= '9', b >= 'a' && b <= 'z', b >= 'A' && b <= 'Z':
		return true
	case b == '_':
		return true
	}
	return false
}

// proseRecordFiles lists every markdown file under the configured stores, sorted
// and de-duplicated. Stores can nest (the issue ledger holds its sibling record
// families), so one file must not be linted — or counted — twice.
func proseRecordFiles(repoRoot string, stores map[string]string) ([]string, error) {
	prefixes := make([]string, 0, len(stores))
	for p := range stores {
		prefixes = append(prefixes, p)
	}
	sort.Strings(prefixes)
	seen := map[string]bool{}
	var out []string
	for _, p := range prefixes {
		abs := filepath.Join(repoRoot, filepath.FromSlash(stores[p]))
		files, err := markdownFiles(abs)
		if err != nil {
			return nil, &configError{ruleProseCitationResolves + ": walking " + stores[p] + ": " + err.Error()}
		}
		for _, f := range files {
			if seen[f] {
				continue
			}
			seen[f] = true
			out = append(out, f)
		}
	}
	sort.Strings(out)
	return out, nil
}

// loadProseBaseline reads the committed baseline, keyed by id.
//
// An ABSENT baseline is an empty one, which is the STRICT reading — nothing is
// carried — and so is safe for a repository adopting the rule with no backlog. A
// baseline that is present and unreadable is an error: carrying on with an empty
// map would silently disarm every exemption the record made.
//
// A PRESENT but EMPTY file is refused, not read as absent. A truncated or
// half-written baseline is the shape a failed write leaves behind, and treating
// it as "nothing carried" would turn a lost file into a corpus-wide failure that
// looks like the gate working. The refusal names the minimal valid document, so
// an author who meant to carry nothing can write it in one line.
func loadProseBaseline(repoRoot, rel string) (map[string]ProseBaselineEntry, error) {
	// The path comes out of the committed config and the file is an exemption
	// list, so a baseline read from outside the tree would disarm the gate with
	// content the repository does not hold: it is read only inside the root, and
	// never through a link (iss-2609261019593167).
	data, err := readRepoLeaf(repoRoot, rel, proseBaselineSizeLimit)
	if err != nil {
		var ce *configError
		if errors.As(err, &ce) {
			return nil, &configError{ruleProseCitationResolves + ": baseline " + ce.Error()}
		}
		if os.IsNotExist(err) || errors.Is(err, syscall.ENOTDIR) {
			return map[string]ProseBaselineEntry{}, nil
		}
		return nil, &configError{ruleProseCitationResolves + ": reading " + rel + ": " + err.Error()}
	}

	if len(bytes.TrimSpace(data)) == 0 {
		return nil, &configError{ruleProseCitationResolves + ": " + rel +
			": the file is empty. An empty baseline is refused rather than read as \"nothing carried\": " +
			"write the minimal document `{\"schema_version\": 1, \"ids\": []}` to carry nothing, " +
			"or delete the file, which IS read as nothing carried"}
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var doc ProseBaseline
	if err := dec.Decode(&doc); err != nil {
		return nil, &configError{ruleProseCitationResolves + ": " + rel + ": " + err.Error()}
	}
	if doc.SchemaVersion != ProseBaselineSchemaVersion {
		return nil, &configError{ruleProseCitationResolves + ": " + rel + ": unsupported schema_version"}
	}

	out := make(map[string]ProseBaselineEntry, len(doc.IDs))
	for _, e := range doc.IDs {
		switch {
		case recordid.CanonCitedID(e.ID) == "":
			return nil, &configError{ruleProseCitationResolves + ": " + rel + ": " + quote(e.ID) +
				" is not a record id (adr-N, itd-N, iss-N, spc-N)"}
		case recordid.CanonCitedID(e.ID) != e.ID:
			return nil, &configError{ruleProseCitationResolves + ": " + rel + ": " + quote(e.ID) +
				" is not canonical; write it as " + recordid.CanonCitedID(e.ID) +
				", the spelling the gate keys on"}
		case !proseBaselineClasses[e.Class]: // includes the empty class
			return nil, &configError{ruleProseCitationResolves + ": " + rel + ": " + e.ID +
				" declares class " + quote(e.Class) + "; it must be one of " + proseClassList()}
		case strings.TrimSpace(e.Note) == "":
			return nil, &configError{ruleProseCitationResolves + ": " + rel + ": " + e.ID +
				" carries no note; a carried id must say why it does not resolve"}
		}
		if _, dup := out[e.ID]; dup {
			return nil, &configError{ruleProseCitationResolves + ": " + rel + ": " + e.ID + " is listed twice"}
		}
		out[e.ID] = e
	}
	return out, nil
}

// proseClassList renders the legal class set for a refusal, composed from the map
// above rather than spelled a second time.
func proseClassList() string {
	names := make([]string, 0, len(proseBaselineClasses))
	for c := range proseBaselineClasses {
		names = append(names, c)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
