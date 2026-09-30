// Package capture is abcd's transport-agnostic issue-ledger engine: the write
// side of a per-repo issue ledger that replaces the free-form .work/issues.md.
// Every capability is a function taking a structured request and returning a
// structured result; nothing here writes to stdout or knows about a CLI, MCP,
// or prompt surface. The front doors under internal/surface/* marshal these
// results for their transport.
//
// The ledger lives at <repoRoot>/.abcd/work/issues with three
// status directories (open/, resolved/, wontfix/) whose folder membership IS
// the status signal — there is no status: frontmatter field. Each issue is a
// YAML-frontmatter + Markdown-body file named iss-<N>-<slug>.md with an
// unpadded, per-repo id namespace.
//
// This package ports scripts/abcd/_issue_lib.py + issue_workflow.py to Go.
package capture

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/intentdriven/abcd/internal/core/issuerecord"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/record/match"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/core/relink"
)

// LedgerRelPath is the ledger root relative to the repo worktree.
const LedgerRelPath = recordid.IssuesRelDir

// issFamily is the ledger's record family, the argument this package hands
// recordid.SplitRecordFilename. Ledger filenames are split by that ONE shared
// splitter rather than by a regex restated here: the record-lint gate asks the
// same filename ↔ frontmatter question of the committed corpus, and two copies
// of the pattern would let a record pass one side and fail the other — which is
// precisely the split (lint-green, reader-skipped) this sharing closes.
const issFamily = "iss"

// Enumerated field types (validated at the boundary; values mirror
// scripts/abcd/schemas/issue.schema.json).
type (
	// Severity is the capture-time severity guess.
	Severity string
	// Category is the loose issue taxonomy.
	Category string
	// Source is the surfacing channel the issue was discovered through.
	Source string
	// State is a ledger status directory (or "all" for a cross-status scan).
	State string
)

// Severity enum values.
const (
	SeverityNitpick  Severity = "nitpick"
	SeverityMinor    Severity = "minor"
	SeverityMajor    Severity = "major"
	SeverityCritical Severity = "critical"
)

// State enum values.
const (
	StateAll      State = "all"
	StateOpen     State = "open"
	StateResolved State = "resolved"
	StateWontfix  State = "wontfix"
)

// The enum-membership sets are derived from the ONE copy of the value lists in
// core/issueschema, the same lists the record lint reads — so the ledger reader
// and the committed-ledger gate can never disagree about what a legal value is.
var (
	validSeverities = enumSet[Severity](issueschema.Severities)
	validCategories = enumSet[Category](issueschema.Categories)
	validSources    = enumSet[Source](issueschema.Sources)
)

// enumSet builds a membership set of a typed-string enum from its canonical
// string values.
func enumSet[T ~string](vals []string) map[T]bool {
	m := make(map[T]bool, len(vals))
	for _, v := range vals {
		m[T(v)] = true
	}
	return m
}

// ResolvedBy is an optional structured pointer to what resolved an issue.
type ResolvedBy struct {
	Intent string `json:"intent,omitempty"`
	Spec   string `json:"spec,omitempty"`
	Commit string `json:"commit,omitempty"`
}

