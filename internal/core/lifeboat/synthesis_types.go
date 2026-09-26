package lifeboat

// synthesis_types.go — the SHARED TYPE + CONSTANT surface for M6 ("synthesis over
// the written record", itd-88). It carries ONLY the vocabulary the M6 verbs agree
// on — the three artifact schemas, the registered ReviewVerdict enum, the
// trust-boundary caps, and the id/semver grammars — so the parallel implementation
// agents build against one contract and cannot drift. It contains NO logic (the
// deterministic builders, the guarded readers, the cite-or-be-dropped validators,
// and the Render methods live in their own files):
//
//   - Agent A1 (synthesis_principles.go + synthesis_pressrelease.go): the
//     principles and press-release cores — SynthesizePrinciples / ComposePressRelease,
//     the deterministic evidence-only fallbacks, the delegated cite-or-be-dropped
//     validators, the .md renders, and the shared read/clean/filter helpers A2 reuses.
//   - Agent A2 (synthesis_review.go): the review core — ReviewLifeboat, the deterministic
//     manifest+coverage verdict mapping, the delegated verdict/finding validator, the
//     .md render. Depends on A1's shared helpers (same package, so a build-order dep).
//   - Agent B (surface/cli): the `disembark principles|press-release|review` verbs and
//     the three orchestration sections in commands/disembark.md.
//   - Agent C (agents/*.md + fixtures): the four host-delegated agent prompt files
//     (principle-distiller, graveyard-interpreter, press-release-composer,
//     lifeboat-reviewer) with itd-5 frontmatter and injection-canary fixtures.
//
// Every M6 artifact is a POST-PACK MUTABLE LAYER written into an already-sealed
// lifeboat, deliberately kept OUT of manifest_sha256 (mirroring graveyard/lessons.json,
// graveyard.go:29-36). Its integrity is the per-entry cite-or-be-dropped rule and the
// registered-verdict gate, NOT the manifest seal. Agent A1 adds these paths to the
// exclusion + report-only sets in embark_types.go (see the doc's "exclusion-set diff").
//
// No wall-clock anywhere: the review artefact is named review-<manifest12>.json, where
// manifest12 is the first 12 hex of the lifeboat's manifest_sha256 (shortHex over
// prov.ManifestSHA256) — deterministic and unique per content, a plan amendment to
// the plan's oracle-<ts>.json (DECISIONS.md entry is the orchestrator's).
//
// See the M6 section of .abcd/development/plans/2026-07-14-lifeboat-coverage-experiment.md,
// itd-5, and the M6 design record.

import "regexp"

// ---------------------------------------------------------------------------
// Schema versions. Each artifact is independently versioned (mirroring
// LessonsSchemaVersion / GraveyardSchemaVersion), so a future breaking change to
// one shape is detectable rather than silently misread.
// ---------------------------------------------------------------------------

const (
	// PrinciplesSchemaVersion stamps principles.json. It is 2 since every entry
	// carries claim_type, reference and comparison beside its evidence
	// (adr-2609021016270132, spc-2609020626042471); a payload at 1 is refused as
	// unsupported, so a consumer sees the change by the version.
	PrinciplesSchemaVersion = 2
	// PressReleaseSchemaVersion stamps press-release.json.
	PressReleaseSchemaVersion = 1
	// ReviewSchemaVersion stamps review/review-<manifest12>.json.
	ReviewSchemaVersion = 1
)

// ---------------------------------------------------------------------------
// Mode. Each synthesis artifact self-records how it was produced. A verb invoked
// WITHOUT its --*-json flag runs ModeDeterministic (an evidence-only fallback built
// from the packed lifeboat's own files); WITH the flag it runs ModeDelegated
// (validated untrusted model output). The mode lives in the artifact itself, never
// in _provenance.json — rewriting the sealed manifest header post-pack would break
// the artifact's integrity story for zero gain (see the doc's decision 3).
// ---------------------------------------------------------------------------

