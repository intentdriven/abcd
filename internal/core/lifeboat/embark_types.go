package lifeboat

// This file is the SHARED TYPE + DATA surface for M5 (embark and the round-trip,
// itd-88). It carries only types, constants, the inverse-mapping table, and the
// closure/exclusion sets — no logic. Three implementation agents build against
// it without diverging:
//
//   - Agent A (plan.go): teaches the packer to copy .abcd/development/specs/**
//     into rescue/specs/<bucket>/<leaf>, adds recordManifestSHA256 over the
//     record-derived families (isRecordDerived below), and records it in
//     Provenance as record_manifest_sha256.
//   - Agent B (embark.go): EmbarkProbe / EmbarkFrom / verifyManifest and the
//     conflict/marker/coverage machinery, plus ahoy.EnsureMarker in package ahoy.
//   - Agent C (surface/cli): the `abcd embark probe|from` command tree,
//     commands/embark.md, and the surface-registry row-3 flip.
//
// See adr-35 and .abcd/development/plans/2026-07-14-lifeboat-coverage-experiment.md
// (the "M5 — embark and the round-trip" section) for the ratified contract.

import (
	"errors"

	corereflect "github.com/intentdriven/abcd/internal/core/reflect"
)

// EmbarkSchemaVersion stamps EmbarkPlan and EmbarkResult so a future breaking
// change to their shape is detectable rather than silently misread.
const EmbarkSchemaVersion = 1

// nativeSpecsDir is the spec store under the abcd tree, named once here so the
// packer (Agent A, rescue/specs → this) and the embarker (Agent B, this →
// rescue/specs) share one definition and cannot drift. It mirrors
// spec.SpecsRelDir; a drift test (Agent A) asserts the two agree.
const nativeSpecsDir = ".abcd/development/specs"

// Embark safety ceilings. The lifeboat dir is UNTRUSTED input (decision 6), so
// every read is bounded: a hostile or corrupt lifeboat cannot exhaust memory or
// smuggle an oversize file past manifest verification. The per-file cap is above
// the 4 MiB pack-time probe cap so a legitimately packed record always fits.
const (
	maxEmbarkFiles      = maxPlanFiles      // 20000, reusing the pack ceiling
	maxEmbarkFileBytes  = 16 << 20          // 16 MiB per lifeboat file read/hashed
	maxEmbarkTotalBytes = maxPlanTotalBytes // 512 MiB across the whole lifeboat
)

// maxEmbarkDepth caps how many directory levels below the lifeboat root the
// walk descends. Like probe.go's maxWalkDepth it prunes the pathological deep
// chain — a directory-only lifeboat `a/a/a/…` holds no regular file, so the
// file cap never fires and an unbounded recursion walks it at O(depth) cost
// (GH #337). A packed lifeboat mirrors a shallow record tree, so the ceiling is
// generous headroom that only a hostile chain reaches; unlike the probe, embark
// REFUSES the lifeboat on reaching it rather than truncating, because manifest
// verification needs the complete file list and a partial one is a mismatch.
const maxEmbarkDepth = 64

// ErrEmbarkConflicts is the sentinel EmbarkFrom returns when the target already
// holds files that differ from what the lifeboat would write. On this error the
// write path has done NOTHING (decision 4): the returned EmbarkResult carries the
// full Conflicts slice for the surface to render as one bulk report, and no file
// was touched. The surface maps errors.Is(err, ErrEmbarkConflicts) to exit 1
// (an expected refusal), distinct from a structural fault (exit 2).
var ErrEmbarkConflicts = errors.New("embark: target has conflicting files; nothing was written")

// ---------------------------------------------------------------------------
// Conflict model (decision 4). A conflict is per-FILE; a target that merely
// carries unrelated files is not a conflict. Byte-identical content is not a
// conflict either — it is an idempotent skip (ActionUnchanged). On ANY conflict
// EmbarkFrom refuses and writes nothing.
// ---------------------------------------------------------------------------

// ConflictKind classifies why embark cannot safely write one planned target.
type ConflictKind string