// Issue is a fully-read ledger entry (frontmatter + provenance + body).
type Issue struct {
	SchemaVersion int      `json:"schema_version"`
	ID            string   `json:"id"`
	Slug          string   `json:"slug"`
	Severity      Severity `json:"severity"`
	Category      Category `json:"category"`
	Source        Source   `json:"source"`
	FoundDuring   string   `json:"found_during"`
	FoundAt       string   `json:"found_at,omitempty"`
	// LapsedAt is the RFC 3339 instant at which a recorded discipline gave way —
	// the lapse itself, never the write-up (spc-60). Required exactly when
	// Category is lapse; optional, and rarely meaningful, for every other.
	LapsedAt string `json:"lapsed_at,omitempty"`
	// Remedy is the proposed fix (itd-82 decision 6): the `remedy:` value, or
	// the older `suggested_fix:` a record carries in its place
	// (issueschema.RemedyOf). A drain takes no issue without one.
	Remedy         string   `json:"remedy,omitempty"`
	RelatedIntents []string `json:"related_intents,omitempty"`
	RelatedSpecs   []string `json:"related_specs,omitempty"`
	RelatedIssues  []string `json:"related_issues,omitempty"`
	BlockedBy      []string `json:"blocked_by,omitempty"` // iss-N dependency edges
	// Duplicates and Refines are the typed links the filing-time match writes
	// (itd-2609212137116617): the iss-N or itd-N this record is a near-identical
	// double of, or a narrower case of. A person confirms a link by leaving it
	// and removes it by deleting the line; a record without either is ordinary.
	Duplicates []string `json:"duplicates,omitempty"`
	Refines    []string `json:"refines,omitempty"`
	// Grounds is the record's recorded conjectures, in the order they were
	// written: one `<token>: <text>` value in the shared core/grounds vocabulary
	// per grounds-bearing act. Appended by promote, resolve and wontfix; never by
	// the create path, because an observation being filed is not yet a conjecture
	// being pursued.
	//
	// It is a LIST because recording is append-only. A record promoted and then
	// resolved carries both conjectures, and the earlier one is precisely what a
	// later reader checks the outcome against (iss-2608301657354776). The values
	// live in the body's `## Grounds` section, not in frontmatter: a frontmatter
	// scalar is set, and setting is what destroyed the first of them.
	Grounds       []string    `json:"grounds,omitempty"`
	Resolution    string      `json:"resolution,omitempty"`
	WontfixReason string      `json:"wontfix_reason,omitempty"`
	ResolvedBy    *ResolvedBy `json:"resolved_by,omitempty"`
	Status        State       `json:"status"` // derived from folder
	Path          string      `json:"path"`   // repo-relative locator (iss-81)
	Body          string      `json:"body"`
	// BlockedByOpen is the derived subset of BlockedBy whose targets are still in
	// open/ (the priority projection populated by List/Status). Not a stored
	// field: an empty slice means the issue is unblocked.
	BlockedByOpen []string `json:"blocked_by_open,omitempty"`
	// Uncommitted is true when git reports the record's file untracked or
	// changed in this checkout (iss-2609100508570527): folder membership is a
	// status signal only once the file is committed, so an uncommitted record is
	// in no state to any other branch, worktree or gate. Derived at read time by
	// Status and List, never stored; false when git cannot answer.
	Uncommitted bool `json:"uncommitted,omitempty"`
}

// CaptureRequest is the input to Capture (append a new issue).
type CaptureRequest struct {
	RepoRoot    string
	IssuesRoot  string
	Text        string // markdown body
	Severity    Severity
	Category    Category
	Source      Source
	Slug        string // caller-supplied; normalised to kebab-case
	FoundDuring string // required, non-empty
	FoundAt     string // optional; "" omits the field
	// LapsedAt is the RFC 3339 instant the discipline gave way. There is no
	// default and none may be invented: the wall clock at write-up is exactly the
	// value the lapse log exists to distinguish itself from (spc-60).
	LapsedAt string
	// Remedy is the proposed fix, written as `remedy:` (itd-82 decision 6).
	// Required: a blank one is refused with ErrRemedyRequired (ruling BX3 of
	// 2026-09-29), and an automatic filer with no fix yet passes
	// issueschema.MachineRemedy (ruling H12), which a drain skips. One line:
	// the serialiser refuses a scalar carrying a line break.
	Remedy         string
	RelatedIntents []string
	RelatedSpecs   []string
	BlockedBy      []string // iss-N dependency edges; each must match ^iss-[0-9]+$
	ForceID        string   // migrator-only; "" = auto-allocate
	// ProductionMode is how the issue's text was produced (itd-178): one of the
	// closed provenance vocabulary, or empty for the vocabulary's default. There
	// is no free-text form. The record's `origin` has no request member at all —
	// it is derived from which command ran, and a capture is researcher-authored
	// by construction.
	ProductionMode string
	// Match, when non-nil, matches the text against every open and resolved
	// issue and every intent before the record is written, under the ledger
	// lock, and writes a `duplicates:` or `refines:` link naming each likely
	// double (itd-2609212137116617). It never refuses the capture: a match that
	// cannot run says why on the result and the record is filed without it.
	// nil files the record unmatched.
	Match *match.Config
	// MatchText, when non-empty, is the text the match compares in place of
	// Text: the finding's own words, for a filer whose record also carries
	// lines every record it files shares (the inbox's provenance, the
	// consistency pass's evidence line). Matched on those, two unrelated
	// records would link each other on the boilerplate alone. Empty compares
	// Text.
	MatchText string
	// MatchExcept names records the match does not compare with: a filer
	// that files several records in one pass passes the ones it has already
	// filed, so two findings of one pass are never linked as doubles of each
	// other.
	MatchExcept []string
}

