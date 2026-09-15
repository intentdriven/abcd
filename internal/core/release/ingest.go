package release

// ingest.go — the WRITE half of `abcd launch ship`: it takes the host-composed
// changelog prose, proves it describes EXACTLY the cut, and writes the dated
// heading ADR-37's release workflow turns into a tag.
//
// The trust model is the disembark synthesis seam's, with one deliberate
// divergence. The payload guards are SynthesizePrinciples' (size cap,
// DisallowUnknownFields, schema gate, prose sanitisation, marker neutralisation).
// The refusal semantics are ComposePressRelease's: WHOLE-DOCUMENT, never
// cite-or-be-dropped. A dropped principle costs a distilled insight; a dropped
// changelog line is a shipped change missing from the permanent release record —
// a lie by omission in the one document users read to learn what changed. So a
// payload that does not match the cut refuses entirely and writes nothing.
//
// The boundary between the agent and the deterministic core is exact:
//
//	the agent owns  — the WORDING of each line, and which of the WRITABLE
//	                  Keep-a-Changelog sections it belongs in (outcome 3: a
//	                  four-value impact does not decide a section, so the
//	                  section is editorial judgement over record content).
//	the core owns   — the VERSION, the INCLUSION SET, the date, the heading
//	                  shape, the section order, the citation suffix, and the
//	                  WRITABLE SET (writableSections): the composer cannot see
//	                  the previous release, so the sections that make claims
//	                  about it are refused rather than trusted.
//
// The bijection guards ID-COMPLETENESS ONLY. It never second-guesses a section
// within the writable set.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/changelog"
	"github.com/intentdriven/abcd/internal/core/surface"
	"github.com/intentdriven/abcd/internal/core/update"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// changelogFile is the one release record this writes to. It is a constant
// rather than a parameter because ADR-37 names a single CHANGELOG.md at the repo
// root as the release record; a configurable target would let a cut land
// somewhere the tagging workflow never looks.
const changelogFile = "CHANGELOG.md"

// ChangelogSchemaVersion stamps the composed-changelog payload. It is versioned
// independently of every other abcd artifact (mirroring the synthesis schemas) so
// a future breaking change to the shape is detectable rather than silently
// misread by a composer written against the old one.
const ChangelogSchemaVersion = 1

// MaxPayloadBytes caps the untrusted composed-changelog payload, mirroring the
// synthesis cap. It is exported so a front door can bound its READ at the same
// ceiling the core enforces — the front-door bound is a convenience, this one is
// the guarantee.
const MaxPayloadBytes = 1 << 20 // 1 MiB

const (
	// maxChangelogEntries caps the lines one release may carry. A cut with more
	// than this is not a release, it is a corrupt or hostile payload.
	maxChangelogEntries = 500
	// maxRecordsPerEntry caps the records a single line may cite.
	maxRecordsPerEntry = 32
	// maxEntryProseBytes caps one line's prose after sanitisation (mirrors the
	// synthesis prose ceiling).
	maxEntryProseBytes = 4096
	// maxRecordIDBytes bounds a cited id before it is matched or reported. The id
	// grammar admits unbounded digits, so without a cap "itd-" followed by a
	// megabyte of zeros is legal, fails the bijection, and is echoed whole into the
	// operator's terminal. The real ids are a handful of bytes.
	maxRecordIDBytes = 32
)

// Section is a Keep-a-Changelog section name — the agent's editorial judgement
// about a record, which is why it is a payload field and not derived from impact.
type Section string

// The six registered Keep-a-Changelog 1.1.0 sections. The set is CLOSED: an
// unregistered section is a structural refusal, not a new heading, because the
// release record's shape is not the composer's to extend.
const (
	SectionAdded      Section = "Added"
	SectionChanged    Section = "Changed"
	SectionDeprecated Section = "Deprecated"
	SectionRemoved    Section = "Removed"
	SectionFixed      Section = "Fixed"
	SectionSecurity   Section = "Security"
)

// sectionOrder is both the membership set and the rendered order, written once
// so the two can never disagree. The order is Keep a Changelog's own.
var sectionOrder = []Section{
	SectionAdded, SectionChanged, SectionDeprecated, SectionRemoved, SectionFixed, SectionSecurity,
}