const (
	// ConflictExistsDiffers: the target path is a regular file whose bytes differ
	// from what the lifeboat would write. Identical bytes are NOT a conflict.
	ConflictExistsDiffers ConflictKind = "exists-differs"
	// ConflictTargetNotRegular: the target path exists but is not a regular file
	// (a directory, symlink, or device). Embark refuses to replace it.
	ConflictTargetNotRegular ConflictKind = "target-not-regular"
	// ConflictDuplicateTarget: two lifeboat files resolve to the same embark
	// target (e.g. a bucket-less rescue/intents/<leaf> alongside
	// rescue/intents/drafts/<leaf>). Writing both would be a silent
	// last-writer-wins overwrite, so the collision refuses instead.
	ConflictDuplicateTarget ConflictKind = "duplicate-target"

	// ConflictParentNotDir: a parent component of the target path exists and is
	// not a real directory (a file, or a symlink). Writing the file would require
	// clobbering or traversing it, so embark refuses.
	ConflictParentNotDir ConflictKind = "parent-not-dir"
)

// Conflict is one target path embark cannot safely write. Path is the target
// repo-relative POSIX path; LifeboatPath is the source file in the lifeboat it
// was mapped from. Detail is a short, sanitised human explanation.
type Conflict struct {
	Path         string       `json:"path"`
	LifeboatPath string       `json:"lifeboat_path"`
	Kind         ConflictKind `json:"kind"`
	Detail       string       `json:"detail,omitempty"`
}

// ---------------------------------------------------------------------------
// Planned writes.
// ---------------------------------------------------------------------------

// EmbarkAction is the disposition of a mapped, non-conflicting file against the
// current target. A conflicting file never appears in Planned — it is recorded
// in Conflicts instead.
type EmbarkAction string

const (
	// ActionCreate: the target path is absent; embark would write it.
	ActionCreate EmbarkAction = "create"
	// ActionUnchanged: the target already holds byte-identical content; embark
	// skips it (idempotent re-embark, and the P2 self-closure property).
	ActionUnchanged EmbarkAction = "unchanged"
)

// PlannedEmbark is one lifeboat file embark maps to a target write. Content is
// carried for the write path but omitted from JSON (mirrors PlannedFile), so a
// probe render reports size, not bytes.
type PlannedEmbark struct {
	LifeboatPath string       `json:"lifeboat_path"`
	TargetPath   string       `json:"target_path"`
	Family       string       `json:"family"`
	Bytes        int          `json:"bytes"`
	Action       EmbarkAction `json:"action"`
	Content      []byte       `json:"-"`
}

// ---------------------------------------------------------------------------
// Files embark does NOT write. Only files under the known families (embarkFamilies)
// are embarked; everything else informs the report and is never written (decision
// 7). A file is Ignored for one of three reasons.
// ---------------------------------------------------------------------------

// IgnoredReason says why a lifeboat file is not embarked.
type IgnoredReason string

const (
	// IgnoredReportOnly: an identity/git-derived or metadata file that informs the
	// report but is never written into a target (brief/**, coverage.*,
	// graveyard/**, rescue/spine.md, _provenance.json).
	IgnoredReportOnly IgnoredReason = "report-only"
	// IgnoredUnmapped: a file under a known family root that does not resolve to a
	// target (unknown bucket, unsafe leaf, or bucket-less where no default exists).
	IgnoredUnmapped IgnoredReason = "unmapped"
	// IgnoredUnknown: a file under no known family — a foreign or tampered entry.
	// It passed manifest verification (it is part of the sealed set) but embark
	// has no home for it, so it is reported and never written.
	IgnoredUnknown IgnoredReason = "unknown"
)

// IgnoredFile is one lifeboat file embark did not write, and why.
type IgnoredFile struct {
	LifeboatPath string        `json:"lifeboat_path"`
	Reason       IgnoredReason `json:"reason"`
	Detail       string        `json:"detail,omitempty"`
}

// ---------------------------------------------------------------------------
// Conventions-file marker (decision 5). Embark NEVER copies lifeboat prose into
// the target's conventions file; it re-injects the CURRENT abcd marker block via
// ahoy.EnsureMarker into the file the target's setup chose: AGENTS.md, or none.
// ---------------------------------------------------------------------------

// MarkerAction is what embark would do (probe) or did (from) to the marker block
// in the target's conventions file.
type MarkerAction string