// CaptureResult is the outcome of a successful Capture. The timestamp-numeric
// mint (adr-45) consults no refs, so the max+1 era's mint_warning degrade note
// no longer exists on this result.
type CaptureResult struct {
	ID     string `json:"id"`
	Slug   string `json:"slug"`
	Path   string `json:"path"`
	Status State  `json:"status"` // always "open"
	// Redacted counts the spans the ledger redactor rewrote on write, and
	// Degraded is non-empty when it ran with a weakened pattern set. Both exist
	// so a surface can SAY the text was altered: redacting in silence would edit
	// a finding's content without telling whoever filed it (loud-staging).
	Redacted int    `json:"redacted,omitempty"`
	Degraded string `json:"redaction_degraded,omitempty"`
	// Uncommitted is true when git reports the record just written as not yet
	// committed — always, in a checkout git answers for, since the file is new.
	// It exists so the write can SAY the record reaches no other branch and no
	// gate until it is committed (iss-2609100508570527).
	Uncommitted bool `json:"uncommitted,omitempty"`
	// NoLocation is true when the capture named no found_at: legitimate for a
	// conceptual finding, and still worth saying, because nothing then ties the
	// record to the repository it is filed into — the shape every misfiled
	// record of iss-2609120511058115 had (iss-2609231156260287).
	NoLocation bool `json:"no_location,omitempty"`
	// Match is the filing-time match's outcome when the request asked for one:
	// the links written, the near misses below the threshold with their scores,
	// or why nothing was compared.
	Match *match.Outcome `json:"match,omitempty"`
}

// ResolveRequest moves an open issue to resolved/.
type ResolveRequest struct {
	RepoRoot   string
	IssuesRoot string
	ID         string
	Resolution string
	// Impact is the product judgement resolved/ requires (issue_impact_valid):
	// one of the shared changelog enum's values (additive|breaking|fix|internal).
	// There is no default — an empty or invalid value is refused, never invented,
	// so a resolved record the tool mints always satisfies its own blocker.
	Impact string
	// ShippedIn optionally names the release that already carried this work, as a
	// tag (v0.6.2). It is a MIGRATION mechanism for the ledger-hygiene case:
	// closing a record for a fix released long ago. A repository abcd manages from
	// its first commit should never need it, because RS001 makes resolution ride
	// the fixing commit and the cut is then right by construction. The derivation reads it and leaves such a record out
	// of the current cut, so the release record cannot announce old work as new
	// (iss-2608241612087533). Absent by default — the ordinary resolution is for
	// work shipping in the release being prepared, and it must never be guessed.
	ShippedIn string
	// ByIntent / BySpec / ByCommit are the optional resolved_by provenance
	// members (spc-25): the intent, spec, or commit that fixed the issue.
	// Ids must exist in their record store (any bucket); the sha is
	// shape-checked only. All optional — absent members are never defaulted.
	ByIntent string // itd-N
	BySpec   string // spc-N
	ByCommit string // 7–64 hex chars (64 covers a SHA-256 repo)
	// Grounds is the REQUIRED conjecture behind the resolution, in the shared
	// `<token>: <text>` grammar (core/grounds). There is no default and none may
	// be invented: a route recorded without its reasoning is the evaporation
	// itd-179 exists to close, and resolve mints the value in the same call, so
	// it has no corpus to fix and refuses from the start.
	Grounds string
	// ProductionMode RESTAMPS the record's production_mode (itd-178). A
	// resolution note is new text with its own mode, so the key is not frozen at
	// mint — but an empty value leaves the existing stamp alone rather than
	// overwriting it with a default, because a transition that declares nothing
	// has made no claim about how the note was produced. `origin` is never
	// rewritten: where a record came from does not change when it is resolved.
	ProductionMode string
}