// writableSections is the CANONICAL set a composed payload may carry, and the
// one place it is declared: the ingest refusal reads it, the rendered notice is
// tested against it, and the composer prompt's section table is pinned to it.
//
// It is narrower than sectionOrder by ruling (iss-2609011207114761, 2026-09-01):
// the composer's inputs are the records that shipped and their bodies, never the
// base tag's surface, yet Changed, Deprecated and Removed are claims about that
// surface — that something a user could reach in the previous release is now
// different, on notice, or gone. In the v0.7.0 cut the composer wrote three such
// lines about a verb that did not exist in v0.6.9. Security makes the same kind
// of claim about a vulnerability the previous release carried. So until the
// composer can see the previous release the changelog carries only what was
// added and what was fixed: a breaking record, a supersession or a withdrawal is
// an Added line that states the break; a closed vulnerability is a Fixed line.
var writableSections = []Section{SectionAdded, SectionFixed}

// sectionNotice is the sentence every derived section carries directly under
// its heading, once per cut, so the absence of the other sections reads as a
// rule rather than an oversight. It is written per section rather than once in
// the preamble because a reader lands on a release, not on the file.
const sectionNotice = "These notes list what was added and what was fixed; changes to earlier behaviour are not claimed until the composer can see the previous release."

// writableSection is writableSections as a membership test.
var writableSection = func() map[Section]bool {
	m := make(map[Section]bool, len(writableSections))
	for _, s := range writableSections {
		m[s] = true
	}
	return m
}()

// registeredSection is sectionOrder as a membership test.
var registeredSection = func() map[Section]bool {
	m := make(map[Section]bool, len(sectionOrder))
	for _, s := range sectionOrder {
		m[s] = true
	}
	return m
}()

var (
	// payloadRecordIDRe constrains a cited record id. It is deliberately the same
	// grammar internal/core/changelog reads off a record FILENAME: a citation that
	// cannot name a record cannot be part of a bijection over records.
	payloadRecordIDRe = regexp.MustCompile(`^(?:itd|iss)-[0-9]+$`)
	// promptVersionRe validates the composing agent's prompt_version (itd-5), so a
	// release record can be traced to the prompt that worded it.
	promptVersionRe = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
	// unreleasedHeadingRe matches the insertion anchor.
	unreleasedHeadingRe = regexp.MustCompile(`^## \[Unreleased\]\s*$`)
)

// ChangelogEntry is one composed changelog line — the untrusted input shape.
type ChangelogEntry struct {
	// Section is the Keep-a-Changelog section this line belongs in.
	Section Section `json:"section"`
	// Records are the record ids this line reports. A line may report more than
	// one record (a bundle shipped as one user-visible change), and a record may
	// be reported by more than one line; what the bijection requires is that the
	// SET of ids cited across the document equals the cut's required set.
	Records []string `json:"records"`
	// Text is the wording, and the wording only: the citation suffix is appended
	// by the core, so prose carrying its own "(itd-73)" will read it twice.
	Text string `json:"text"`
}

// ChangelogPayload is the whole untrusted host-composed document.
type ChangelogPayload struct {
	// SchemaVersion must equal ChangelogSchemaVersion.
	SchemaVersion int `json:"schema_version"`
	// PromptVersion is the composing agent prompt's semver (itd-5).
	PromptVersion string `json:"prompt_version"`
	// NextTag is the version the composer was given, echoed back. It must equal
	// the version this cut derives: the emit step and the ingest step are two
	// separate reads of a moving repository, and a mismatch means the record set
	// changed underneath the composer — prose written against a stale cut would
	// pass the bijection only by accident.
	NextTag string `json:"next_tag"`
	// Entries are the composed lines, in the order they should appear within
	// their section.
	Entries []ChangelogEntry `json:"entries"`
}

// IncompleteError is the completeness bijection's refusal: the composed document
// and the cut do not describe the same set of records, so NOTHING was written.
//
// The three groups are kept apart because they are three different mistakes with
// three different fixes, and an operator handed one merged list cannot tell which
// they have.
type IncompleteError struct {
	// Missing are required records no line cites — the omission that would make
	// the release record silently untrue.
	Missing []string
	// Invented are cited ids that are not in the cut at all — a line about
	// something that did not ship.
	Invented []string
	// Internal are cited ids that ARE in the cut but declared `impact: internal`.
	// Citing one is an INVENTION, not a tolerated extra: internal is the class
	// that earns no changelog line at all (Impact.InChangelog), so a line about
	// one tells a user that plumbing work changed their world. It is named apart
	// from Invented only because the fix differs — delete the line, or correct the
	// record's impact if the judgement was wrong.
	Internal []string
}