const (
	// MarkerActionInstall: no block present; embark injects the current block.
	MarkerActionInstall MarkerAction = "install"
	// MarkerActionRefresh: an outdated/duplicated block present; embark replaces it.
	MarkerActionRefresh MarkerAction = "refresh"
	// MarkerActionCurrent: the current block is already present; no write.
	MarkerActionCurrent MarkerAction = "current"
	// MarkerActionSkip: the target's setup chose no conventions file, or the
	// chosen file is a symlink or otherwise unwritable; embark reports why and
	// writes no block (non-fatal to the record writes).
	MarkerActionSkip MarkerAction = "skip"
)

// MarkerResult reports the marker disposition for the target's conventions file.
// Target names the file chosen ("AGENTS.md"), and is empty when none was chosen,
// which always comes with MarkerActionSkip and a Note saying why.
type MarkerResult struct {
	Target  string       `json:"target"`  // "AGENTS.md", or "" when no file was chosen
	Action  MarkerAction `json:"action"`  // probe: predicted; from: performed
	Changed bool         `json:"changed"` // probe: would change; from: did change
	Note    string       `json:"note,omitempty"`
}

// ---------------------------------------------------------------------------
// Coverage handoff (decision 7). Embark surfaces the coverage BLANKS and their
// questions FIRST — before the write summary — because that is what a lifeboat
// hands a product thinker. Read from the lifeboat's coverage.json; an absent or
// unparseable file degrades to a note, never a fatal error.
// ---------------------------------------------------------------------------

// BlankPrompt is one unanswered brief section a human must supply.
type BlankPrompt struct {
	Section  Section  `json:"section"`
	Kind     Kind     `json:"kind"`
	Question string   `json:"question,omitempty"`
	Searched []string `json:"searched,omitempty"`
}

// CoverageHandoff is the blanks-first payload embark surfaces before the write
// summary. Present is false when the lifeboat carried no coverage.json; Degraded
// is true when it was present but unreadable/unparseable (Note explains).
type CoverageHandoff struct {
	Present  bool          `json:"present"`
	Degraded bool          `json:"degraded,omitempty"`
	Note     string        `json:"note,omitempty"`
	Summary  Summary       `json:"summary"`
	Blanks   []BlankPrompt `json:"blanks,omitempty"`
	// PassBExemption is the lifeboat's own declaration that the transcript pass
	// did not run for it, carried through from _provenance.json. It is part of
	// the handoff because it is the difference between a gap a human has to close
	// and a pass that was never there: nil is a lifeboat that declares nothing,
	// and reads exactly as it did before the field existed.
	PassBExemption *PassBExemption `json:"pass_b_exemption,omitempty"`
}

// ---------------------------------------------------------------------------
// The two top-level results. EmbarkPlan is the read-only `embark probe` output;
// EmbarkResult is the post-write `embark from` summary. Both share Conflict,
// IgnoredFile, MarkerResult, and CoverageHandoff.
// ---------------------------------------------------------------------------

// EmbarkPlan is the read-only result of `embark probe <lifeboat> [target]`: what
// would land where, what conflicts would block a write, which lifeboat files are
// not embarked, the marker action, and the coverage handoff. It writes nothing.
type EmbarkPlan struct {
	SchemaVersion        int              `json:"schema_version"`
	LifeboatDir          string           `json:"lifeboat_dir"`
	TargetDir            string           `json:"target_dir"`
	SourceName           string           `json:"source_name"`
	ManifestVerified     bool             `json:"manifest_verified"`
	ManifestSHA256       string           `json:"manifest_sha256"`
	RecordManifestSHA256 string           `json:"record_manifest_sha256,omitempty"`
	Coverage             *CoverageHandoff `json:"coverage,omitempty"`
	Planned              []PlannedEmbark  `json:"planned,omitempty"`
	Conflicts            []Conflict       `json:"conflicts,omitempty"`
	Ignored              []IgnoredFile    `json:"ignored,omitempty"`
	Marker               MarkerResult     `json:"marker"`
}