// WontfixRequest moves an open issue to wontfix/.
type WontfixRequest struct {
	RepoRoot   string
	IssuesRoot string
	ID         string
	Reason     string
	// Grounds optionally overrides the text stamped as `declined: <text>`. A
	// wontfix can never be recorded without grounds — transition already refuses
	// an empty Reason — so what it lacked was the TYPE, not the text, and the
	// reason supplies the default. The override exists because the user-facing
	// reason and the conjecture are not always the same sentence. The token stays
	// declined: a non-action is what that value names.
	Grounds string
	// ProductionMode restamps production_mode on the same terms as
	// ResolveRequest's: declared restamps, absent leaves the stamp alone.
	ProductionMode string
	// Duplicates names the records this one duplicates (iss-N or itd-N), written
	// to the typed `duplicates` link the filing-time match also writes, so a
	// machine reader sees a duplicate closure rather than a wontfix whose
	// relation lives only in the reason's prose (iss-2609291118049254). Each
	// target must exist, in any status folder or intent bucket, and may not be
	// the record itself; a link the record already carries is kept.
	Duplicates []string
}

// TransitionResult is the outcome of a Resolve or Wontfix. ResolvedBy echoes
// the provenance members a Resolve wrote (nil on a flagless resolve and on
// every Wontfix — a non-action points at nothing).
type TransitionResult struct {
	ID         string      `json:"id"`
	Path       string      `json:"path"`
	FromStatus State       `json:"from_status"`
	ToStatus   State       `json:"to_status"`
	ResolvedBy *ResolvedBy `json:"resolved_by,omitempty"`
	// Redacted / Degraded mirror CaptureResult: a resolution or wontfix note is
	// free text written to the same committed ledger, so it goes through the
	// same redactor and reports the same way.
	Redacted int    `json:"redacted,omitempty"`
	Degraded string `json:"redaction_degraded,omitempty"`
	// Relinked lists every relative markdown link the transition repointed
	// because it named the issue's old path in open/ (iss-2609250846525896).
	Relinked []relink.Rewrite `json:"relinked,omitempty"`
	// RelinkError is a NON-FATAL report of a repoint that failed part-way: the
	// issue has moved and the transition stands, so the surface prints it
	// loudly, and record-lint's links_resolve names any link left behind.
	RelinkError string `json:"relink_error,omitempty"`
}

// ListRequest queries one state (or "all").
type ListRequest struct {
	RepoRoot   string
	IssuesRoot string
	State      State // "" is treated as "all"
}

// SkipRecord surfaces a corrupt/invalid ledger file without failing the scan.
//
// Layer names WHICH reader stage refused the file (iss-2609120452071388). A
// skip reported as a bare error left the reader unable to tell whether the
// writer or the validator was the side that was wrong: a name the grammar
// refuses, a leaf the guarded read refuses, and a value the schema refuses are
// three different defects with three different remedies.
type SkipRecord struct {
	Path  string    `json:"path"`
	Layer SkipLayer `json:"layer"`
	Error string    `json:"error"`
}

// SkipLayer is the reader stage that refused a ledger file, in scan order. It
// is the record reader's own type (core/issuerecord), so the board and the
// committed-ledger gate name a refusal with one vocabulary.
type SkipLayer = issuerecord.Layer

// The reader's stages, in the order a file meets them; see core/issuerecord.
const (
	SkipLayerName        = issuerecord.LayerName
	SkipLayerRead        = issuerecord.LayerRead
	SkipLayerFrontmatter = issuerecord.LayerFrontmatter
	SkipLayerSchema      = issuerecord.LayerSchema
	SkipLayerInvariant   = issuerecord.LayerInvariant
)