// Error names every id in every group, and what each group means. It is the
// loud-stage: an operator reading it must be able to fix the document without
// re-deriving the cut by hand.
func (e *IncompleteError) Error() string {
	var b strings.Builder
	b.WriteString("the composed changelog does not describe this cut — nothing was written")
	if len(e.Missing) > 0 {
		fmt.Fprintf(&b, "\n  MISSING (shipped, but no line cites it — the release record would lie by omission): %s",
			strings.Join(e.Missing, ", "))
	}
	if len(e.Invented) > 0 {
		fmt.Fprintf(&b, "\n  INVENTED (cited, but not in this cut — a line about something that did not ship): %s",
			strings.Join(e.Invented, ", "))
	}
	if len(e.Internal) > 0 {
		fmt.Fprintf(&b, "\n  INVENTED (cited, but declared `impact: internal` — internal records earn no changelog line): %s",
			strings.Join(e.Internal, ", "))
	}
	b.WriteString("\nre-compose against the emit step's record set: every user-facing record cited, and nothing else")
	return b.String()
}

// IngestResult is the write step's transport-agnostic outcome. It carries the
// whole Cut so a front door can render the same report the emit step renders —
// a refused cut reaches here as a RESULT, not an error, and must still be shown.
type IngestResult struct {
	// Cut is the deterministic cut the prose was validated against.
	Cut Cut `json:"cut"`
	// Written reports whether the release record was updated. It is false both
	// for a refused cut and (with an error) for a refused document; nothing
	// partial is ever written.
	Written bool `json:"written"`
	// Path is the record written, repo-relative; empty when nothing was written.
	Path string `json:"path"`
	// Heading is the dated heading written — the exact line auto-release.yml
	// greps to decide what to tag.
	Heading string `json:"heading"`
	// Lines counts the changelog lines written.
	Lines int `json:"lines"`
	// Cited is the required record-id set, sorted: the bijection's proof, so a
	// reviewer can check the release record against it without re-running a cut.
	Cited []string `json:"cited"`
}

// Ingest validates the host-composed changelog against the deterministic cut and
// writes the dated section into CHANGELOG.md.
//
// current is the caller's view of the command surface, passed in for the reason
// Emit states: internal/core must not walk a cobra tree. at is the clock, passed
// in rather than read, so the date in a durable release heading is an input a
// test can pin instead of a wall-clock read buried in a writer.
//
// The three outcomes are distinguishable on purpose:
//
//	(result, nil) with Written        — the section landed.
//	(result, nil) without Written     — the CUT refuses (result.Cut.Refusals says
//	                                    why). A refusal is a result to render, and
//	                                    the front door maps it to exit 1.
//	(result, error)                   — the DOCUMENT is unusable (a payload fault
//	                                    or a failed bijection). Exit 2; the file
//	                                    is byte-identical to what it was.
//
// Every refusal path returns before the single atomic write, so a half-composed
// changelog can never land.
func Ingest(root string, current surface.Snapshot, raw []byte, at time.Time) (IngestResult, error) {
	cut, err := Emit(root, current)
	if err != nil {
		return IngestResult{}, err
	}
	res := IngestResult{Cut: cut}
	if !cut.Ready {
		return res, nil
	}

	payload, err := decodeChangelogPayload(raw)
	if err != nil {
		return res, err
	}
	if payload.NextTag != cut.NextTag {
		return res, fmt.Errorf("the payload was composed against %q but this cut derives %q — "+
			"the record set moved under the composer; re-run the emit step and compose again",
			termsafe.Sanitize(payload.NextTag), cut.NextTag)
	}

	entries, cited, err := validateEntries(payload.Entries)
	if err != nil {
		return res, err
	}
	required, err := checkBijection(cut, cited)
	if err != nil {
		return res, err
	}

	heading := datedHeading(cut.NextTag, at)
	content, err := insertSection(root, renderSection(heading, entries))
	if err != nil {
		return res, err
	}
	// One atomic replace, preserving the record's existing mode: a crash mid-write
	// must leave the previous CHANGELOG intact, never a truncated one.
	if err := fsutil.WriteFileAtomicPreserveMode(filepath.Join(root, changelogFile), []byte(content)); err != nil {
		return res, err
	}

	res.Written = true
	res.Path = changelogFile
	res.Heading = heading
	res.Lines = len(entries)
	res.Cited = required
	return res, nil
}