// EmbarkResult is the outcome of `embark from <lifeboat> [target]`. On success
// Conflicts is empty and Written/Unchanged account for every mapped file. On the
// ErrEmbarkConflicts refusal path Conflicts is populated, Written is zero, and no
// file was touched.
type EmbarkResult struct {
	SchemaVersion int              `json:"schema_version"`
	LifeboatDir   string           `json:"lifeboat_dir"`
	TargetDir     string           `json:"target_dir"`
	SourceName    string           `json:"source_name"`
	Written       int              `json:"written"`
	Unchanged     int              `json:"unchanged"`
	BytesWritten  int              `json:"bytes_written"`
	Families      map[string]int   `json:"families,omitempty"` // family name -> files written
	Marker        MarkerResult     `json:"marker"`
	Coverage      *CoverageHandoff `json:"coverage,omitempty"`
	Ignored       []IgnoredFile    `json:"ignored,omitempty"`
	Conflicts     []Conflict       `json:"conflicts,omitempty"`
}

// ---------------------------------------------------------------------------
// The inverse mapping table (decision 7): the SINGLE SOURCE OF TRUTH for which
// lifeboat files embark writes back into a repo, and where. Only files under
// these families are embarked; everything else informs the report and is never
// written.
//
// A flat family (Buckets == nil) maps <LifeboatPrefix><leaf> ->
// <TargetPrefix><leaf>. A bucketed family maps <LifeboatPrefix><bucket>/<leaf>
// -> <TargetPrefix><bucket>/<leaf> for bucket in Buckets; a file directly under
// LifeboatPrefix with no bucket maps to DefaultBucket when it is non-empty and is
// otherwise Unmapped (reported, never written). Every bucket must be an EXACT
// member of Buckets and every leaf must pass safeLeaf, so a hostile lifeboat
// filename can never steer a write outside the target family.
//
// The intent family alone carries a DefaultBucket ("drafts"): a bucket-less
// rescue/intents/<f> lands in drafts/, the entry lifecycle state, because a
// bucket-less intent has an unknown lifecycle and drafts/ is the only bucket that
// fabricates no commitment (planned/shipped/superseded would assert a state the
// lifeboat never recorded). Issues and specs have NO default: their bucket is a
// load-bearing status (open/resolved/wontfix, open/closed) that must not be
// invented, so a bucket-less issue/spec is Unmapped.
type embarkFamily struct {
	Name           string
	LifeboatPrefix string   // POSIX, trailing slash
	TargetPrefix   string   // POSIX, trailing slash
	Buckets        []string // nil => flat family, unless BucketValid is set
	DefaultBucket  string   // used for a bucket-less file; "" => such a file is Unmapped
	// BucketValid, when set, admits a bucket by its shape instead of by
	// membership of Buckets: the retrospective family's buckets are release
	// tags, an open set, so each is held to the tag shape rather than listed.
	BucketValid func(string) bool
	// Leaf, when set, is the one leaf name the family admits in a bucket.
	Leaf string
}

// intentEmbarkBuckets mirrors intent.Buckets; specEmbarkBuckets mirrors the spec
// store's {open,closed}. Issue buckets reuse the package-local nativeIssueStates
// ([open resolved wontfix]). Kept as local literals so embark_types.go pulls in
// no store packages; drift tests (Agent B) assert each equals its canonical list.
var (
	intentEmbarkBuckets = []string{"drafts", "planned", "shipped", "disciplines", "superseded"}
	specEmbarkBuckets   = []string{"open", "closed"}
)

// embarkFamilies is the inverse mapping. TargetPrefix values reuse the
// package-local record-location constants (nativeADRDir, nativeIssuesDir,
// nativeIntentsDir) plus nativeSpecsDir so the packer and the embarker share one
// source of truth. NOTE the embark SET (these four families) is a SUBSET of the
// closure SET (recordDerivedPrefixes): graveyard/abandoned.json is record-derived
// and part of the P1 closure, but it is NOT embarked — a fresh target receives no
// graveyard/; abandoned.json informs the report only and re-derives on re-pack
// from the embarked ADRs/intents/issues.
var embarkFamilies = []embarkFamily{
	{Name: "adrs", LifeboatPrefix: "docs/adrs/", TargetPrefix: nativeADRDir + "/", Buckets: nil, DefaultBucket: ""},
	{Name: "issues", LifeboatPrefix: "activity/issues/", TargetPrefix: nativeIssuesDir + "/", Buckets: nativeIssueStates, DefaultBucket: ""},
	{Name: "intents", LifeboatPrefix: "rescue/intents/", TargetPrefix: nativeIntentsDir + "/", Buckets: intentEmbarkBuckets, DefaultBucket: "drafts"},
	{Name: "specs", LifeboatPrefix: "rescue/specs/", TargetPrefix: nativeSpecsDir + "/", Buckets: specEmbarkBuckets, DefaultBucket: ""},
	// The release retrospectives (itd-24): retrospectives/<release-tag>/README.md,
	// one per release, the tag held to the release-tag shape and the leaf to
	// README.md, so a hostile name can steer no write out of the store.
	{Name: "retrospectives", LifeboatPrefix: retrospectivesLifeboatDir + "/", TargetPrefix: corereflect.RetrospectivesRelDir + "/",
		BucketValid: corereflect.IsReleaseTag, Leaf: retrospectiveLeaf},
}