// ListResult is Issues sorted ascending by numeric N plus a corrupt roster.
type ListResult struct {
	Issues  []Issue      `json:"issues"`
	Skipped []SkipRecord `json:"skipped"`
}

// StatusRequest is the input to the read-only status render.
type StatusRequest struct {
	RepoRoot   string
	IssuesRoot string
}

// StatusResult is the bare-invocation status snapshot (guaranteed no mutation).
type StatusResult struct {
	OpenCount     int `json:"open_count"`
	ResolvedCount int `json:"resolved_count"`
	WontfixCount  int `json:"wontfix_count"`
	// SkippedCount is the number of files that claim to be records and that
	// none of the three totals counts, because the reader refused them. It is
	// len(Skipped), carried as a count beside the others so the board states
	// what it excluded next to what it counted (iss-2609120452071388).
	SkippedCount int `json:"skipped_count"`
	// UncommittedCount is the number of readable records across the three
	// folders that git reports untracked or changed (iss-2609100508570527).
	UncommittedCount int          `json:"uncommitted_count"`
	RecentOpen       []Issue      `json:"recent_open"` // up to 10, newest first
	Skipped          []SkipRecord `json:"skipped"`
}

// FieldValueError is a capture request member outside its closed vocabulary.
// It names the member, the value and the accepted set, and nothing about
// frontmatter: the value came from the caller's request, not from a record,
// so the refusal speaks about the request (iss-2608290810037524). A front door
// maps Field to its own spelling of the input — the CLI's flag of the same
// name.
type FieldValueError struct {
	Field    string
	Value    string
	Accepted []string
}

func (e *FieldValueError) Error() string {
	return fmt.Sprintf("capture: %s %q is not accepted; %s (nothing written)", e.Field, e.Value, acceptedValues(e.Accepted))
}

// Sentinel errors the surface maps to exit codes and messages. Core never
// prints them.
var (
	// ErrUnknownIssueID means the id was absent from all three dirs.
	ErrUnknownIssueID = errors.New("unknown issue id")
	// ErrTransitionConflict means the id was found but not in open/ (already
	// resolved/wontfixed), or a concurrent move consumed it.
	ErrTransitionConflict = errors.New("transition conflict")
	// ErrDuplicateIssueID means a ForceID (or on-disk state) collided.
	ErrDuplicateIssueID = errors.New("duplicate issue id")
	// ErrAllocatorContention means the lock timed out or the O_EXCL retry
	// budget was exhausted.
	ErrAllocatorContention = errors.New("allocator contention")
	// ErrChecksumMismatch means a concurrent edit occurred during a transition.
	ErrChecksumMismatch = errors.New("checksum mismatch")
	// ErrAlreadyPromoted means the issue already names, and is named by, an
	// intent other than the one this call is joining — refused rather than
	// promoted twice.
	ErrAlreadyPromoted = errors.New("already promoted")
	// ErrGroundsRefused means the triage's grounds argument was absent, outside
	// the closed vocabulary, malformed, or below the substance floor. It is one
	// sentinel for every one of those because they are one thing to a caller —
	// the argument was not usable and nothing was written — and a surface that
	// distinguished them by exit code would teach a script that a misspelled
	// token is a different KIND of failure from a missing one
	// (iss-2608300930057882).
	ErrGroundsRefused = errors.New("grounds refused")
	// ErrRequestRefused means a member of a transition request is outside its
	// shape or names nothing: an impact outside its set, a provenance reference
	// that is malformed or absent, a release tag or production mode that is not
	// one. Like ErrGroundsRefused it is the caller's input to fix with nothing
	// written, and the surface gives both the one exit code a refusal takes
	// (iss-2609260552251398).
	ErrRequestRefused = errors.New("request refused")
	// ErrRemedyRequired means a new issue was filed without a remedy: every new
	// issue carries the fix it proposes (ruling BX3 of 2026-09-29, itd-82
	// decision 6), and an automatic filer with none writes
	// issueschema.MachineRemedy. It is a request refusal, with nothing written.
	ErrRemedyRequired = errors.New("remedy required")
	// The record reader's own sentinels (core/issuerecord), re-exported: one
	// value under either name, so errors.Is holds across the two packages.
	//
	// ErrInvariantViolation means frontmatter passed the schema but violates a
	// folder-status cross-field invariant.
	ErrInvariantViolation = issuerecord.ErrInvariantViolation
	// ErrMalformedFrontmatter means frontmatter could not be parsed or failed
	// schema validation.
	ErrMalformedFrontmatter = issuerecord.ErrMalformedFrontmatter
	// ErrMissingRequiredField means a schema-required field was absent.
	ErrMissingRequiredField = issuerecord.ErrMissingRequiredField
	// ErrPathUnsafe means the ledger root or a status dir is a symlink.
	ErrPathUnsafe = issuerecord.ErrPathUnsafe
	// ErrNotCharacterised means a disposition or an admission was asked for at
	// the widening position before a committed comparative run named the item's
	// run. The design characterises first and admits second, and the refusal
	// names the run it is waiting on (spc-2609020626040342).
	ErrNotCharacterised = errors.New("not yet characterised")
)

