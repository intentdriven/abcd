// Package ideate is the deterministic frame around abcd's idea-admission
// protocol (itd-104, spc-18): the three-leg gauntlet an idea may be put through
// before it is allowed to become a record entry.
//
// DIVISION OF LABOUR. Ideate follows the pattern the disembark synthesis family
// proved. The HOST runs the judgement — primary-source research, a grill against
// the existing record, and an independent adversarial review — as agents. This
// package is the frame: it validates what those legs produced, proves every
// citation names a real record, and writes the durable verdict. It never fetches
// a source, never calls a model, and never decides whether an idea is any good.
//
// WHY A RECORD EITHER WAY. The verdict is written whether the idea survives or
// dies, and it must carry the alternatives that were rejected. A killed idea with
// no recorded reason is an idea that gets re-proposed in six months; the research
// directory is where a session looks before re-proposing one. That is why an empty
// rejected-alternatives list is admissible only behind an explicit marker —
// silence by omission is exactly the failure the record exists to prevent.
//
// NEVER A GATE. Nothing here is reachable from capture or intent-create. Ideate is
// an optional verb the routing help names for big, unproven ideas; skipping it is
// never warned about, and no other verb requires it.
package ideate

import (
	"regexp"
	"time"
)

// SchemaVersion stamps the verdict payload. It is versioned independently of
// every other abcd artifact, so a future breaking change to the shape is
// detectable rather than silently misread by a host composing against the old one.
const SchemaVersion = 1

// MaxPayloadBytes caps the untrusted verdict document, mirroring the synthesis and
// release-ingest ceilings. It is exported so a front door can bound its READ at
// the same ceiling the core enforces — the front-door bound is a convenience, this
// one is the guarantee.
const MaxPayloadBytes = 1 << 20 // 1 MiB

// MaxSlugLen bounds the idea slug, which reaches a filename. Paired with slugRe it
// is the lexical half of the write-containment defence.
const MaxSlugLen = 64

// ResearchRelDir is the one directory a verdict record is ever written into, and
// DecisionsRelDir is the log its dated pointer is appended to. Both are constants
// rather than parameters: a configurable target would let a verdict land somewhere
// no future session looks.
//
// It is the research store's notes/ bucket rather than its root because a verdict
// record IS a dated research note (spc-18: "killed ideas are research outcomes,
// and the research directory is where a future session looks before re-proposing
// one"), and research/README.md files dated notes under notes/. Writing to the
// root would make the shipped binary the standing producer of the very
// convention violation iss-42 cleared out of that directory by hand.
const (
	ResearchRelDir  = ".abcd/development/research/notes"
	DecisionsRelDir = ".abcd/work/DECISIONS.md"
)

// decisionsLockRel is the advisory lock serialising the decision log's
// read-modify-write across concurrent abcd processes in one worktree. It is a
// runtime artefact beside the log it guards, named like capture's allocator lock
// and gitignored on the same footing.
const decisionsLockRel = ".abcd/work/.decisions.lock"

// decisionsLockTimeout bounds the wait for that lock. A verdict record is a
// once-per-idea act, so contention is brief when it happens at all; waiting
// longer than this means something is wedged, and saying so beats hanging.
const decisionsLockTimeout = 5 * time.Second

const (
	// maxClaims caps the research leg's claims table.
	maxClaims = 200
	// maxHits caps the record grill's hits.
	maxHits = 200
	// maxKillAttempts caps the adversarial leg's attempts.
	maxKillAttempts = 200
	// maxRejected caps the rejected-alternatives list.
	maxRejected = 200
	// maxProseBytes caps one prose field after cleaning (mirrors the synthesis
	// prose ceiling).
	maxProseBytes = 4096
	// maxIdeaBytes caps the idea text — a longer-form field than a single claim.
	maxIdeaBytes = 8192
	// maxSourceBytes caps a claim's primary-source reference (a URL or a document
	// citation, never a document).
	maxSourceBytes = 1024
	// maxRecordIDBytes bounds a cited id BEFORE it is matched or reported. The id
	// grammar admits unbounded digits, so without a cap "itd-" followed by a
	// megabyte of zeros is legal, fails resolution, and is echoed whole into the
	// operator's terminal.
	maxRecordIDBytes = 32
	// maxDecisionsBytes caps the read of the decision log before its line is
	// appended.
	maxDecisionsBytes = 4 << 20 // 4 MiB
)