// retrospectivesLifeboatDir is where a lifeboat carries the retrospectives, and
// retrospectiveLeaf the one file each release's directory holds.
const (
	retrospectivesLifeboatDir = "retrospectives"
	retrospectiveLeaf         = "README.md"
)

// ---------------------------------------------------------------------------
// Closure and exclusion sets.
// ---------------------------------------------------------------------------

// recordDerivedPrefixes are the lifeboat path prefixes whose bytes derive purely
// from the repo's RECORD (never from git or the operator's identity), so they
// must round-trip byte-identically through pack -> embark -> re-pack (closure
// property P1, decision 1). recordManifestSHA256 (Agent A) hashes exactly the
// files matching one of these prefixes. A slash-terminated entry matches a whole
// family; "graveyard/abandoned.json" is deliberately slash-LESS so it matches only
// itself (the deterministic layer-2 record extraction), not a family.
//
// Excluded BY DESIGN (identity/git-derived, so they legitimately differ across a
// re-pack): coverage.* (probe-derived), brief/** (probe-derived), rescue/spine.md
// (git-derived when no intent corpus exists), graveyard/archaeology.json
// (git-derived), and _provenance.json (carries the hash; cannot hash itself).
var recordDerivedPrefixes = []string{
	"docs/adrs/",
	"activity/issues/",
	"rescue/intents/",
	"rescue/specs/",
	retrospectivesLifeboatDir + "/",
	"graveyard/abandoned.json",
}

// reportOnlyPrefixes are the lifeboat paths embark never writes but reads to
// inform the report (Ignored{report-only}). coverage.* is handled by prefix
// "coverage." so both coverage.json and coverage.md match. _provenance.json is a
// slash-less exact match. The M6 synthesis layer (principles.*, press-release.*,
// audit/**) is post-pack, host-delegated or deterministically derived; it is
// reported here so embark surfaces it as report-only rather than mis-flagging it
// Ignored{unknown} (it is neither a record family nor a foreign/tampered entry).
// This set is the embark-side complement of embarkFamilies: a file matches at
// most one, and a file matching neither is Ignored{unknown}.
var reportOnlyPrefixes = []string{
	"brief/",
	"coverage.",
	"graveyard/",
	"rescue/spine.md",
	"principles.",
	"press-release.",
	"review/",
	"audit/", // pre-spc-30 home; still recognised on an older pack
	ProvenanceName,
}

// manifestExcludedExact / manifestExcludedPrefixes name the on-disk lifeboat
// files that are NOT part of manifest_sha256, so verifyManifest (Agent B) can walk
// the tree and reproduce the pinned hash exactly. _provenance.json cannot hash
// itself; graveyard/lessons.json and graveyard/low-confidence/** are the mutable,
// post-pack, host-delegated layer-3 interpretation that IngestLessons writes into
// an already-sealed lifeboat and deliberately keeps out of the manifest. The M6
// synthesis layer (principles.json/.md, press-release.json/.md, and every
// review/** verdict artefact) is the same kind of post-pack mutable artifact — written
// into an already-sealed lifeboat, its integrity the per-entry cite-or-be-dropped
// rule and the registered-verdict gate, not the manifest seal — so it is excluded
// here too and verifyManifest still reproduces the pinned hash after synthesis.
var (
	manifestExcludedExact    = []string{ProvenanceName, "graveyard/lessons.json", "principles.json", "principles.md", "press-release.json", "press-release.md"}
	manifestExcludedPrefixes = []string{"graveyard/low-confidence/", "review/", "audit/"}
)