// SynthesisMode records how a synthesis artifact was produced.
type SynthesisMode string

const (
	// ModeDeterministic: built by the Go core from the packed lifeboat's own files,
	// no model in the loop. prompt_version is omitted.
	ModeDeterministic SynthesisMode = "deterministic"
	// ModeDelegated: validated host-delegated model output. prompt_version records
	// the agent prompt's semver.
	ModeDelegated SynthesisMode = "delegated"
)

// ---------------------------------------------------------------------------
// ReviewVerdict — the FIRST Go home of abcd's registered review-verdict enum
// (brief 02-constraints/04-naming.md:83, 05-internals/01-agents.md § Verdict-tag
// protocol). Membership-validated exactly like intent.verdictEnum: an out-of-enum
// verdict in a delegated payload is a structural refusal, never a silent coercion.
// ---------------------------------------------------------------------------

// ReviewVerdict is a lifeboat-reviewer review verdict.
type ReviewVerdict string

const (
	// VerdictShip: the lifeboat is a faithful, shippable proxy of the record.
	VerdictShip ReviewVerdict = "SHIP"
	// VerdictNeedsWork: shippable but with named, addressable gaps.
	VerdictNeedsWork ReviewVerdict = "NEEDS_WORK"
	// VerdictMajorRethink: the lifeboat does not faithfully carry the record.
	VerdictMajorRethink ReviewVerdict = "MAJOR_RETHINK"
)

// reviewVerdictEnum is the closed set of review verdicts.
var reviewVerdictEnum = map[ReviewVerdict]bool{
	VerdictShip: true, VerdictNeedsWork: true, VerdictMajorRethink: true,
}

// Valid reports whether v is a registered review verdict.
func (v ReviewVerdict) Valid() bool { return reviewVerdictEnum[v] }

// ---------------------------------------------------------------------------
// principles.json (+ principles.md). One entry per distilled principle; each cites
// evidence that must resolve to a live record id, a live graveyard finding id, or a
// packed lifeboat path (cite-or-be-dropped). Confidence reuses the coverage
// Confidence enum (high/medium/low); unlike lessons, a principle is NOT routed by
// confidence — every survivor lands in the single principles.json file.
// ---------------------------------------------------------------------------

// Principle is one distilled principle — both the untrusted delegated input shape
// and the written output shape. Principle (the prose) is sanitised and
// marker-neutralised before it is written into the lifeboat.
//
// The three claim keys beside Evidence are the record's principle keys
// (adr-2609021016270132): claim_type (criterion, causal or context), reference
// (the entity the principle is about) and comparison (what was compared to
// produce it). They are pointers so a declined claim is carried as a JSON null
// rather than omitted, and they are never omitempty: an absent key and a null
// one are different claims, and the validator reads the payload raw to tell
// them apart.
type Principle struct {
	ID         string     `json:"id"`
	Principle  string     `json:"principle"`
	Confidence Confidence `json:"confidence"`
	ClaimType  *string    `json:"claim_type"`
	Reference  *string    `json:"reference"`
	Comparison *string    `json:"comparison"`
	Evidence   []string   `json:"evidence"`
}

// PrinciplesFile is the on-disk shape of principles.json. PromptVersion is omitted
// in deterministic mode.
type PrinciplesFile struct {
	SchemaVersion int           `json:"schema_version"`
	Mode          SynthesisMode `json:"mode"`
	PromptVersion string        `json:"prompt_version,omitempty"`
	Principles    []Principle   `json:"principles"`
}

// PrincipleDrop records one delegated principle the validator refused to write, and
// why. A drop is reported, never fatal.
type PrincipleDrop struct {
	ID     string `json:"id"`
	Reason string `json:"reason"`
}