var (
	// slugRe is the idea slug's grammar: lower-case kebab-case, no leading or
	// trailing hyphen, no doubled hyphen. It admits no separator, no dot segment,
	// and no absolute form, so a slug can never build a path outside the research
	// directory — the lexical layer of the two-layer containment (os.Root is the
	// other).
	slugRe = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	// promptVersionRe validates the composing agents' prompt_version (itd-5), so a
	// verdict record can be traced to the prompts that produced it.
	promptVersionRe = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
)

// Verdict is the admission outcome. The set is CLOSED: an unregistered verdict is
// a whole-document refusal, never a coercion, because the verdict is the one line
// of the record a later session reads first.
type Verdict string

const (
	// VerdictSurvives: the idea cleared all three legs and may graduate to a draft
	// intent through the existing quoted-text create. Ideate mints no intent itself.
	VerdictSurvives Verdict = "survives"
	// VerdictKilled: a leg was fatal. The record is the reason it stays dead.
	VerdictKilled Verdict = "killed"
	// VerdictReframed: the idea as posed does not survive, but a reframing of it
	// does; the record carries both.
	VerdictReframed Verdict = "reframed"
)

var verdictEnum = map[Verdict]bool{VerdictSurvives: true, VerdictKilled: true, VerdictReframed: true}

// Valid reports whether v is a registered verdict.
func (v Verdict) Valid() bool { return verdictEnum[v] }

// LegKind identifies one leg of the gauntlet. The three values are also the
// REQUIRED ORDER: research grounds the idea, the grill checks it against what is
// already written, and only then does an evaluator who did none of that work try
// to kill it. Running them in any other order changes what each leg is looking at,
// so the order is validated rather than assumed.
type LegKind string

const (
	// LegResearch: every load-bearing claim checked against its primary source.
	LegResearch LegKind = "research"
	// LegRecordGrill: does the existing record cover, contradict, or supersede
	// this? Every hit is cited by record id.
	LegRecordGrill LegKind = "record-grill"
	// LegAdversarialReview: a fresh-context, off-policy evaluator that did not
	// conduct the research, receiving the idea as an artefact of unknown
	// authorship, trying to kill it.
	LegAdversarialReview LegKind = "adversarial-review"
)

// legOrder is both the membership set and the required order, written once so the
// two can never disagree.
var legOrder = []LegKind{LegResearch, LegRecordGrill, LegAdversarialReview}

// ClaimStatus is what checking a claim against its primary source established.
type ClaimStatus string

const (
	// ClaimVerified: the primary source supports the claim.
	ClaimVerified ClaimStatus = "verified"
	// ClaimFalsified: the primary source contradicts it. Three of these killed an
	// idea in the proven manual run.
	ClaimFalsified ClaimStatus = "falsified"
	// ClaimUnverifiable: no primary source settles it either way. This is an
	// honest outcome, not a pass.
	ClaimUnverifiable ClaimStatus = "unverifiable"
)

var claimStatusEnum = map[ClaimStatus]bool{ClaimVerified: true, ClaimFalsified: true, ClaimUnverifiable: true}

// Relation is how an existing record entry bears on the idea.
type Relation string

const (
	// RelationCovered: the entry already says this.
	RelationCovered Relation = "covered"
	// RelationContradicted: the entry rules this out.
	RelationContradicted Relation = "contradicted"
	// RelationSuperseded: the entry has moved past this.
	RelationSuperseded Relation = "superseded"
)

var relationEnum = map[Relation]bool{RelationCovered: true, RelationContradicted: true, RelationSuperseded: true}

// KillOutcome is what one adversarial kill attempt achieved.
type KillOutcome string

const (
	// KillSurvived: the idea withstood the attempt.
	KillSurvived KillOutcome = "survived"
	// KillPartial: the attempt landed but is not on its own fatal.
	KillPartial KillOutcome = "partial"
	// KillFatal: the attempt is fatal to the idea as posed.
	KillFatal KillOutcome = "fatal"
)

var killOutcomeEnum = map[KillOutcome]bool{KillSurvived: true, KillPartial: true, KillFatal: true}

// Claim is one load-bearing claim and the primary source it was checked against.
//
// The binary cannot judge whether a source is PRIMARY — that is the researching
// agent's obligation, stated in its prompt. What the binary enforces is that a
// source is named at all, for every claim including an unverifiable one: a claims
// table row with no source is indistinguishable from an unchecked assertion.
type Claim struct {
	Claim         string      `json:"claim"`
	PrimarySource string      `json:"primary_source"`
	Status        ClaimStatus `json:"status"`
}