// decodeChangelogPayload reads the untrusted document behind the synthesis
// guards: a size cap, an unknown-field refusal (an invented key means the
// composer and this core disagree about the contract), the schema gate, and the
// prompt_version stamp. Every fault here is structural — the document is
// unusable, so nothing is written.
func decodeChangelogPayload(raw []byte) (ChangelogPayload, error) {
	if len(raw) > MaxPayloadBytes {
		return ChangelogPayload{}, fmt.Errorf("changelog payload exceeds the %d-byte cap", MaxPayloadBytes)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var p ChangelogPayload
	if err := dec.Decode(&p); err != nil {
		return ChangelogPayload{}, fmt.Errorf("malformed changelog JSON: %v", err)
	}
	// Decode stops at the first complete value. Anything after it means the host
	// emitted something other than the one document this contract describes, and
	// the posture beside DisallowUnknownFields is uniform: a surprise in the
	// document refuses the document.
	if dec.More() {
		return ChangelogPayload{}, errors.New("the changelog payload carries trailing data after the JSON document")
	}
	switch {
	case p.SchemaVersion == 0:
		return ChangelogPayload{}, errors.New("changelog payload is missing schema_version")
	case p.SchemaVersion > ChangelogSchemaVersion:
		return ChangelogPayload{}, update.TooNew("changelog",
			p.SchemaVersion, ChangelogSchemaVersion)
	case p.SchemaVersion != ChangelogSchemaVersion:
		return ChangelogPayload{}, fmt.Errorf("unsupported changelog schema_version %d", p.SchemaVersion)
	}
	if !promptVersionRe.MatchString(p.PromptVersion) {
		return ChangelogPayload{}, errors.New("changelog payload is missing a semver prompt_version")
	}
	return p, nil
}

// validateEntries checks every line's structure and sanitises its prose,
// returning the cleaned entries and the set of ids they cite.
//
// A fault in ONE entry fails the WHOLE document, which is the divergence from
// cite-or-be-dropped that this seam turns on: dropping a malformed line would
// leave its record uncited, and the bijection would then refuse anyway — with a
// misleading "missing record" instead of the real fault.
func validateEntries(entries []ChangelogEntry) ([]ChangelogEntry, map[string]bool, error) {
	if len(entries) == 0 {
		return nil, nil, errors.New("the changelog payload carries no entries, but the cut has records to report")
	}
	if len(entries) > maxChangelogEntries {
		return nil, nil, fmt.Errorf("too many changelog entries (%d > %d)", len(entries), maxChangelogEntries)
	}

	out := make([]ChangelogEntry, 0, len(entries))
	cited := map[string]bool{}
	for i, in := range entries {
		at := i + 1
		if !registeredSection[in.Section] {
			return nil, nil, fmt.Errorf("entry %d names section %q; a changelog section must be one of %s",
				at, termsafe.Sanitize(string(in.Section)), sectionList())
		}
		if !writableSection[in.Section] {
			// Registered, so the shape is right; refused because the claim is one
			// the composer cannot check. Named apart from the structural refusal so
			// the operator is told the rule, not just the list.
			return nil, nil, fmt.Errorf("entry %d names section %s; the composer cannot see the previous release, "+
				"so a composed changelog carries only %s until it can (iss-2609011207114761) — "+
				"restate the line under one of those and say in the line what changed",
				at, in.Section, strings.Join(sectionNames(writableSections), "|"))
		}
		if len(in.Records) == 0 {
			return nil, nil, fmt.Errorf("entry %d cites no record; every changelog line cites the record it reports", at)
		}
		if len(in.Records) > maxRecordsPerEntry {
			return nil, nil, fmt.Errorf("entry %d cites %d records (max %d)", at, len(in.Records), maxRecordsPerEntry)
		}
		ids := make([]string, 0, len(in.Records))
		seen := map[string]bool{}
		for _, id := range in.Records {
			// Bounded BEFORE the id is matched or quoted, so an over-long one is
			// described rather than echoed.
			if len(id) > maxRecordIDBytes {
				return nil, nil, fmt.Errorf("entry %d cites a %d-byte record id (max %d)", at, len(id), maxRecordIDBytes)
			}
			if !payloadRecordIDRe.MatchString(id) {
				return nil, nil, fmt.Errorf("entry %d cites %q, which is not a record id (want itd-N or iss-N)",
					at, termsafe.Sanitize(id))
			}
			if seen[id] {
				continue
			}
			seen[id] = true
			ids = append(ids, id)
			cited[id] = true
		}
		text := cleanChangelogProse(in.Text)
		if text == "" {
			return nil, nil, fmt.Errorf("entry %d (%s) has empty prose", at, strings.Join(ids, ", "))
		}
		out = append(out, ChangelogEntry{Section: in.Section, Records: ids, Text: text})
	}
	return out, cited, nil
}

// checkBijection is the heart of this phase: the set of cited ids must equal the
// cut's required set — every user-facing record in the cut, and nothing else.
//
// The required set is the cut's records MINUS the internal ones, over BOTH
// directions. The removed side is included deliberately: a record that left a
// terminal folder is a supersession, which is a user-visible change, and letting
// it go uncited would be the same omission as dropping an addition.
//
// It returns the required set, sorted, so the caller can record what was proved.
func checkBijection(cut Cut, cited map[string]bool) ([]string, error) {
	inCut := map[string]bool{}
	requiredSet := map[string]bool{}
	for _, e := range append(append([]Entry{}, cut.Added...), cut.Removed...) {
		inCut[e.ID] = true
		if e.InChangelog {
			requiredSet[e.ID] = true
		}
	}

	var missing, invented, internal []string
	for id := range requiredSet {
		if !cited[id] {
			missing = append(missing, id)
		}
	}
	for id := range cited {
		switch {
		case requiredSet[id]:
		case inCut[id]:
			// In the cut, but not required: by construction the only records the
			// cut carries and does not require are the internal ones.
			internal = append(internal, id)
		default:
			invented = append(invented, id)
		}
	}
	if len(missing)+len(invented)+len(internal) > 0 {
		sort.Strings(missing)
		sort.Strings(invented)
		sort.Strings(internal)
		return nil, &IncompleteError{Missing: missing, Invented: invented, Internal: internal}
	}

	required := make([]string, 0, len(requiredSet))
	for id := range requiredSet {
		required = append(required, id)
	}
	sort.Strings(required)
	return required, nil
}

// datedHeading renders the release heading. Its shape is a CONTRACT with
// .github/workflows/auto-release.yml, which greps
// `^## \[v?[0-9]+\.[0-9]+\.[0-9]+\] - ` for the version it turns into a git tag —
// so this is the one line in the whole programme that must not drift. The version
// is written bare (no `v`), matching the existing record; the workflow accepts
// both. The date is UTC so the same cut reads the same everywhere.
func datedHeading(nextTag string, at time.Time) string {
	return fmt.Sprintf("## [%s] - %s", strings.TrimPrefix(nextTag, "v"), at.UTC().Format("2006-01-02"))
}

// renderSection renders the whole dated section as lines. The section ORDER is
// the core's (Keep a Changelog's own), and entries keep the composer's order
// within their section — so the document is deterministic given the payload.
//
// The notice sits directly under the heading, before any section, once: the
// function renders exactly one cut, and each cut is its own dated section, so
// idempotence across cuts holds by construction rather than by a scan.
func renderSection(heading string, entries []ChangelogEntry) []string {
	lines := []string{heading, "", sectionNotice, ""}
	for _, section := range sectionOrder {
		var body []string
		for _, e := range entries {
			if e.Section != section {
				continue
			}
			// The citation is appended by the core, never trusted from the prose:
			// it is the bijection made visible in the durable record, so a reader
			// can trace every line back to the record it reports.
			body = append(body, fmt.Sprintf("- %s (%s)", e.Text, strings.Join(e.Records, ", ")))
		}
		if len(body) == 0 {
			continue
		}
		lines = append(lines, "### "+string(section), "")
		lines = append(lines, body...)
		lines = append(lines, "")
	}
	return lines
}

// insertSection splices the rendered section into CHANGELOG.md directly beneath
// the `## [Unreleased]` heading and returns the whole new file. It writes
// nothing: the caller performs the single atomic write.
//
// Two preconditions are fail-closed, and both are outcome 7's clean cutover:
//
//   - The `## [Unreleased]` heading must exist. It is the insertion anchor, and
//     guessing where a release goes in a release record is not a thing this does.
//   - Its section must be EMPTY. There is no fold: the maintainer does one final
//     manual roll, and from that cut on every section is derived. Prose sitting
//     under [Unreleased] means the cutover has not happened, and inserting below
//     it would strand hand-written lines above a derived release forever.
//
// Inserting directly beneath [Unreleased] is INTENDED to keep the new heading
// above every older one, which is load-bearing: the tagging workflow greps `-m1`
// and would otherwise re-tag a past release. It is an intention, not a
// guarantee — a CHANGELOG whose anchor sits below an older section satisfies both
// preconditions above — so the finished content is checked for that ordering
// before it is returned, rather than assumed from where the anchor was found.
func insertSection(root string, section []string) (string, error) {
	data, err := fsutil.ReadGuarded(filepath.Join(root, changelogFile), changelog.MaxChangelogBytes)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", changelogFile, err)
	}
	lines := strings.Split(string(data), "\n")

	anchor := -1
	for i, line := range lines {
		if unreleasedHeadingRe.MatchString(strings.TrimRight(line, "\r")) {
			anchor = i
			break
		}
	}
	if anchor < 0 {
		return "", fmt.Errorf("%s has no `## [Unreleased]` heading — that heading is where a derived "+
			"section is inserted, and this writer will not guess where a release belongs", changelogFile)
	}

	end := len(lines)
	for i := anchor + 1; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimRight(lines[i], "\r"), "## ") {
			end = i
			break
		}
	}
	for _, line := range lines[anchor+1 : end] {
		if strings.TrimSpace(line) != "" {
			return "", fmt.Errorf("the `## [Unreleased]` section of %s is not empty — a derived cut never folds "+
				"hand-written prose into a generated section; roll the existing entries into a dated heading "+
				"once, by hand, and every cut after that is fully derived", changelogFile)
		}
	}
	// The composer's prose reached a file whose first dated heading a CI workflow
	// turns into a git tag, so assert the ONE line that matters is the one this
	// built — read back through the same predicate the derivation reads it with.
	if !changelog.IsDatedHeading(section[0]) {
		return "", fmt.Errorf("refusing to write %q: it is not a dated release heading", termsafe.Sanitize(section[0]))
	}

	out := append([]string{}, lines[:anchor+1]...)
	out = append(out, "")
	out = append(out, section...)
	out = append(out, lines[end:]...)
	content := strings.Join(out, "\n")
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	// Shape is not enough: the workflow greps `-m1`, so what it tags is the FIRST
	// dated heading in the finished file. An [Unreleased] anchor sitting below an
	// older section satisfies both preconditions above and still leaves that older,
	// already-tagged heading first — the write would "succeed" and the release
	// would silently never be tagged. Assert POSITION against the bytes about to be
	// written, through the reader's own predicate.
	for _, line := range strings.Split(content, "\n") {
		if !changelog.IsDatedHeading(line) {
			continue
		}
		if line != section[0] {
			return "", fmt.Errorf("refusing to write %s: its newest release heading would still be %q, not the "+
				"derived one — the `## [Unreleased]` anchor does not sit above every dated heading, so the "+
				"tagging workflow would re-tag a past release", changelogFile, termsafe.Sanitize(line))
		}
		break
	}
	return content, nil
}

// cleanChangelogProse sanitises one untrusted composed line through termsafe's
// canonical prose cleaner. The LINE form is load-bearing here in a way the plain
// form is not: this prose lands in a file whose line structure is machine-read, so
// an embedded newline could otherwise forge a heading, a section, or a whole
// release.
func cleanChangelogProse(s string) string {
	return termsafe.CleanProseLine(s, maxEntryProseBytes)
}

// sectionList renders the registered sections for an error message, written from
// sectionOrder so the message can never fall out of step with the enum.
func sectionList() string {
	return strings.Join(sectionNames(sectionOrder), "|")
}

// sectionNames renders a section list as strings, for messages and tests.
func sectionNames(sections []Section) []string {
	names := make([]string, 0, len(sections))
	for _, s := range sections {
		names = append(names, string(s))
	}
	return names
}