// PrinciplesResult is the transport-agnostic outcome of `disembark principles`.
type PrinciplesResult struct {
	LifeboatDir    string          `json:"lifeboat_dir"`
	Mode           SynthesisMode   `json:"mode"`
	Written        int             `json:"written"`
	Dropped        int             `json:"dropped"`
	Drops          []PrincipleDrop `json:"drops,omitempty"`
	PrinciplesPath string          `json:"principles_path,omitempty"`
	RenderPath     string          `json:"render_path,omitempty"`
}

// ---------------------------------------------------------------------------
// press-release.json (+ press-release.md). A single composed document (not a list of
// entries), derived from the packed brief, spine, and principles. Its Evidence must
// carry at least one ref that resolves to a packed lifeboat path (brief/**,
// rescue/spine.md, or principles.json); a delegated press release citing nothing
// resolvable is a whole-document refusal (structural), mirroring memory ingest's
// "refusing to write an unattributable page".
// ---------------------------------------------------------------------------

// PressReleaseQuote is one attributed pull-quote. Both fields are sanitised.
type PressReleaseQuote struct {
	Attribution string `json:"attribution"`
	Text        string `json:"text"`
}

// PressReleaseFile is the on-disk shape of press-release.json. PromptVersion is
// omitted in deterministic mode.
type PressReleaseFile struct {
	SchemaVersion int                 `json:"schema_version"`
	Mode          SynthesisMode       `json:"mode"`
	PromptVersion string              `json:"prompt_version,omitempty"`
	Headline      string              `json:"headline"`
	Subhead       string              `json:"subhead,omitempty"`
	Body          string              `json:"body"`
	Quotes        []PressReleaseQuote `json:"quotes,omitempty"`
	Evidence      []string            `json:"evidence"`
}

// PressReleaseResult is the transport-agnostic outcome of `disembark press-release`.
type PressReleaseResult struct {
	LifeboatDir      string        `json:"lifeboat_dir"`
	Mode             SynthesisMode `json:"mode"`
	EvidenceRefs     int           `json:"evidence_refs"`
	PressReleasePath string        `json:"press_release_path,omitempty"`
	RenderPath       string        `json:"render_path,omitempty"`
}

// ---------------------------------------------------------------------------
// review/review-<manifest12>.json (+ .md). The lifeboat-reviewer audit: a registered
// verdict, the manifest attestation, the packed coverage summary, and findings that
// each cite packed lifeboat paths (cite-or-be-dropped). Coverage reuses the coverage
// Summary shape. In deterministic mode the verdict is a mechanical mapping over
// VerifyManifest + the packed coverage summary; in delegated mode the model's
// verdict is membership-validated and its findings are cite-or-dropped.
// ---------------------------------------------------------------------------

// ReviewFinding is one audit finding. Finding (the prose) is sanitised; Evidence
// must cite packed lifeboat paths. Severity is optional, sanitised free text (no
// closed enum in M6).
type ReviewFinding struct {
	ID       string   `json:"id"`
	Severity string   `json:"severity,omitempty"`
	Finding  string   `json:"finding"`
	Evidence []string `json:"evidence"`
}

// ReviewFindingDrop records one delegated finding the validator refused to write.
type ReviewFindingDrop struct {
	ID     string `json:"id"`
	Reason string `json:"reason"`
}

// ReviewArtefact is the on-disk shape of review/review-<manifest12>.json. PromptVersion
// is omitted in deterministic mode. ManifestVerified is the VerifyManifest outcome —
// a false value is a MAJOR_RETHINK verdict input, NOT a fatal error.
type ReviewArtefact struct {
	SchemaVersion    int             `json:"schema_version"`
	Mode             SynthesisMode   `json:"mode"`
	PromptVersion    string          `json:"prompt_version,omitempty"`
	Verdict          ReviewVerdict   `json:"verdict"`
	SourceName       string          `json:"source_name"`
	ManifestSHA256   string          `json:"manifest_sha256"`
	ManifestVerified bool            `json:"manifest_verified"`
	Coverage         Summary         `json:"coverage"`
	Findings         []ReviewFinding `json:"findings"`
}