// GrillHit is one existing record entry that bears on the idea. Record is a cited
// id and MUST resolve in the repository; Note is optional prose, and is where a
// reference to a brief section or a principle lives, since neither carries a
// per-entry id that could be cited.
type GrillHit struct {
	Record   string   `json:"record"`
	Relation Relation `json:"relation"`
	Note     string   `json:"note,omitempty"`
}

// KillAttempt is one adversarial attempt on the idea and what it achieved.
type KillAttempt struct {
	Attempt string      `json:"attempt"`
	Outcome KillOutcome `json:"outcome"`
}

// Leg is one leg of the gauntlet. It is a union over the three kinds: each kind
// populates exactly its own evidence field, and a leg carrying another kind's
// field is a refusal — a payload that smuggles grill hits into the research leg is
// not a leg abcd knows how to read.
type Leg struct {
	Kind         LegKind       `json:"kind"`
	Claims       []Claim       `json:"claims,omitempty"`
	Hits         []GrillHit    `json:"hits,omitempty"`
	KillAttempts []KillAttempt `json:"kill_attempts,omitempty"`
}

// RejectedAlternative is one option considered and turned down, with the reason.
// Both fields are required: an alternative with no reason records that a choice
// existed without recording why it was not taken, which is the half of the entry
// that stops the re-litigation.
type RejectedAlternative struct {
	Alternative string `json:"alternative"`
	WhyRejected string `json:"why_rejected"`
}

// Payload is the whole untrusted host-composed verdict document.
type Payload struct {
	// SchemaVersion must equal SchemaVersion.
	SchemaVersion int `json:"schema_version"`
	// PromptVersion is the orchestrating prompt's semver (itd-5).
	PromptVersion string `json:"prompt_version"`
	// Idea is the idea as captured, in the words it entered the gauntlet with.
	Idea string `json:"idea"`
	// Legs are the three legs, IN ORDER. An array rather than three named fields
	// precisely so the order is a property of the document the binary can check.
	Legs []Leg `json:"legs"`
	// Verdict is the admission outcome.
	Verdict Verdict `json:"verdict"`
	// RejectedAlternatives are the options turned down on the way to the verdict.
	RejectedAlternatives []RejectedAlternative `json:"rejected_alternatives"`
	// NoRejectedAlternatives is the EXPLICIT empty marker. An empty list is
	// admissible only with this set; omission is not, and setting it beside a
	// populated list is a contradiction the document is refused for.
	NoRejectedAlternatives bool `json:"no_rejected_alternatives,omitempty"`
}

// ClaimTally counts the research leg's outcomes, so a reader of the result knows
// what the claims table found without re-reading the record.
type ClaimTally struct {
	Verified     int `json:"verified"`
	Falsified    int `json:"falsified"`
	Unverifiable int `json:"unverifiable"`
}

// Total reports how many claims the research leg carried.
func (t ClaimTally) Total() int { return t.Verified + t.Falsified + t.Unverifiable }

// Result is the transport-agnostic outcome of `ideate record`. It is returned only
// when the record was written; every refusal is an error, so a caller holding a
// Result knows a durable record exists.
type Result struct {
	Slug    string  `json:"slug"`
	Idea    string  `json:"idea"`
	Verdict Verdict `json:"verdict"`
	// Graduates reports whether this verdict admits the idea to a draft intent.
	// Ideate mints nothing itself; this is the signal the surface relays.
	Graduates bool `json:"graduates"`
	// Path is the verdict record, repo-relative.
	Path string `json:"path"`
	// DecisionPath is the log the dated pointer was appended to, repo-relative.
	DecisionPath string `json:"decision_path"`
	// Claims tallies the research leg.
	Claims ClaimTally `json:"claims"`
	// GrillHits counts the record-grill hits.
	GrillHits int `json:"grill_hits"`
	// CitedRecords is the sorted set of record ids the grill cited, every one of
	// them proved to resolve — the citation gate's proof, so a reviewer can check
	// the record without re-running the resolution.
	CitedRecords []string `json:"cited_records"`
	// KillAttempts counts the adversarial leg's attempts.
	KillAttempts int `json:"kill_attempts"`
	// RejectedAlternatives counts the recorded alternatives.
	RejectedAlternatives int `json:"rejected_alternatives"`
	// Redactions counts the secret/PII spans the store-before-commit redactor
	// rewrote out of the verdict's free text on the way to the record. It is
	// reported rather than kept quiet because redacting in silence is how a
	// composer learns nothing about the credential it just pasted (loud-staging).
	Redactions int `json:"redactions"`
}

// recordDate renders the date a verdict record is filed under. UTC so the same
// run files the same record everywhere.
func recordDate(at time.Time) string { return at.UTC().Format("2006-01-02") }