// refusedError marks an error as a refusal of the request's own input, keeping
// its text: it unwraps to ErrRequestRefused and to the error it wraps.
type refusedError struct{ err error }

func (e refusedError) Error() string   { return e.err.Error() }
func (e refusedError) Unwrap() []error { return []error{ErrRequestRefused, e.err} }

// refused marks err as a refusal of the request's input; nil stays nil.
func refused(err error) error {
	if err == nil {
		return nil
	}
	return refusedError{err: err}
}

// Field regexes mirroring issue.schema.json.
var (
	reIssID     = issuerecord.IssIDRe
	reItdID     = issuerecord.ItdIDRe
	reSpcID     = issuerecord.SpcIDRe
	reCommitSha = regexp.MustCompile(`^[0-9a-f]{7,64}$`)
	reSlug      = issueschema.SlugRe // the ONE kebab-slug pattern, shared with record-lint
	// issFileNumRe is the ONE grammar that decides whether a ledger filename NAMES
	// a record — the same recordid.FilenameNumRe the read-side resolver and
	// record-lint's per-store rule match, so capture, the resolver and the gate
	// agree on which files are records rather than sitting on two detection
	// grammars (iss-2608280739112123). It is deliberately DISTINCT from the
	// filename<->frontmatter slug agreement, which stays on the stricter
	// recordid.SplitRecordFilename (core/issuerecord) because that check EXTRACTS and
	// compares the slug; detection only needs the ordinal.
	issFileNumRe = recordid.FilenameNumRe(issFamily)
	reAbcdListID = regexp.MustCompile(`^(itd|fn|iss|rdi)-[0-9]+$`)
	reSortIssID  = regexp.MustCompile(`^iss-([0-9]+)(-|$|\.)`)
	reScalarKey  = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	// statusDirs is the ledger's status list projected into State, and
	// statusDirName its inverse. Both are DERIVED from issueschema.StatusDirs —
	// the one canonical list the allocator provisions, the readers scan and the
	// deterministic gates scope to — rather than restated here, so the State
	// projection and the directory names cannot disagree about what a status is.
	statusDirs    = stateProjection()
	statusDirName = dirNameProjection()
)

// stateProjection renders the canonical status list as States, in the same order.
func stateProjection() []State {
	out := make([]State, 0, len(issueschema.StatusDirs))
	for _, d := range issueschema.StatusDirs {
		out = append(out, State(d))
	}
	return out
}

// dirNameProjection is stateProjection's inverse: a State back to the directory
// name it names. A State and its directory are the same string by construction,
// which is the point — the two spellings cannot drift because there is only one.
func dirNameProjection() map[State]string {
	out := make(map[State]string, len(issueschema.StatusDirs))
	for _, d := range issueschema.StatusDirs {
		out[State(d)] = d
	}
	return out
}
