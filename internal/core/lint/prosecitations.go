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
// FOUR ways an id can legitimately fail to resolve, and each has its own door
// (the fourth, a record still in review, is THE PENDING REFERENCE at the end of
// this file):
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
//   - Eleven record stores are configured, but only FIVE families resolve
//     (recordid.familyRoots: adr, itd, iss, spc, dcn). The `rdi`, `dsp`, `rdg`,
//     `adm`, `srp` and `rfm` stores are scanned as FILES — their prose is read like any
//     other — but their own ids are not the cited-id grammar, so a mention of
//     `rdg-2609022207359936` is not a citation and is not checked by anything
//     here.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/intentdriven/abcd/internal/core/mdrecord"
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
var proseCitationRe = regexp.MustCompile(`(?i)\b(?:adr|itd|iss|spc|dcn)-[0-9]+`)

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
	// A graph field nothing resolves stays prose to this rule.
	for _, f := range recordUnjudgedGraphFields {
		delete(m, f)
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
	// mention of that id, anywhere, passes unmarked. The corpus's illustrative
	// ids (itd-1990, iss-2608201142077341) are each marked on their own line
	// instead. It stays as a class because some shapes genuinely cannot be
	// marked — a value a reader consumes verbatim, such as the `slug:` field, or
	// a decision record migrated verbatim from the retired single log, which
	// carries itd-9999 and adr-9999 — and the alternative to a named class there
	// is an entry that lies about which door it used.
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
	in, err := proseInputs(repoRoot, cfg)
	if err != nil {
		return nil, err
	}
	files, resolver, baseline, baselinePath := in.files, in.resolver, in.baseline, in.baselinePath

	// Every file is read before any finding is made, so the ids no file's tree
	// resolves are asked of the pending lookup once, together: the forge is a
	// network read, and one listing answers every citation in the corpus.
	type fileCites struct {
		rel   string
		cites []ProseCitation
	}
	var all []fileCites
	used := map[string]bool{}
	for _, abs := range files {
		rel, cites, err := proseCitationsInFile(repoRoot, abs, resolver, baseline, used)
		if err != nil {
			return nil, err
		}
		if len(cites) > 0 {
			all = append(all, fileCites{rel: rel, cites: cites})
		}
	}
	var missing []ProseCitation
	for _, fc := range all {
		missing = append(missing, fc.cites...)
	}
	pending := lookupPending(repoRoot, missing)

	var out []Finding
	for _, fc := range all {
		for _, c := range fc.cites {
			if rec, ok := pending.found[c.ID]; ok {
				out = append(out, Finding{
					File: fc.rel, Line: c.Line, RuleID: ruleProseCitationPending, Severity: severityInfo,
					Message: pendingCitationMessage(rec),
				})
				continue
			}
			out = append(out, Finding{
				File: fc.rel, Line: c.Line, RuleID: ruleProseCitationResolves, Severity: cfg.Severity,
				Message: proseCitationMessage(c.ID) + pending.refusalNote(),
			})
		}
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
		if resolver.Has(id) {
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
// its own two reads. An id two files claim renders as every claimant: the
// message is about the citation, and the duplicate is the uniqueness rules'
// blocker, reported on its own.
func mustLookup(r *recordid.Resolver, id string) string {
	p, ok, err := r.Lookup(id)
	var amb *recordid.AmbiguousIDError
	switch {
	case errors.As(err, &amb):
		return "more than one record (" + strings.Join(amb.Paths, ", ") + ")"
	case ok:
		return p
	}
	return "a record"
}

// proseCitationsInFile reads one record and returns its repo-relative path and
// every unresolvable handle its prose names. `used` accumulates which baseline
// entries were actually needed, so the caller can report the spent ones.
func proseCitationsInFile(repoRoot, abs string, resolver *recordid.Resolver, baseline map[string]ProseBaselineEntry, used map[string]bool) (string, []ProseCitation, error) {
	rel, lines, skip, err := readProseRecord(repoRoot, abs)
	if err != nil || lines == nil {
		return rel, nil, err
	}
	return rel, unresolvedProseCitations(lines, skip, resolver, baseline, used), nil
}

// readProseRecord reads one record the rule judges: its repo-relative path, its
// lines and the frontmatter lines the rule does not read. A record that vanished
// mid-walk (a checkout racing the gate) returns no lines and no error, since it
// is not a finding about the record; every other fault is returned so the caller
// fails closed rather than reporting a file it never read as clean.
func readProseRecord(repoRoot, abs string) (string, []string, []bool, error) {
	rel := repoRel(repoRoot, abs)
	data, err := fsutil.ReadGuarded(abs, citationPageSizeLimit)
	if err != nil {
		if os.IsNotExist(err) {
			return rel, nil, nil, nil
		}
		return rel, nil, nil, &configError{ruleProseCitationResolves + ": reading " + rel + ": " + err.Error()}
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	return rel, lines, proseFrontmatterSkip(lines, recordBodyStart(lines)), nil
}

// ProseCitation is one record id a line of prose cites that the rule refuses:
// it names no record, the baseline does not carry it, and its line carries no
// escape marker. Line is 1-based.
type ProseCitation struct {
	Line int
	ID   string
}

// unresolvedProseCitations is the rule's reading of a body's lines, shared by
// the gate and by a writer that asks before it writes, so the two cannot
// disagree about what the gate refuses. Fenced lines and the lines skip marks
// are not read.
func unresolvedProseCitations(lines []string, skip []bool, resolver *recordid.Resolver, baseline map[string]ProseBaselineEntry, used map[string]bool) []ProseCitation {
	var out []ProseCitation
	for _, pl := range readProseLines(lines, skip) {
		for _, id := range pl.ids {
			if resolver.Has(id) {
				continue
			}
			if _, carried := baseline[id]; carried {
				used[id] = true
				continue
			}
			if pl.illustrative || pl.forwardLooking {
				continue
			}
			out = append(out, ProseCitation{Line: pl.line, ID: id})
		}
	}
	return out
}

// proseLine is one line of prose the rule reads and that cites at least one
// id: the ids it cites, each once, in order, and the markers it carries.
type proseLine struct {
	line int // 1-based
	ids  []string
	// illustrative and forwardLooking are the line escape as the gate reads it:
	// the marker anywhere on the line.
	illustrative, forwardLooking bool
	// forwardMarker is a forward-looking marker outside every inline code span
	// (mdrecord.CodeSpanRanges, the one reading of a span): a marker, not a
	// sentence about the marker, which is what a report asking for the marker's
	// removal must be about. The spans are paired on the line alone, whatever
	// the lines above it leave open, so a marker quoted on its line is a
	// mention even where the page may pair a backtick over the line break.
	forwardMarker bool
}

// readProseLines is the one reading of a record's lines that the gate, the
// writer's question and the release cut's audit share, so the three cannot
// disagree about what a line cites or which marker it carries. Fenced lines,
// the lines skip marks, and lines that cite no id are not returned.
func readProseLines(lines []string, skip []bool) []proseLine {
	mask := fenceMask(lines)
	var out []proseLine
	for i := 0; i < len(lines); i++ {
		if mask[i] || (i < len(skip) && skip[i]) {
			continue
		}
		line := lines[i]
		pl := proseLine{line: i + 1}
		seen := map[string]bool{}
		for _, id := range proseCitedIDs(line) {
			if seen[id] {
				// A line naming one record twice cites it once.
				continue
			}
			seen[id] = true
			pl.ids = append(pl.ids, id)
		}
		if len(pl.ids) == 0 {
			continue
		}
		for _, m := range proseEscapeRe.FindAllStringSubmatch(line, -1) {
			switch strings.ToLower(m[1]) {
			case "illustrative":
				pl.illustrative = true
			case "forward-looking":
				pl.forwardLooking = true
			}
		}
		spans := mdrecord.CodeSpanRanges(line)
		for _, m := range proseEscapeRe.FindAllStringSubmatchIndex(line, -1) {
			if strings.EqualFold(line[m[2]:m[3]], "forward-looking") && !mdrecord.InAnyRange(spans, m[0]) {
				pl.forwardMarker = true
			}
		}
		out = append(out, pl)
	}
	return out
}

// DefaultRecordLintConfigPath is the repository's record-lint configuration,
// repo-relative: the one a writer consults to learn what the gate will refuse.
const DefaultRecordLintConfigPath = ".abcd/record-lint.json"

// UnresolvedProseCitationsInRecord is UnresolvedProseCitationsInText under the
// repository's own record-lint configuration. A repository with no
// configuration gates nothing, and the answer is empty; a configuration that
// cannot be read is an error, since the question cannot then be answered.
func UnresolvedProseCitationsInRecord(repoRoot, rel, text string) ([]ProseCitation, error) {
	cfg, err := LoadConfig(filepath.Join(repoRoot, filepath.FromSlash(DefaultRecordLintConfigPath)))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading %s: %w", DefaultRecordLintConfigPath, err)
	}
	return UnresolvedProseCitationsInText(cfg, repoRoot, rel, text)
}

// UnresolvedProseCitationsInText runs prose_citation_resolves over text a
// writer is about to put into the record at rel, BEFORE it is written: the one
// resolver, the committed baseline and the line escape marker, read exactly as
// the gate reads them. A writer composing a record from a host-delegated
// payload asks this first, because a payload the writer accepts must never
// produce a record the gate then refuses (iss-2609231036448320). A
// configuration that does not arm the rule, or whose record stores do not hold
// rel, gates nothing, and the answer is empty. text is a fragment of a record
// body, never its frontmatter.
func UnresolvedProseCitationsInText(cfg Config, repoRoot, rel, text string) ([]ProseCitation, error) {
	rc, on := cfg.Rules[ruleProseCitationResolves]
	if !on || !rc.Enabled || (!underAnyStore(rel, rc.RecordStores) && !underAnyExtraRoot(rel, rc.ExtraRoots)) {
		return nil, nil
	}
	resolver, err := recordid.NewResolver(repoRoot)
	if err != nil {
		return nil, &configError{ruleProseCitationResolves + ": " + err.Error()}
	}
	baselinePath := rc.Baseline
	if baselinePath == "" {
		baselinePath = DefaultProseBaselinePath
	}
	baseline, err := loadProseBaseline(repoRoot, baselinePath)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	cites := unresolvedProseCitations(lines, nil, resolver, baseline, map[string]bool{})
	// A record in review is cited by id here as in the gate: the writer asks
	// the gate's own pending question, so it neither refuses what the gate
	// accepts nor pushes the author to reword a citation the gate would take.
	pending := lookupPending(repoRoot, cites)
	out := cites[:0]
	for _, c := range cites {
		if _, ok := pending.found[c.ID]; !ok {
			out = append(out, c)
		}
	}
	return out, nil
}

// underAnyStore reports whether the repo-relative path rel lies inside one of
// the configured record stores.
func underAnyStore(rel string, stores map[string]string) bool {
	rel = filepath.ToSlash(filepath.Clean(rel))
	for _, dir := range stores {
		dir = strings.TrimSuffix(filepath.ToSlash(filepath.Clean(dir)), "/")
		if strings.HasPrefix(rel, dir+"/") {
			return true
		}
	}
	return false
}

// underAnyExtraRoot reports whether rel is one of the rule's extra roots or sits
// beneath one: an entry names a directory or a single file.
func underAnyExtraRoot(rel string, roots []string) bool {
	rel = filepath.ToSlash(filepath.Clean(rel))
	for _, r := range roots {
		r = strings.TrimSuffix(filepath.ToSlash(filepath.Clean(r)), "/")
		if rel == r || strings.HasPrefix(rel, r+"/") {
			return true
		}
	}
	return false
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
		"A record still in review is cited by id too, never in words: an id naming a record on an open pull request's " +
		"branch on origin is accepted as a pending reference, and the release cut refuses until that record lands. " +
		"The baseline at " + DefaultProseBaselinePath + " carries the ids that predate this gate; it ratchets down, never up."
}

// pendingCitationMessage is the report for a citation the tree does not resolve
// that names a record in review. It says where the record is, and where it must
// have landed: the cut, not this change, is the gate (iss-2611031731385043).
func pendingCitationMessage(rec PendingRecord) string {
	return "prose cites " + rec.ID + ", which is not in this tree yet: it names " + rec.Path +
		" on open pull request #" + strconv.Itoa(rec.PR) + " (" + rec.Branch + "), still in review. " +
		"It is accepted as a pending reference and holds nothing up now; the release cut refuses while any " +
		"reference names a record that has not reached main, so land that pull request, or correct the reference, " +
		"before the next release. Keep the citation by id, never in words."
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

// proseExtraRootFiles lists the markdown files under the rule's extra_roots: the
// parts of the durable record that are not a record store — the brief, the
// principles, the roadmap — whose prose names records as surely as a record's
// does (iss-2608271804497247). An entry is a directory or a single file, held
// inside the repository the way links_resolve holds its own extra roots, and an
// entry that does not exist is refused: a configured tree that does not resolve
// would disarm the rule for it without a word.
func proseExtraRootFiles(repoRoot string, roots []string) ([]string, error) {
	var out []string
	for _, root := range roots {
		if err := containedRepoPath(root); err != nil {
			return nil, &configError{ruleProseCitationResolves + " extra_roots entry " + quote(root) + " " + err.Error() +
				"; the lint reads only inside the repository"}
		}
		rootAbs := filepath.Join(repoRoot, filepath.FromSlash(root))
		if err := resolvedInsideRoot(repoRoot, rootAbs); err != nil {
			return nil, &configError{ruleProseCitationResolves + " extra_roots entry " + quote(root) + " " + err.Error() +
				"; the lint reads only inside the repository"}
		}
		if _, err := os.Stat(rootAbs); err != nil {
			if os.IsNotExist(err) {
				return nil, &configError{ruleProseCitationResolves + " extra_roots entry " + quote(root) +
					" does not exist; a configured tree that does not resolve silently disarms the rule for it"}
			}
			return nil, err
		}
		files, err := markdownFiles(rootAbs)
		if err != nil {
			return nil, &configError{ruleProseCitationResolves + ": walking " + root + ": " + err.Error()}
		}
		out = append(out, files...)
	}
	return out, nil
}

// mergeFileLists joins two file lists into one sorted list with no repeats, so a
// file an extra root shares with a store is read once.
func mergeFileLists(a, b []string) []string {
	seen := make(map[string]bool, len(a)+len(b))
	out := make([]string, 0, len(a)+len(b))
	for _, f := range append(append([]string{}, a...), b...) {
		if seen[f] {
			continue
		}
		seen[f] = true
		out = append(out, f)
	}
	sort.Strings(out)
	return out
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
				" is not a record id (" + recordid.CitedIDForms() + ")"}
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

// THE PENDING REFERENCE (iss-2611031731385043). A fourth way an id can fail to
// resolve here and still be a true citation: its record is minted and in review,
// on another open pull request's branch, and has not reached this tree. The
// product thinker's ruling is that records are cited by id, never in words, so
// a decision resting on a ruling still in review carries that ruling's id from
// the start; and that a pending reference holds nothing up when it is written,
// nor while its record is in review, because the release cut is the gate.
//
// So the gate asks, for the ids it would otherwise refuse, whether an open pull
// request's branch on origin carries them (SetPendingLookup; the front doors
// register internal/core/pendingref). An id one does is reported under its own
// rule id, prose_citation_pending, at info, naming the pull request; the cut
// (AuditReferences) refuses while any reference names a record not in the tree
// being cut. The lookup is asked only for what the tree lacks, once per run, so
// a corpus whose every citation resolves never touches the network; and an
// answer it could not give leaves the refusal standing, with the reason.

// ruleProseCitationPending is the report for a pending reference. It is its own
// rule id for the reason ruleProseCitationBaselineStale is: a reader of the
// output must never mistake an accepted reference for the gate's refusal.
const ruleProseCitationPending = "prose_citation_pending"

// PendingRecord is a record an open pull request's branch on origin carries and
// the tree being linted does not.
type PendingRecord struct {
	ID     string
	Path   string // repo-relative, on that branch
	PR     int
	Branch string
}

// PendingLookup answers which of ids name a record on an open pull request's
// branch on origin, and, as unread, why some branches were not read (empty when
// every one was). It is internal/core/pendingref.Lookup in this package's
// vocabulary, registered by the front doors (cmd/record-lint, and the CLI for
// `abcd lint` and every host-prose ingest) for the reason issueReadRefusal is:
// the forge client lives in core/landing, which imports this package.
type PendingLookup func(repoRoot string, ids []string) (found map[string]PendingRecord, unread string)

// pendingLookup is the registered lookup, nil when no front door registered
// one: every unresolved id is then refused, as it was before the lookup
// existed, which is the safe direction.
var pendingLookup PendingLookup

// SetPendingLookup registers the pending-reference lookup the gate and the
// writer's question ask. Pass an adapter over pendingref.Lookup.
func SetPendingLookup(fn PendingLookup) { pendingLookup = fn }

// pendingAnswer is one run's answer from the lookup.
type pendingAnswer struct {
	asked  bool
	found  map[string]PendingRecord
	unread string
}

// lookupPending asks the registered lookup, once, about the distinct ids of
// cites; it asks nothing when there is nothing to ask or no lookup.
func lookupPending(repoRoot string, cites []ProseCitation) pendingAnswer {
	if pendingLookup == nil || len(cites) == 0 {
		return pendingAnswer{}
	}
	seen := map[string]bool{}
	var ids []string
	for _, c := range cites {
		if !seen[c.ID] {
			seen[c.ID] = true
			ids = append(ids, c.ID)
		}
	}
	sort.Strings(ids)
	found, unread := pendingLookup(repoRoot, ids)
	return pendingAnswer{asked: true, found: found, unread: unread}
}

// refusalNote is what a refusal adds once the pending lookup was asked: that no
// open pull request carries the id, or why the lookup could not say.
func (p pendingAnswer) refusalNote() string {
	switch {
	case !p.asked:
		return ""
	case p.unread != "":
		return " It could not be looked for as a pending reference, a record in review on an open pull request's branch: " +
			p.unread + "."
	}
	return " No open pull request's branch on origin carries it either, so it is not a pending reference."
}

// UnlandedReference is a prose citation of a record the tree being cut does not
// hold: an id that resolves nowhere in it, that the baseline does not carry and
// whose line no illustrative marker excuses.
type UnlandedReference struct {
	// File is the citing record, repo-relative; Line is 1-based.
	File string `json:"file"`
	Line int    `json:"line"`
	// ID is the cited record's id.
	ID string `json:"id"`
	// Marked is a citation whose line carries the forward-looking marker; an
	// unmarked one is a pending reference record-lint accepted, or one it
	// refused and the tree carries anyway.
	Marked bool `json:"marked_forward_looking"`
}

// StaleMarker is a forward-looking marker whose every cited record is now in the
// tree: the marker says something no longer true and is reported for removal.
type StaleMarker struct {
	File string   `json:"file"`
	Line int      `json:"line"`
	IDs  []string `json:"ids"`
}

// ReferenceAudit is the release cut's reading of the record's prose citations
// (iss-2611031731385043).
type ReferenceAudit struct {
	// Unlanded is every citation of a record the tree being cut does not hold.
	// Each one refuses the cut.
	Unlanded []UnlandedReference `json:"unlanded,omitempty"`
	// StaleMarkers is every forward-looking marker whose records all arrived,
	// reported for removal and never refused on.
	StaleMarkers []StaleMarker `json:"stale_markers,omitempty"`
}

// AuditReferences reads the record's prose citations for the release cut, under
// the repository's own record-lint configuration: the files, the frontmatter
// lines skipped, the resolver, the baseline and the markers exactly as
// prose_citation_resolves reads them, and no pending lookup. At the cut a record
// in review is not yet in the release, so a citation of it is UNLANDED, whether
// record-lint accepted it as pending or its line carries the forward-looking
// marker: no release ships a citation to a record that never landed. The fix is
// to land that record or correct the reference.
//
// What the cut does not refuse on, and why. An illustrative marker says the
// number is not a citation. A baseline entry is the record's own ruling on one
// id, with its class and reason, and the baseline is the standing backlog that
// drains by its own ratchet: a cut that refused on it would refuse every release
// until a separate ruling was made, which is how a gate gets disabled rather
// than satisfied (the same reasoning as changelog.GuardFindings' standing
// backlog).
//
// It reads the working tree, as the gate does; the ingest's dirty-tree refusal
// is what keeps that the tree being cut. A repository with no record-lint
// configuration, or one that does not arm the rule, gates no citation, and the
// audit is empty.
func AuditReferences(repoRoot string) (ReferenceAudit, error) {
	cfg, err := LoadConfig(filepath.Join(repoRoot, filepath.FromSlash(DefaultRecordLintConfigPath)))
	if err != nil {
		if os.IsNotExist(err) {
			return ReferenceAudit{}, nil
		}
		return ReferenceAudit{}, fmt.Errorf("reading %s: %w", DefaultRecordLintConfigPath, err)
	}
	rc, on := cfg.Rules[ruleProseCitationResolves]
	if !on || !rc.Enabled {
		return ReferenceAudit{}, nil
	}
	in, err := proseInputs(repoRoot, rc)
	if err != nil {
		return ReferenceAudit{}, err
	}
	var audit ReferenceAudit
	for _, abs := range in.files {
		rel, lines, skip, err := readProseRecord(repoRoot, abs)
		if err != nil {
			return ReferenceAudit{}, err
		}
		for _, pl := range readProseLines(lines, skip) {
			resolved := true
			for _, id := range pl.ids {
				if in.resolver.Has(id) {
					continue
				}
				resolved = false
				if _, carried := in.baseline[id]; carried || pl.illustrative {
					continue
				}
				audit.Unlanded = append(audit.Unlanded, UnlandedReference{File: rel, Line: pl.line, ID: id, Marked: pl.forwardLooking})
			}
			if resolved && pl.forwardMarker {
				audit.StaleMarkers = append(audit.StaleMarkers, StaleMarker{File: rel, Line: pl.line, IDs: pl.ids})
			}
		}
	}
	return audit, nil
}

// proseRuleInputs is what one reading of the record's prose needs: the files
// under the stores and extra roots, the resolver over this tree, and the
// baseline, keyed by id.
type proseRuleInputs struct {
	files        []string
	resolver     *recordid.Resolver
	baseline     map[string]ProseBaselineEntry
	baselinePath string
}

// proseInputs gathers them under rc, failing closed as the gate does: no stores
// configured, or stores that hold no record files, would be a reading that
// looked at nothing and reported clean.
func proseInputs(repoRoot string, rc RuleConfig) (proseRuleInputs, error) {
	if len(rc.RecordStores) == 0 {
		return proseRuleInputs{}, &configError{ruleProseCitationResolves +
			": no record_stores configured; the rule would read no prose and report clean"}
	}
	files, err := proseRecordFiles(repoRoot, rc.RecordStores)
	if err != nil {
		return proseRuleInputs{}, err
	}
	if len(files) == 0 {
		return proseRuleInputs{}, &configError{ruleProseCitationResolves +
			": the configured record stores hold no record files; the gate would pass by not looking"}
	}
	extra, err := proseExtraRootFiles(repoRoot, rc.ExtraRoots)
	if err != nil {
		return proseRuleInputs{}, err
	}
	resolver, err := recordid.NewResolver(repoRoot)
	if err != nil {
		return proseRuleInputs{}, &configError{ruleProseCitationResolves + ": " + err.Error()}
	}
	baselinePath := rc.Baseline
	if baselinePath == "" {
		baselinePath = DefaultProseBaselinePath
	}
	baseline, err := loadProseBaseline(repoRoot, baselinePath)
	if err != nil {
		return proseRuleInputs{}, err
	}
	return proseRuleInputs{files: mergeFileLists(files, extra), resolver: resolver, baseline: baseline, baselinePath: baselinePath}, nil
}