// ReviewResult is the transport-agnostic outcome of `disembark review`.
type ReviewResult struct {
	LifeboatDir string              `json:"lifeboat_dir"`
	Mode        SynthesisMode       `json:"mode"`
	Verdict     ReviewVerdict       `json:"verdict"`
	Written     int                 `json:"written"`
	Dropped     int                 `json:"dropped"`
	Drops       []ReviewFindingDrop `json:"drops,omitempty"`
	ReviewPath  string              `json:"review_path,omitempty"`
	RenderPath  string              `json:"render_path,omitempty"`
}

// ---------------------------------------------------------------------------
// Trust-boundary caps. The --*-json payloads are untrusted host/model output, read
// behind the same guards as an intent verdict or a lessons payload (regular file, no
// symlink, size cap). Every ceiling mirrors a maxLessons* constant so the two seams
// stay legible together.
// ---------------------------------------------------------------------------

const (
	// maxSynthesisBytes caps any untrusted synthesis payload (mirrors maxLessonsBytes).
	maxSynthesisBytes = 1 << 20 // 1 MiB
	// maxPrinciples caps how many principles one delegated ingest may carry.
	maxPrinciples = 1000
	// maxReviewFindings caps how many findings one delegated audit may carry.
	maxReviewFindings = 1000
	// maxPressReleaseQuotes caps the pull-quotes one press release may carry.
	maxPressReleaseQuotes = 64
	// maxSynthEvidenceRefs caps the evidence refs read per entry (mirrors
	// maxLessonEvidenceRefs).
	maxSynthEvidenceRefs = 128
	// maxSynthProseBytes caps a single prose field (principle, finding, headline,
	// subhead, quote) after sanitisation (mirrors maxLessonProseBytes).
	maxSynthProseBytes = 4096
	// maxPressReleaseBodyBytes caps the press-release body — a longer-form field
	// than a one-line principle or finding.
	maxPressReleaseBodyBytes = 16 << 10 // 16 KiB
	// maxSynthIDLen bounds a synthesis id's length; paired with the id regexes it is
	// the path-traversal defence for any id that could reach a filename.
	maxSynthIDLen = 64
)

// MaxSynthesisBytes is the exported cap a front door uses to bound its read of an
// untrusted synthesis payload (file or stdin) before handing it to a core verb,
// which enforces the same ceiling internally (mirrors MaxLessonsBytes).
const MaxSynthesisBytes = maxSynthesisBytes

// ---------------------------------------------------------------------------
// Id + evidence grammars. Every synthesis id is kebab-case with a namespacing
// prefix so it can never collide with a graveyard finding id or build a path that
// escapes its family; the caps pair with maxSynthIDLen before any id is used.
// ---------------------------------------------------------------------------

var (
	// prnIDRe constrains a principle id: "prn-" then kebab-case [a-z0-9] segments.
	prnIDRe = regexp.MustCompile(`^prn-[a-z0-9]+(?:-[a-z0-9]+)*$`)
	// fndIDRe constrains a review finding id: "fnd-" then kebab-case [a-z0-9] segments.
	fndIDRe = regexp.MustCompile(`^fnd-[a-z0-9]+(?:-[a-z0-9]+)*$`)
	// synthRecordIDRe classifies a record-id evidence ref (adr-N / itd-N / iss-N).
	// A ref matching this shape is valid iff it is also a member of the live record-id
	// set discovered in the packed lifeboat; a ref not matching it is validated as a
	// packed lifeboat path instead (membership in the walked file set).
	synthRecordIDRe = regexp.MustCompile(`^(?:adr|itd|iss)-[0-9]+$`)
	// promptVersionRe validates a delegated payload's prompt_version as semver-shaped
	// (itd-5); the deterministic verbs omit it entirely.
	promptVersionRe = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
)
