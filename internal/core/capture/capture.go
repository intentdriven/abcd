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
	"regexp"

	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/recordid"
)

// LedgerRelPath is the ledger root relative to the repo worktree.
const LedgerRelPath = ".abcd/work/issues"

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
	LapsedAt       string   `json:"lapsed_at,omitempty"`
	RelatedIntents []string `json:"related_intents,omitempty"`
	RelatedSpecs   []string `json:"related_specs,omitempty"`
	RelatedIssues  []string `json:"related_issues,omitempty"`
	BlockedBy      []string `json:"blocked_by,omitempty"` // iss-N dependency edges
	PromotedTo     string   `json:"promoted_to,omitempty"`
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
	LapsedAt       string
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
}

// ListRequest queries one state (or "all").
type ListRequest struct {
	RepoRoot   string
	IssuesRoot string
	State      State // "" is treated as "all"
}

// SkipRecord surfaces a corrupt/invalid ledger file without failing the scan.
type SkipRecord struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

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
	OpenCount     int          `json:"open_count"`
	ResolvedCount int          `json:"resolved_count"`
	WontfixCount  int          `json:"wontfix_count"`
	RecentOpen    []Issue      `json:"recent_open"` // up to 10, newest first
	Skipped       []SkipRecord `json:"skipped"`
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
	// ErrGroundsRefused means the triage's grounds argument was absent, outside
	// the closed vocabulary, malformed, or below the substance floor. It is one
	// sentinel for every one of those because they are one thing to a caller —
	// the argument was not usable and nothing was written — and a surface that
	// distinguished them by exit code would teach a script that a misspelled
	// token is a different KIND of failure from a missing one
	// (iss-2608300930057882).
	ErrGroundsRefused = errors.New("grounds refused")
	// ErrInvariantViolation means frontmatter passed the schema but violates a
	// folder-status cross-field invariant.
	ErrInvariantViolation = errors.New("invariant violation")
	// ErrMalformedFrontmatter means frontmatter could not be parsed or failed
	// schema validation.
	ErrMalformedFrontmatter = errors.New("malformed frontmatter")
	// ErrMissingRequiredField means a schema-required field was absent.
	ErrMissingRequiredField = errors.New("missing required field")
	// ErrPathUnsafe means the ledger root or a status dir is a symlink.
	ErrPathUnsafe = errors.New("path unsafe")
)

// Field regexes mirroring issue.schema.json.
var (
	reIssID     = regexp.MustCompile(`^iss-[0-9]+$`)
	reItdID     = regexp.MustCompile(`^itd-[0-9]+$`)
	reSpcID     = regexp.MustCompile(`^spc-[0-9]+$`)
	reCommitSha = regexp.MustCompile(`^[0-9a-f]{7,64}$`)
	reSlug      = issueschema.SlugRe // the ONE kebab-slug pattern, shared with record-lint
	// reIssNameClaim matches a filename that CLAIMS to be a ledger record: the
	// family prefix followed by an ordinal. It is deliberately LOOSER than
	// recordid.SplitRecordFilename, and the gap between the two is the point — a
	// name that claims to be a record and then is not well-formed is reported as a
	// skip rather than dropped, while a file claiming nothing (README.md,
	// notes.md, iss-notes.md, the allocator lock) stays silently ignored. The
	// ordinal is what parts the two: prose that merely starts with the prefix is
	// not asserting an id.
	reIssNameClaim = regexp.MustCompile(`^iss-[0-9]`)
	// issFileNumRe is the ONE grammar that decides whether a ledger filename NAMES
	// a record — the same recordid.FilenameNumRe the read-side resolver and
	// record-lint's per-store rule match, so capture, the resolver and the gate
	// agree on which files are records rather than sitting on two detection
	// grammars (iss-2608280739112123). It is deliberately DISTINCT from the
	// filename<->frontmatter slug agreement, which stays on the stricter
	// recordid.SplitRecordFilename (validate.go) because that check EXTRACTS and
	// compares the slug; detection only needs the ordinal.
	issFileNumRe = recordid.FilenameNumRe(issFamily)
	reAbcdListID = regexp.MustCompile(`^(itd|fn|iss)-[0-9]+$`)
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
