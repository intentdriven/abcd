package intent

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"syscall"

	"github.com/intentdriven/abcd/internal/core/condition"
	"github.com/intentdriven/abcd/internal/core/mdrecord"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/core/spec"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// audit.go — the intent-audit outbox+inbox (itd-80 phase 4; renamed from
// review per adr-40/spc-28: it emits family-2 verdicts, so it is the audit).
//
// The intent file IS the record: its `## Audit Notes` section holds one machine
// marker per review receipt, so idempotency and review state live in one
// committed place (directory/file-as-truth) with no side database.
//
// Two flows meet here:
//
//   - EMIT (emitAuditForIntent, called by Reconcile after a ship move): parks an
//     OWED stub in the shipped intent's Audit Notes and writes an ephemeral review
//     request under .abcd/.work.local/reviews/. Report-only — the caller treats a
//     failure as non-fatal (the intent still ships).
//   - INGEST (IngestVerdict): reads an untrusted verdict JSON emitted by the
//     host-delegated intent-auditor agent, validates it FAIL-CLOSED against the
//     schema and against the parked OWED receipt, then either replaces the OWED
//     stub with the rendered verdict (INGESTED) or quarantines a bad payload
//     (DEAD_LETTER) — never a partial application.
//
// Receipt digest. resource_digest = sha256 over the intent's `## Acceptance
// Criteria` section body — the authority the reviewer judges and the only intent
// text the criteria map onto. It deliberately EXCLUDES the Audit Notes section,
// so writing the marker does not change the receipt: re-emit and re-ingest stay
// idempotent. receipt_id = "rcp-" + first-12-hex of sha256(intent_id | spec_id |
// hex(resource_digest)). No timestamps feed it (deterministic).

// VerdictType is the only _type the ingest accepts.
const VerdictType = "abcd/intent-fidelity-verdict/v1"

// reviewsRelDir is the ephemeral (gitignored) review outbox/quarantine.
const reviewsRelDir = ".abcd/.work.local/reviews"

// maxVerdictBytes caps the untrusted verdict payload (trust boundary).
const maxVerdictBytes = 1 * 1024 * 1024

// verdictEnum is the closed set of acceptance verdicts.
var verdictEnum = map[string]bool{
	"MET": true, "MET_WITH_CONCERNS": true, "NOT_MET": true, "INCONCLUSIVE": true,
}

// dispositionUntested is the disposition vocabulary's word for the absence of a
// judgement — the value the quarantine path records, and the only one exempt
// from the cited-evidence rule. It is core/condition's value: the vocabulary is
// shared with the condition verb and every reader of both writers' blocks.
const dispositionUntested = condition.Untested

// dispositionNarrowed is the one disposition that requires a stated narrowing —
// and the only one permitted to carry one.
const dispositionNarrowed = condition.Narrowed

// dispositionEnum is the closed set of scope-condition dispositions (spc-59),
// as a set over core/condition's Enum. It is deliberately disjoint from
// verdictEnum: a condition is not a criterion, and an acceptance verdict is not
// a judgement about an ex-ante assumption.
var dispositionEnum = func() map[string]bool {
	m := make(map[string]bool, len(condition.Enum))
	for _, v := range condition.Enum {
		m[v] = true
	}
	return m
}()

var (
	// rcpIDRe constrains a receipt id so it can never build a path that escapes
	// the reviews dir (path-traversal defence). 12 lowercase hex chars.
	rcpIDRe = regexp.MustCompile(`^rcp-[0-9a-f]{12}$`)
	// auditHeadingRe matches the `## Audit Notes` heading (any heading depth).
	// The review block's reader and writer find the section through
	// condition.AuditNotes; the section-body proofs read it here.
	auditHeadingRe = regexp.MustCompile(`^#{1,6}\s+Audit Notes\s*$`)
	// markerRe matches a parked review marker LINE. It is line-anchored and
	// whole-line on purpose: the marker is the ledger's own review state, and an
	// unanchored pattern would find one anywhere in the record's bytes —
	// mid-sentence inside a rendered verdict field, for instance, where an
	// untrusted payload put it. termsafe's cleaner is the first defence (it
	// breaks `<!` and `-->` in every field it writes, code span or not); this is
	// the second, so a marker has to occupy a line of its own to count.
	//
	// It is a pattern over ONE line, never over the record: readReviewBlocks is
	// the only caller, and it offers only the live lines of the live Audit Notes
	// section, so a marker-shaped line in a fence, a comment span or another
	// section is never matched (iss-2609020529185438).
	//
	// The grammar is core/condition's ReviewMarkerRe, shared with the condition
	// block's reader.
	markerRe = condition.ReviewMarkerRe
	// auditPlaceholderRe matches an intent template's Audit Notes placeholder,
	// dropped when the first real review block lands so a populated audit carries no
	// stale "Empty" claim. It tolerates both delimiter styles the templates have
	// used — italic `_Empty. Populated by ..._` and angle-bracket `<Empty until ...>`
	// — so template wording drift does not silently leave the placeholder behind. A
	// real audit line never starts with `_Empty`/`<Empty` (they start with `<!--`,
	// `Fidelity review`, `Provenance:`, or a `- ac-` bullet), so this cannot eat one.
	auditPlaceholderRe = regexp.MustCompile(`^\s*[_<]Empty\b.*[_>]\s*$`)
	// criterionIDRe validates a criterion id shape before it is positionally bounded.
	criterionIDRe = regexp.MustCompile(`^ac-([0-9]+)$`)
	// sha256FieldRe is the shape the auditor's contract publishes for the two
	// policy hashes and for an attestation digest: `sha256:<64 lowercase hex>`.
	// A field is a validated shape only where a validator says so — the
	// alternative is free text under a structural name, which is how a home path
	// arrived in a field the record called a hash (iss-2609022002241168).
	sha256FieldRe = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
)

// ---------------------------------------------------------------------------
// Verdict schema (hand-rolled; stdlib encoding/json only)
// ---------------------------------------------------------------------------

type verdict struct {
	Type              string             `json:"_type"`
	ReceiptID         string             `json:"receipt_id"`
	Verifier          verdictVerifier    `json:"verifier"`
	Policy            verdictPolicy      `json:"policy"`
	InputAttestations []verdictAttest    `json:"input_attestations"`
	Criteria          []verdictCriterion `json:"criteria"`
	AcceptanceRollup  map[string]int     `json:"acceptance_rollup"`
	GapAudit          verdictGapAudit    `json:"gap_audit"`
	ScopeConditions   []verdictCondition `json:"scope_conditions"`
}

type verdictVerifier struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

type verdictPolicy struct {
	RubricHash string `json:"rubric_hash"`
	PromptHash string `json:"prompt_hash"`
}

type verdictAttest struct {
	Kind   string `json:"kind"`
	Ref    string `json:"ref"`
	Digest string `json:"digest"`
}

type verdictEvidence struct {
	Ref   string `json:"ref"`
	Quote string `json:"quote"`
}

type verdictCriterion struct {
	CriterionID string            `json:"criterion_id"`
	Verdict     string            `json:"verdict"`
	Rationale   string            `json:"rationale"`
	Evidence    []verdictEvidence `json:"evidence"`
}

// verdictCondition is one scope-condition disposition, keyed to the identity
// spc-55 minted for the condition rather than to its wording — so a reworded
// condition keeps its judgement and a narrowing is stated rather than implied by
// edited prose.
type verdictCondition struct {
	ConditionID string            `json:"condition_id"`
	Disposition string            `json:"disposition"`
	Rationale   string            `json:"rationale"`
	Narrowing   string            `json:"narrowing"`
	Evidence    []verdictEvidence `json:"evidence"`
}

type verdictGapEntry struct {
	Claim    string            `json:"claim"`
	Evidence []verdictEvidence `json:"evidence"`
}

type verdictGapAudit struct {
	Honoured []verdictGapEntry `json:"honoured"`
	Diverged []verdictGapEntry `json:"diverged"`
	Missing  []verdictGapEntry `json:"missing"`
}

// ---------------------------------------------------------------------------
// Results
// ---------------------------------------------------------------------------

// AuditEmitResult reports one emit (OWED stub + request file).
//
// Status names the receipt's state and RequestWritten names the act, because
// the two differ: an emit on a receipt already OWED rewrites its request, and
// reported only already_owed, which a caller read as "nothing happened"
// (iss-2609190337598356). RequestPath is the request this emit wrote, so a
// terminal receipt — whose emit writes nothing — names none.
type AuditEmitResult struct {
	ReceiptID      string `json:"receipt_id"`
	IntentID       string `json:"intent_id"`
	Status         string `json:"status"` // owed | already_owed | already_ingested | already_dead_letter
	RequestPath    string `json:"request_path,omitempty"`
	RequestWritten bool   `json:"request_written"`
}

// IngestVerdictResult reports one verdict ingest.
type IngestVerdictResult struct {
	Status         string `json:"status"` // ingested | dead_letter | noop
	ReceiptID      string `json:"receipt_id"`
	IntentID       string `json:"intent_id"`
	Criteria       int    `json:"criteria"`
	Met            int    `json:"met"`
	MetWithConcern int    `json:"met_with_concerns"`
	NotMet         int    `json:"not_met"`
	Inconclusive   int    `json:"inconclusive"`
	Conditions     int    `json:"conditions"`
	Survived       int    `json:"survived"`
	Narrowed       int    `json:"narrowed"`
	Falsified      int    `json:"falsified"`
	Untested       int    `json:"untested"`
	DeadLetterPath string `json:"dead_letter_path,omitempty"`
	Reason         string `json:"reason,omitempty"`
	// Replaced is set when the ingest replaced a verdict already ingested for
	// the receipt: a re-ingest whose payload renders differently from the block
	// on the record. A re-ingest that renders identically is a noop.
	Replaced bool `json:"replaced,omitempty"`
	// ReadingOccasionedStanding is every condition-block disposition the fold
	// still reports as standing after this write: the verdict did not override
	// it, because its rationale did not name the block's occasion
	// (spc-2609020626046252). An auditor who meant to override one names its
	// occasion in the rationale and ingests again for the same receipt: a
	// payload that renders differently replaces the ingested block (Replaced).
	ReadingOccasionedStanding []condition.Disposition `json:"reading_occasioned_standing,omitempty"`
}

// MarshalJSON writes the result the way its outcome reads (iss-2609190337545165).
// One struct serves every status, so its counters are zero-valued members on a
// quarantine and a noop, and a reader took a dead letter's "criteria: 0" beside
// the conditions it recorded untested for a rollup. The JSON therefore states
// what the ingest recorded — `verdict` (ingested), `quarantine` (dead_letter) or
// `nothing` (noop) — and carries the rollup only beside a recorded verdict,
// where a zero is a count. A quarantine states the one split it did record,
// every scope condition untested, under a name of its own.
func (r IngestVerdictResult) MarshalJSON() ([]byte, error) {
	type rollup struct {
		Criteria       int `json:"criteria"`
		Met            int `json:"met"`
		MetWithConcern int `json:"met_with_concerns"`
		NotMet         int `json:"not_met"`
		Inconclusive   int `json:"inconclusive"`
		Conditions     int `json:"conditions"`
		Survived       int `json:"survived"`
		Narrowed       int `json:"narrowed"`
		Falsified      int `json:"falsified"`
		Untested       int `json:"untested"`
	}
	out := struct {
		Status    string `json:"status"`
		ReceiptID string `json:"receipt_id"`
		IntentID  string `json:"intent_id"`
		Recorded  string `json:"recorded"`
		// A nil embedded pointer contributes no members at all.
		*rollup
		ConditionsUntested        *int                    `json:"conditions_untested,omitempty"`
		DeadLetterPath            string                  `json:"dead_letter_path,omitempty"`
		Reason                    string                  `json:"reason,omitempty"`
		Replaced                  bool                    `json:"replaced,omitempty"`
		ReadingOccasionedStanding []condition.Disposition `json:"reading_occasioned_standing,omitempty"`
	}{
		Status: r.Status, ReceiptID: r.ReceiptID, IntentID: r.IntentID,
		DeadLetterPath: r.DeadLetterPath, Reason: r.Reason, Replaced: r.Replaced,
		ReadingOccasionedStanding: r.ReadingOccasionedStanding,
	}
	switch r.Status {
	case "ingested":
		out.Recorded = "verdict"
		out.rollup = &rollup{
			Criteria: r.Criteria, Met: r.Met, MetWithConcern: r.MetWithConcern, NotMet: r.NotMet,
			Inconclusive: r.Inconclusive, Conditions: r.Conditions, Survived: r.Survived,
			Narrowed: r.Narrowed, Falsified: r.Falsified, Untested: r.Untested,
		}
	case "dead_letter":
		out.Recorded = "quarantine"
		n := r.Untested
		out.ConditionsUntested = &n
	default:
		out.Recorded = "nothing"
	}
	return json.Marshal(out)
}

// ---------------------------------------------------------------------------
// Emit (called by Reconcile; also the manual re-emit verb)
// ---------------------------------------------------------------------------

// receiptFor computes the deterministic receipt id for an intent/spec pair from
// the intent's Acceptance Criteria section (see the package-level digest note).
func receiptFor(intentID, specID, content string) string {
	acBody := sectionBody(content, acHeadingRe)
	rd := sha256.Sum256([]byte(acBody))
	h := sha256.Sum256([]byte(intentID + "|" + specID + "|" + hex.EncodeToString(rd[:])))
	return "rcp-" + hex.EncodeToString(h[:])[:12]
}

// emitAuditForIntent parks an OWED stub in the intent's Audit Notes and writes
// the ephemeral review request. It is idempotent: if a marker for the computed
// receipt already exists (OWED/INGESTED/DEAD_LETTER) the Audit Notes are left
// untouched. All ids are validated before any path is built.
func emitAuditForIntent(repoRoot string, it Intent) (AuditEmitResult, error) {
	return emitAuditWith(repoRoot, it, AuditEmitOptions{})
}

// AuditEmitOptions carries what a front door adds to an emitted request.
type AuditEmitOptions struct {
	// RoutingSection is the request block's routing section
	// (itd-2609170822093401): the tier and fan-out bound the host is asked to
	// run the auditor at, rendered by the front door from the resolved route.
	// It lands after the provenance block, outside the hashed prompt, so the
	// verdict's prompt_hash does not move with the machine's routing.
	RoutingSection string
}

func emitAuditWith(repoRoot string, it Intent, opts AuditEmitOptions) (AuditEmitResult, error) {
	if !recordid.ValidIntentID(it.ID) {
		return AuditEmitResult{}, fmt.Errorf("intent: id %q must match ^itd-[0-9]+$", it.ID)
	}
	// The stored spec_id is checked the tolerant way (a number must be readable
	// from it), not against the strict argument grammar: record-lint accepts a
	// slug-suffixed or zero-padded spec_id, and no path is built from this value.
	if !spec.HasNum(it.SpecID) {
		return AuditEmitResult{}, fmt.Errorf("intent: spec id %q must carry a spec number (spc-N)", it.SpecID)
	}
	// The read, the marker judgement and the writes are ONE critical section
	// under the store's advisory lock (iss-2609261935407925): the emit parks its
	// stub on the bytes it read, so a condition disposition or a verdict ingest
	// landing between an unlocked read and the write would be erased, with both
	// verbs exiting 0. Held here, the emit judges the record that writer left.
	var res AuditEmitResult
	err := withIntentMintLock(repoRoot, func() error {
		var err error
		res, err = emitLocked(repoRoot, it, opts)
		return err
	})
	return res, err
}

// emitLocked is emitAuditWith's critical section, called under the intent
// store lock.
func emitLocked(repoRoot string, it Intent, opts AuditEmitOptions) (AuditEmitResult, error) {
	abs := filepath.Join(repoRoot, it.Path)
	data, err := readRepoFile(abs, it.Path)
	if err != nil {
		return AuditEmitResult{}, err
	}
	content := string(data)

	// Reuse an already-parked receipt rather than recomputing one. The receipt
	// digest excludes the Audit Notes section, but creating that section on the
	// first emit (when it was absent and the Acceptance Criteria was the file's
	// last section) can still shift what sectionBody reads as the AC body — so a
	// freshly recomputed receipt may disagree with the parked marker and append a
	// second stub. The parked marker is the authority the ingest resolves against.
	if rcp, state, ok := existingMarker(content); ok {
		res := AuditEmitResult{ReceiptID: rcp, IntentID: it.ID}
		switch state {
		case "INGESTED":
			res.Status = "already_ingested"
		case "DEAD_LETTER":
			res.Status = "already_dead_letter"
		default:
			res.Status = "already_owed"
			// Only an OWED receipt still awaits a verdict: ensure its ephemeral
			// request still exists (it is gitignored and may have been swept). A
			// terminal INGESTED/DEAD_LETTER receipt needs no request rewrite.
			if err := writeAuditRequest(repoRoot, it, rcp, content, opts); err != nil {
				return res, err
			}
			res.RequestPath = filepath.Join(reviewsRelDir, rcp+".request.md")
			res.RequestWritten = true
		}
		return res, nil
	}

	rcp := receiptFor(it.ID, it.SpecID, content)
	res := AuditEmitResult{ReceiptID: rcp, IntentID: it.ID}
	block := owedBlock(rcp)
	updated := upsertReviewBlock(content, rcp, block)
	// The request is written before the intent file (iss-2609252127427592): a
	// request that cannot be written then leaves the intent untouched, rather
	// than parking an OWED stub no request backs. The reverse failure, an intent
	// write refused after its request landed, leaves only a gitignored request
	// that the next emit of the same content rewrites under the same receipt.
	// Either way an error here means no stub was parked.
	if err := writeAuditRequest(repoRoot, it, rcp, updated, opts); err != nil {
		return AuditEmitResult{}, err
	}
	if err := writeIntentFile(abs, it.Path, updated); err != nil {
		return AuditEmitResult{}, err
	}
	res.Status = "owed"
	res.RequestPath = filepath.Join(reviewsRelDir, rcp+".request.md")
	res.RequestWritten = true
	return res, nil
}

// ReEmitAudit handles the manual `abcd intent audit <itd-N>` verb for a shipped
// intent. It resolves the intent, refuses one not in shipped/, and delegates to
// emitAuditForIntent. Behaviour depends on the intent's current review state: an
// OWED receipt (or none) (re-)parks the OWED stub and rewrites its ephemeral
// request; a TERMINAL receipt is not re-reviewed — an already-INGESTED or
// already-DEAD_LETTER receipt returns that status unchanged (re-reviewing would
// discard the recorded audit), so the caller learns the review is already
// resolved rather than silently receiving a fresh stub.
func ReEmitAudit(repoRoot, intentID string) (AuditEmitResult, error) {
	return ReEmitAuditWith(repoRoot, intentID, AuditEmitOptions{})
}

// ReEmitAuditWith is ReEmitAudit with what the front door adds to the request.
func ReEmitAuditWith(repoRoot, intentID string, opts AuditEmitOptions) (AuditEmitResult, error) {
	if !recordid.ValidIntentID(intentID) {
		return AuditEmitResult{}, fmt.Errorf("intent: id %q must match ^itd-[0-9]+$", intentID)
	}
	corpus, err := Load(repoRoot)
	if err != nil {
		return AuditEmitResult{}, err
	}
	it, ok := corpus.Lookup(intentID)
	if !ok {
		return AuditEmitResult{}, fmt.Errorf("intent: %s not found in any bucket", intentID)
	}
	if it.Bucket != BucketShipped {
		return AuditEmitResult{}, fmt.Errorf("intent: %s is in %s, not shipped; only a shipped intent owes a fidelity audit", intentID, it.Bucket)
	}
	if !spec.HasNum(it.SpecID) {
		return AuditEmitResult{}, fmt.Errorf("intent: %s has no well-formed spec_id (%q); refusing to emit a review", intentID, it.SpecID)
	}
	return emitAuditWith(repoRoot, it, opts)
}

// writeAuditRequest writes the ephemeral review request markdown. The request is
// a prompt over the intent's Acceptance Criteria plus the receipt metadata; the
// host reads it, runs the reviewer, and produces the verdict JSON.
func writeAuditRequest(repoRoot string, it Intent, rcp, content string, opts AuditEmitOptions) error {
	if !rcpIDRe.MatchString(rcp) {
		return fmt.Errorf("intent: receipt id %q is malformed; refusing to build a request path", rcp)
	}
	realised, err := deliveredSpecs(repoRoot, it)
	if err != nil {
		return err
	}
	dir := filepath.Join(repoRoot, reviewsRelDir)
	if err := ensureRecordDir(repoRoot, reviewsRelDir); err != nil {
		return err
	}
	body := auditPromptBody(it, rcp, content, realised)
	doc := body + auditProvenanceBlock(auditPolicyFor(it, rcp, content, realised))
	if opts.RoutingSection != "" {
		doc += "\n## Routing\n\n" + opts.RoutingSection
	}

	path := filepath.Join(dir, rcp+".request.md")
	if err := fsutil.WriteFileAtomic(path, []byte(doc), 0o644); err != nil {
		return fmt.Errorf("intent: writing review request %s: %w", filepath.Join(reviewsRelDir, rcp+".request.md"), err)
	}
	return nil
}

// auditPromptBody composes the PROMPT the auditor is handed — everything in the
// request except the provenance block. It is a pure function of the receipt, the
// intent's path, the specs that realised it and its Acceptance Criteria, so the
// ingest can recompute it byte-for-byte and verify the echoed prompt_hash rather
// than trust it. Anything non-deterministic added here (a timestamp, a host path,
// a diff range the host resolved) breaks that, so it stays out.
//
// `realised` is the intent's whole delivery, not its scalar spec_id. An intent
// owns one or more specs (adr-2609151513118583), and the ship transition it is
// audited at arrives only once every one of them has closed — so the diff the
// auditor has to read spans all of them. Naming one spec asked for a fraction of
// the delivery while the criteria being judged describe the whole capability,
// which is a question no honest verdict can answer. It is a caller-supplied
// argument rather than a store read here so this function stays pure and the
// ingest recomputes the identical bytes.
func auditPromptBody(it Intent, rcp, content string, realised []string) string {
	ac := strings.TrimSpace(sectionBody(content, acHeadingRe))
	specs := strings.Join(realised, ", ")
	if specs == "" {
		specs = "(none recorded)"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# Fidelity review request — %s\n\n", rcp)
	fmt.Fprintf(&b, "- receipt_id: %s\n", rcp)
	fmt.Fprintf(&b, "- intent: %s\n", it.Path)
	fmt.Fprintf(&b, "- specs: %s\n", specs)
	fmt.Fprintf(&b, "- delivered: the diff/commit range that realised ALL of %s (host supplies the range)\n\n", specs)
	// The counts are the ingest's own: K is the bullet count validateVerdict
	// judges against, so the request never leaves the auditor to count
	// (iss-2609181121301638).
	switch k := countAcceptanceCriteria(content); k {
	case 0:
		b.WriteString("## Acceptance Criteria (authority; no bullets found)\n\n")
	default:
		fmt.Fprintf(&b, "## Acceptance Criteria (authority; %d %s, numbered ac-1..ac-%d in order)\n\n",
			k, plural(k, "criterion", "criteria"), k)
	}
	if ac == "" {
		b.WriteString("(none found)\n")
	} else {
		b.WriteString(ac + "\n")
	}
	writeScopeConditions(&b, content)
	b.WriteString("\n## Rubric (authority; the contract the ingest enforces)\n\n")
	b.WriteString(rubricText())
	b.WriteString("\n## Verdict shape (authority; the fields the ingest decodes, and no other)\n\n")
	b.WriteString(verdictShape(rcp))
	b.WriteString("\nRun the intent-auditor agent over the criteria, the scope conditions and\n")
	b.WriteString("the delivered diff; its verdict JSON takes the shape above. Ingest it with:\n\n")
	fmt.Fprintf(&b, "    abcd intent audit ingest --verdict-json <path>   # receipt %s\n", rcp)
	return b.String()
}

// writeScopeConditions renders the request's Scope Conditions block: every
// scope condition the intent carries, by the minted identity the verdict must
// dispose it under, with its text (iss-2609181121301638). The identities used to
// reach the auditor only as HTML comments in the record, which the request did
// not quote, so an auditor scraped them by hand and a miscount quarantined the
// verdict. They are read through ParseClaims, the reader the ingest's coverage
// check reads them through, so the set stated here is the set it enforces. An
// unstamped condition is listed as one, since the ingest refuses a verdict for
// an intent carrying it and the auditor should see why.
func writeScopeConditions(b *strings.Builder, content string) {
	conds := ParseClaims(content).Conditions
	if len(conds) == 0 {
		b.WriteString("\n## Scope Conditions (authority; none recorded, so scope_conditions is an empty list)\n\n")
		b.WriteString("(none recorded)\n")
		return
	}
	fmt.Fprintf(b, "\n## Scope Conditions (authority; %d %s, each disposed exactly once under its identity verbatim)\n\n",
		len(conds), plural(len(conds), "condition", "conditions"))
	for _, c := range conds {
		text := strings.Join(strings.Fields(c.Text), " ")
		if c.ID == "" {
			fmt.Fprintf(b, "- (condition %d carries no minted identity) — %s\n", c.Ordinal, text)
			continue
		}
		fmt.Fprintf(b, "- %s — %s\n", c.ID, text)
	}
}

// plural picks the noun form for a count.
func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// verdictShapeHints are the placeholders the stated shape shows for the fields
// that have one, keyed by JSON name; every other string shows `<string>`. The
// vocabularies come from the enum maps the validator consults, as the rubric's
// do, and the receipt is the one this request issued.
func verdictShapeHints(rcp string) map[string]string {
	return map[string]string{
		"_type":        VerdictType,
		"receipt_id":   rcp,
		"rubric_hash":  "sha256:<the Provenance block's rubric_hash>",
		"prompt_hash":  "sha256:<the Provenance block's prompt_hash>",
		"digest":       "sha256:<64 lowercase hex, or empty where not known>",
		"criterion_id": "ac-<N>",
		"verdict":      strings.Join(sortedKeys(verdictEnum), " | "),
		"condition_id": "cond-<an identity from Scope Conditions>",
		"disposition":  strings.Join(sortedKeys(dispositionEnum), " | "),
		"narrowing":    "<required on narrowed, empty otherwise>",
	}
}

// verdictShape renders the verdict the ingest decodes as one example object
// (iss-2609181121305984): the verdict struct itself — the type validateVerdict
// decodes into with DisallowUnknownFields — rendered by renderShape, so it is
// that schema rather than a copy of it. acceptance_rollup shows every
// acceptance verdict as a key.
func verdictShape(rcp string) string {
	return renderShape(reflect.TypeOf(verdict{}), shapeSpec{
		hints:   verdictShapeHints(rcp),
		mapKeys: map[string][]string{"acceptance_rollup": sortedKeys(verdictEnum)},
	})
}

// shapeSpec says what a stated JSON shape shows beyond the struct's own fields,
// each keyed by JSON name.
type shapeSpec struct {
	hints   map[string]string   // a string field's placeholder; any other shows <string>
	mapKeys map[string][]string // the keys a map field shows
	lens    map[string]int      // the elements a list shows; any other shows one
}

// renderShape renders the struct type t as one example object, indented as a
// markdown code block, for a request to state the shape its ingest decodes. It
// is built by reflection over the very struct the ingest decodes into, so a
// field added to or dropped from that struct moves the stated shape, and the
// prompt_hash with it: the request states the schema, never a copy of it. Every
// list shows at least one element so its members are named.
func renderShape(t reflect.Type, spec shapeSpec) string {
	v := reflect.New(t).Elem()
	fillShape(v, "", spec)
	var buf strings.Builder
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false) // the placeholders' angle brackets stay readable
	enc.SetIndent("    ", "  ")
	if err := enc.Encode(v.Interface()); err != nil {
		// The value is strings, ints, slices and string-keyed maps, which always
		// encode; the branch keeps the composition total.
		return "    (the shape could not be rendered: " + err.Error() + ")\n"
	}
	return "    " + buf.String()
}

// fillShape populates v with the placeholder for each field, by JSON name.
func fillShape(v reflect.Value, name string, spec shapeSpec) {
	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			tag, _, _ := strings.Cut(v.Type().Field(i).Tag.Get("json"), ",")
			fillShape(v.Field(i), tag, spec)
		}
	case reflect.String:
		if h, ok := spec.hints[name]; ok {
			v.SetString(h)
		} else {
			v.SetString("<string>")
		}
	case reflect.Slice:
		n := max(spec.lens[name], 1)
		s := reflect.MakeSlice(v.Type(), n, n)
		for i := 0; i < n; i++ {
			fillShape(s.Index(i), name, spec)
		}
		v.Set(s)
	case reflect.Map:
		m := reflect.MakeMap(v.Type())
		for _, k := range spec.mapKeys[name] {
			m.SetMapIndex(reflect.ValueOf(k), reflect.Zero(v.Type().Elem()))
		}
		v.Set(m)
	}
}

// ---------------------------------------------------------------------------
// Host-issued provenance (iss-2609100505140261)
// ---------------------------------------------------------------------------
//
// The two policy hashes are the attestation chain: which rubric and which prompt
// produced this verdict. They used to be required by the ingest and issued by
// nobody, so every auditor invented a value, the ingest checked only the SHAPE,
// and the Audit Notes gained a provenance claim that looked verified and was not.
//
// Both are now HOST-COMPUTED and DETERMINISTIC, which is what makes them
// checkable rather than merely well-formed:
//
//   - rubric_hash = sha256 over rubricText() — the judging contract this binary
//     ENFORCES, serialised from the very vocabularies and rules validateVerdict
//     applies. It is written verbatim into the request, so the auditor is handed
//     the exact bytes that were hashed, and it moves the moment the enforced
//     contract moves.
//   - prompt_hash = sha256 over auditPromptBody() — the request document the host
//     hands the auditor, excluding the provenance block itself (a block cannot
//     carry its own hash). The body is a pure function of the receipt, the
//     intent's path, its spec id and its Acceptance Criteria, all of which the
//     ingest holds, so the ingest RECOMPUTES the expected value instead of
//     trusting the echo.
//
// Both are therefore STALENESS-SENSITIVE by construction, and deliberately so.
// Editing the intent's Acceptance Criteria (or its path or spec id) between the
// emit and the ingest moves prompt_hash; upgrading the binary across a rubric
// change moves rubric_hash. Either refuses the verdict, because either means the
// verdict judged something other than what the receipt issued — the condition
// that used to pass silently. The remedy in both cases is to re-emit the request
// and re-run the audit, which the refusal names.
//
// A verdict echoing anything else is refused outright rather than dead-lettered:
// the DEAD_LETTER path is for a payload that IS this receipt's answer but is
// malformed, while a hash the host never issued says the verdict answers a
// different question. Refusing outright leaves the OWED marker parked, so
// re-emitting the request and re-auditing is still open; a DEAD_LETTER is
// terminal and could not be re-emitted.

// auditRubricID names the judging contract the hash is taken over. It is a
// version, not a checksum: bump it when the rubric's SHAPE changes, while the
// hash tracks its content automatically.
const auditRubricID = "abcd/intent-fidelity-rubric/v1"

// auditRubricRules is the canonical statement of what validateVerdict enforces.
// Every line names a check that actually runs below; nothing here is decorative,
// because the hash over it is what a stored Audit Note attests to. It is the ONE
// home for that statement — agents/intent-auditor.md quotes the rendered block
// out of the request rather than keeping its own copy.
var auditRubricRules = []string{
	"criteria: the intent's Acceptance Criteria bullets are the authority, numbered positionally ac-1..ac-K; every bullet is judged exactly once, and none is reordered, reworded, invented or dropped",
	"evidence: every criterion cites at least one evidence ref; a criterion with no citation is not MET",
	"gap audit: every honoured/diverged/missing claim cites at least one evidence ref",
	"dispositions: every scope-condition identity the intent carries is disposed exactly once, keyed to the minted identity and never to a paraphrase of the condition",
	"narrowing: required on the `narrowed` disposition, empty on every other one",
	"rollup: acceptance_rollup keys are acceptance verdicts and sum to the number of criteria",
}

// rubricText renders the rubric the hash is computed over. The two vocabularies
// come from the enum maps the validator itself consults (sorted, so the render is
// deterministic) rather than from a restatement of them: a word added to either
// map changes the rubric hash without anyone remembering to edit a string.
func rubricText() string {
	var b strings.Builder
	b.WriteString(auditRubricID + "\n")
	fmt.Fprintf(&b, "acceptance verdicts: %s\n", strings.Join(sortedKeys(verdictEnum), " | "))
	fmt.Fprintf(&b, "scope-condition dispositions: %s\n", strings.Join(sortedKeys(dispositionEnum), " | "))
	for _, r := range auditRubricRules {
		b.WriteString(r + "\n")
	}
	return b.String()
}

// sortedKeys is the deterministic render order for a set.
func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// sha256Field renders s as the `sha256:<64 lowercase hex>` shape sha256FieldRe
// validates — the one spelling every policy hash and attestation digest uses.
func sha256Field(s string) string {
	sum := sha256.Sum256([]byte(s))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// auditPolicy is the provenance the host issues for one receipt: the pair the
// auditor must echo and the ingest recomputes.
type auditPolicy struct {
	RubricHash string
	PromptHash string
}

// auditPolicyFor computes the host-issued provenance for one receipt. content is
// the intent file's bytes as the request was (or will be) composed from them, so
// emit and ingest agree as long as the record has not moved underneath the audit.
func auditPolicyFor(it Intent, rcp, content string, realised []string) auditPolicy {
	return auditPolicy{
		RubricHash: sha256Field(rubricText()),
		PromptHash: sha256Field(auditPromptBody(it, rcp, content, realised)),
	}
}

// deliveredSpecs lists every CLOSED spec realising the intent, in spec-number
// (minting) order — the delivery one fidelity audit has to read now that an
// intent owns one or more specs (adr-2609151513118583).
//
// Closed ones only: the audit runs at the ship transition, which by definition
// arrives when no OPEN spec names the intent, so an open spec in this list would
// mean the caller is auditing something that has not shipped. An intent whose
// store holds no closed spec at all — a record whose specs predate the store, or
// a re-emit in a tree that carries only the intent — falls back to its own
// scalar spec_id, so the request still names the delivery it can name rather
// than handing the auditor nothing.
//
// Both the emit and the ingest's prompt_hash recomputation read this, so the two
// agree; a spec closing between them would move the hash, and cannot, because no
// verb mints or closes a spec against an already-shipped intent.
func deliveredSpecs(repoRoot string, it Intent) ([]string, error) {
	store, err := spec.Load(repoRoot)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, sp := range store.SpecsForIntent(it.ID) {
		if sp.Status == spec.StatusClosed {
			out = append(out, sp.ID)
		}
	}
	if len(out) == 0 && spec.HasNum(it.SpecID) {
		out = []string{it.SpecID}
	}
	return out, nil
}

// auditProvenanceBlock renders the block appended to the request. It is NOT part
// of auditPromptBody: prompt_hash covers the prompt, and a block carrying that
// hash cannot be inside the bytes it hashes.
func auditProvenanceBlock(p auditPolicy) string {
	var b strings.Builder
	b.WriteString("\n## Provenance (host-computed — echo both verbatim into `policy`)\n\n")
	fmt.Fprintf(&b, "- rubric_hash: %s\n", p.RubricHash)
	fmt.Fprintf(&b, "- prompt_hash: %s\n", p.PromptHash)
	b.WriteString("\nDo not compute these yourself. `abcd intent audit ingest` recomputes both\n")
	b.WriteString("and refuses a verdict carrying any other value.\n")
	return b.String()
}

// ---------------------------------------------------------------------------
// Ingest (untrusted verdict -> committed Audit Notes)
// ---------------------------------------------------------------------------

// IngestVerdict reads an untrusted intent-fidelity verdict JSON and applies it to
// the committed record, FAIL-CLOSED at every step:
//
//   - malformed/oversize/unreadable payload with no resolvable receipt -> reject;
//   - receipt matching no parked marker (unsolicited) -> reject;
//   - already INGESTED for this receipt -> no-op when the payload renders to
//     the block on the record, a replacement in place when it renders
//     differently, and a refusal with nothing written when it does not
//     validate (see reingestVerdict);
//   - schema/semantic validation failure on a resolvable receipt -> DEAD_LETTER
//     (marker + INCONCLUSIVE criteria + retained raw payload), never partial;
//   - a rendered block citing a record id the repository's record gate
//     refuses -> reject, nothing written (checkReviewCitations);
//   - otherwise -> INGESTED (OWED stub replaced by the rendered verdict).
func IngestVerdict(repoRoot, verdictPath string) (IngestVerdictResult, error) {
	raw, err := readVerdictFile(verdictPath)
	if err != nil {
		return IngestVerdictResult{}, err
	}
	return IngestVerdictBytes(repoRoot, raw)
}

// ReadVerdict reads a verdict file the way IngestVerdict does (guarded, capped),
// for a front door that needs the payload itself as well as its ingest.
func ReadVerdict(verdictPath string) ([]byte, error) {
	return readVerdictFile(verdictPath)
}

// IngestVerdictBytes is IngestVerdict over a payload a front door has already
// read through ReadVerdict. The front door reads the verdict once and hands the
// same bytes to the ingest and to whatever else it reports from the payload
// (the receipt's model_reported), so the two can never describe different
// reads of a file that changed between them.
func IngestVerdictBytes(repoRoot string, raw []byte) (IngestVerdictResult, error) {

	// Lenient first pass: recover _type + receipt id so we can classify and
	// resolve the payload. A payload that is not a fidelity verdict at all, or that
	// we cannot even key on, has no home and is rejected outright (not dead-lettered).
	var lenient struct {
		Type      string `json:"_type"`
		ReceiptID string `json:"receipt_id"`
	}
	if err := json.Unmarshal(raw, &lenient); err != nil {
		return IngestVerdictResult{}, fmt.Errorf("intent: verdict is not parseable JSON; refusing to ingest: %w", err)
	}
	if lenient.Type != VerdictType {
		return IngestVerdictResult{}, fmt.Errorf("intent: verdict _type %q is not %q; refusing to ingest", lenient.Type, VerdictType)
	}
	if !rcpIDRe.MatchString(lenient.ReceiptID) {
		return IngestVerdictResult{}, fmt.Errorf("intent: verdict has no resolvable receipt_id (malformed or absent); refusing to ingest")
	}
	rcp := lenient.ReceiptID

	// The receipt's resolution, every check judged on the record, and the
	// write(s) are ONE critical section under the store's advisory lock, as every
	// other intent writer's are (iss-2609261935343851). Two ingests on one
	// intent, or an ingest beside a condition disposition, would otherwise each
	// write the bytes they read: the later write erases the earlier one and both
	// exit 0. Held here, the ingest reads the record another writer just left —
	// a verdict that landed first makes this a re-ingest, not a fresh one — and
	// the dead-letter's two writes land inside the same hold.
	//
	// Taking the lock creates the intent store, and a repository without one
	// holds no receipt to resolve: that refusal is made without the lock, so it
	// still writes nothing.
	if _, err := os.Lstat(filepath.Join(repoRoot, IntentsRelDir)); errors.Is(err, fs.ErrNotExist) {
		return ingestLocked(repoRoot, raw, rcp)
	}
	var res IngestVerdictResult
	err := withIntentMintLock(repoRoot, func() error {
		var err error
		res, err = ingestLocked(repoRoot, raw, rcp)
		return err
	})
	return res, err
}

// ingestLocked is IngestVerdictBytes's critical section, called under the
// intent store lock: it resolves rcp to its intent on the bytes read there and
// applies the verdict to those bytes. reingestVerdict and deadLetter are reached
// only from here, so they run under the same hold.
func ingestLocked(repoRoot string, raw []byte, rcp string) (IngestVerdictResult, error) {
	it, content, state, ok, err := findIntentByReceipt(repoRoot, rcp)
	if err != nil {
		return IngestVerdictResult{}, err
	}
	if !ok {
		return IngestVerdictResult{}, fmt.Errorf("intent: verdict receipt %s matches no parked review marker (unsolicited); refusing to ingest", rcp)
	}
	if state == "INGESTED" {
		return reingestVerdict(repoRoot, raw, it, rcp, content)
	}

	// The attestation chain must be the pair THIS receipt issued, not merely two
	// well-formed hashes. This is the check that was missing: presence and shape
	// were enforced and the VALUES were trusted, so a verdict could attest to a
	// rubric and a prompt nobody ever pinned and read back as verified.
	//
	// It sits here rather than in validateVerdict because the consequences differ.
	// A hash the host never issued says the verdict answers a different question,
	// which is the unsolicited-receipt shape: refused outright, OWED marker left
	// parked, so a re-emit and a re-audit are still open. An absent or malformed
	// hash is a malformed payload and keeps its existing DEAD_LETTER path below.
	if err := checkIssuedPolicy(repoRoot, raw, it, rcp, content); err != nil {
		return IngestVerdictResult{}, err
	}

	// The free-text renderer for this write, built ONCE and before anything is
	// composed. Both paths below persist agent-produced prose into a committed
	// record, so a degraded detector has to stop the write here rather than
	// halfway through the block it was about to render.
	free, err := newVerdictProse(repoRoot)
	if err != nil {
		return IngestVerdictResult{}, err
	}

	// Full schema + semantic validation. Any failure with a resolvable receipt
	// quarantines the payload rather than corrupting the record.
	v, verr := validateVerdict(raw, rcp, content)
	if verr != nil {
		return deadLetter(repoRoot, it, content, rcp, raw, verr.Error(), free)
	}

	rollup := countVerdicts(v)
	block := ingestedBlock(rcp, v, rollup, free)
	if err := checkReviewCitations(repoRoot, it, rcp, block); err != nil {
		return IngestVerdictResult{}, err
	}
	updated := upsertReviewBlock(content, rcp, block)
	if err := writeIntentFile(filepath.Join(repoRoot, it.Path), it.Path, updated); err != nil {
		return IngestVerdictResult{}, err
	}
	split := countDispositions(v)
	return IngestVerdictResult{
		Status: "ingested", ReceiptID: rcp, IntentID: it.ID, Criteria: len(v.Criteria),
		Met: rollup["MET"], MetWithConcern: rollup["MET_WITH_CONCERNS"],
		NotMet: rollup["NOT_MET"], Inconclusive: rollup["INCONCLUSIVE"],
		Conditions: len(v.ScopeConditions), Survived: split["survived"],
		Narrowed: split[dispositionNarrowed], Falsified: split["falsified"],
		Untested:                  split[dispositionUntested],
		ReadingOccasionedStanding: occasionedStanding(updated),
	}, nil
}

// reingestVerdict applies a verdict for a receipt already INGESTED. The receipt
// is the idempotency key, so a payload that renders to the block already on the
// record is a noop and writes nothing. A payload that renders differently
// replaces that block in place, after the same checks a first ingest makes:
// this is how an auditor who has since weighed a reading-occasioned condition
// block names its occasion and ingests again (the 2026-09-25 ruling in
// .abcd/work/DECISIONS.md). A payload that does not validate is refused with
// nothing written rather than dead-lettered: quarantine is for a receipt still
// owed a verdict, and a bad re-ingest must never replace a good one. It runs
// under the store lock ingestLocked holds.
func reingestVerdict(repoRoot string, raw []byte, it Intent, rcp, content string) (IngestVerdictResult, error) {
	free, err := newVerdictProse(repoRoot)
	if err != nil {
		return IngestVerdictResult{}, err
	}
	v, verr := validateVerdict(raw, rcp, content)
	if verr != nil {
		return IngestVerdictResult{}, fmt.Errorf("intent: receipt %s is already INGESTED and this verdict does not validate: %s; "+
			"an ingested verdict is replaced only by a valid one (nothing written)", rcp, free(verr.Error()))
	}
	rollup := countVerdicts(v)
	block := ingestedBlock(rcp, v, rollup, free)
	if existing, ok := reviewBlockText(content, rcp); ok && sameReviewBlock(existing, block, rcp) {
		return IngestVerdictResult{Status: "noop", ReceiptID: rcp, IntentID: it.ID}, nil
	}
	if err := checkIssuedPolicy(repoRoot, raw, it, rcp, content); err != nil {
		return IngestVerdictResult{}, err
	}
	if err := checkReviewCitations(repoRoot, it, rcp, block); err != nil {
		return IngestVerdictResult{}, err
	}
	updated := upsertReviewBlock(content, rcp, block)
	if err := writeIntentFile(filepath.Join(repoRoot, it.Path), it.Path, updated); err != nil {
		return IngestVerdictResult{}, err
	}
	split := countDispositions(v)
	return IngestVerdictResult{
		Status: "ingested", Replaced: true, ReceiptID: rcp, IntentID: it.ID, Criteria: len(v.Criteria),
		Met: rollup["MET"], MetWithConcern: rollup["MET_WITH_CONCERNS"],
		NotMet: rollup["NOT_MET"], Inconclusive: rollup["INCONCLUSIVE"],
		Conditions: len(v.ScopeConditions), Survived: split["survived"],
		Narrowed: split[dispositionNarrowed], Falsified: split["falsified"],
		Untested:                  split[dispositionUntested],
		ReadingOccasionedStanding: occasionedStanding(updated),
	}, nil
}

// checkIssuedPolicy compares the verdict's policy hashes against the pair the
// host issued for this receipt, and returns a refusal naming what each hash is
// computed over when they disagree.
//
// It deliberately passes on an ABSENT or MALFORMED hash: that is the malformed-
// payload class validateVerdict already quarantines with its own message, and
// duplicating the judgement here would move an established DEAD_LETTER onto the
// reject path. Only a well-shaped hash that is not ours is refused outright.
func checkIssuedPolicy(repoRoot string, raw []byte, it Intent, rcp, content string) error {
	var lenient struct {
		Policy verdictPolicy `json:"policy"`
	}
	if err := json.Unmarshal(raw, &lenient); err != nil {
		return nil // the strict decode below reports an unparseable payload.
	}
	got := lenient.Policy
	if !sha256FieldRe.MatchString(got.RubricHash) || !sha256FieldRe.MatchString(got.PromptHash) {
		return nil
	}
	realised, err := deliveredSpecs(repoRoot, it)
	if err != nil {
		return err
	}
	want := auditPolicyFor(it, rcp, content, realised)
	if got.RubricHash == want.RubricHash && got.PromptHash == want.PromptHash {
		return nil
	}
	// Both values are sha256-shaped by the guard above, so quoting them back
	// cannot carry payload prose into the message.
	return fmt.Errorf("intent: verdict %s carries policy hashes this receipt never issued; refusing to ingest.\n"+
		"  rubric_hash: got %s, issued %s\n"+
		"  prompt_hash: got %s, issued %s\n"+
		"The host computes both and writes them into the request's Provenance block: rubric_hash is sha256 over the "+
		"rubric the request states, prompt_hash is sha256 over the request's prompt body (everything above that block). "+
		"Re-emit with `abcd intent audit %s` and echo the two values it writes, rather than computing a hash yourself.",
		rcp, got.RubricHash, want.RubricHash, got.PromptHash, want.PromptHash, it.ID)
}

// readVerdictFile reads the untrusted verdict payload behind fsutil.ReadGuarded
// (O_NOFOLLOW + regular-file on the open fd + size cap, in one call). The single
// guarded open is the only race-free form: an Lstat-then-ReadFile pair leaves a
// window in which a symlink swapped in after the Lstat is followed by ReadFile
// (which also ignores the pre-checked size), so this joins the shared operand
// primitive rather than re-checking by hand (mirrors cli.readGuardedOperand).
func readVerdictFile(path string) ([]byte, error) {
	data, err := fsutil.ReadGuarded(path, maxVerdictBytes)
	if err != nil {
		switch {
		case errors.Is(err, fsutil.ErrNotRegular) || errors.Is(err, syscall.ELOOP):
			return nil, fmt.Errorf("intent: verdict %s is not a regular file (a symlink or non-regular operand is refused)", path)
		case errors.Is(err, fsutil.ErrTooBig):
			return nil, fmt.Errorf("intent: verdict %s exceeds the %d-byte cap", path, maxVerdictBytes)
		default:
			return nil, fmt.Errorf("intent: reading verdict %s: %w", path, err)
		}
	}
	return data, nil
}

// validateVerdict parses and fully validates the payload against the reviewer
// contract and the intent's actual Acceptance Criteria. It returns a non-nil
// error describing the first violation (the DEAD_LETTER reason).
func validateVerdict(raw []byte, rcp, intentContent string) (verdict, error) {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields() // reject smuggled extra fields
	var v verdict
	if err := dec.Decode(&v); err != nil {
		return verdict{}, fmt.Errorf("malformed verdict JSON: %v", err)
	}
	if v.Type != VerdictType {
		return verdict{}, fmt.Errorf("wrong _type %q (want %q)", v.Type, VerdictType)
	}
	if v.ReceiptID != rcp {
		return verdict{}, fmt.Errorf("receipt_id %q disagrees with the resolved receipt %q", v.ReceiptID, rcp)
	}
	// The attestation chain (rubric + prompt the host pinned) is what makes this a
	// VSA-shaped verdict rather than an unverifiable opinion; require both.
	if strings.TrimSpace(v.Policy.RubricHash) == "" || strings.TrimSpace(v.Policy.PromptHash) == "" {
		return verdict{}, fmt.Errorf("policy.rubric_hash and policy.prompt_hash are both required")
	}
	// And both must be the shape the contract publishes. Presence alone left the
	// two fields as free text under a structural name: the block renders them
	// unredacted on the stated ground that a hash is a validated shape, and
	// nothing was validating them.
	for _, h := range [][2]string{
		{"policy.rubric_hash", v.Policy.RubricHash},
		{"policy.prompt_hash", v.Policy.PromptHash},
	} {
		if !sha256FieldRe.MatchString(h[1]) {
			return verdict{}, fmt.Errorf("%s %q is not a sha256 digest (want sha256:<64 lowercase hex>)", h[0], h[1])
		}
	}
	// An attestation digest is `sha256:<if-known>` in the contract, so ABSENCE is
	// legitimate and a present-but-wrong shape is not. `kind` and `ref` carry no
	// declared shape — a real ref is a commit range with prose beside it — so
	// they are free text and are redacted at render rather than validated here.
	for i, a := range v.InputAttestations {
		if strings.TrimSpace(a.Digest) == "" {
			continue
		}
		if !sha256FieldRe.MatchString(a.Digest) {
			return verdict{}, fmt.Errorf("input_attestations[%d].digest %q is not a sha256 digest "+
				"(want sha256:<64 lowercase hex>, or empty where the digest is not known)", i, a.Digest)
		}
	}
	if len(v.Criteria) == 0 {
		return verdict{}, fmt.Errorf("criteria is empty")
	}

	k := countAcceptanceCriteria(intentContent)
	if k == 0 {
		return verdict{}, fmt.Errorf("intent has no parseable Acceptance Criteria bullets to judge")
	}
	seen := map[int]bool{}
	for i, c := range v.Criteria {
		m := criterionIDRe.FindStringSubmatch(c.CriterionID)
		if m == nil {
			return verdict{}, fmt.Errorf("criterion[%d] id %q is not ac-N", i, c.CriterionID)
		}
		n := atoiPositive(m[1])
		if n < 1 || n > k {
			return verdict{}, fmt.Errorf("criterion %q is out of range (intent has ac-1..ac-%d)", c.CriterionID, k)
		}
		// Dedup on the resolved index, so ac-1 and a zero-padded ac-01 (which map to
		// the same bullet) cannot both be applied.
		if seen[n] {
			return verdict{}, fmt.Errorf("criterion %q targets an already-judged Acceptance-Criteria bullet (ac-%d)", c.CriterionID, n)
		}
		seen[n] = true
		if !verdictEnum[c.Verdict] {
			return verdict{}, fmt.Errorf("criterion %q has out-of-enum verdict %q", c.CriterionID, c.Verdict)
		}
		if !hasCitedEvidence(c.Evidence) {
			return verdict{}, fmt.Errorf("criterion %q cites no evidence ref", c.CriterionID)
		}
	}
	// Every criterion must be judged: a verdict covering only some of ac-1..ac-K is
	// a PARTIAL judgement. Accepting it would write an incomplete INGESTED state and
	// let the idempotency short-circuit drop a later complete verdict — fail closed.
	if len(seen) != k {
		return verdict{}, fmt.Errorf("verdict judges %d of %d Acceptance-Criteria bullets (every ac-1..ac-%d must be judged exactly once)", len(seen), k, k)
	}

	// Rollup counts must sum to the number of criteria (reviewer contract rule 4).
	sum := 0
	for key, n := range v.AcceptanceRollup {
		if !verdictEnum[key] {
			return verdict{}, fmt.Errorf("acceptance_rollup has non-verdict key %q", key)
		}
		sum += n
	}
	if sum != len(v.Criteria) {
		return verdict{}, fmt.Errorf("acceptance_rollup sums to %d, not the %d criteria", sum, len(v.Criteria))
	}

	for _, bucket := range [][2]any{{"honoured", v.GapAudit.Honoured}, {"diverged", v.GapAudit.Diverged}, {"missing", v.GapAudit.Missing}} {
		name := bucket[0].(string)
		for i, e := range bucket[1].([]verdictGapEntry) {
			if !hasCitedEvidence(e.Evidence) {
				return verdict{}, fmt.Errorf("gap_audit.%s[%d] cites no evidence ref", name, i)
			}
		}
	}

	if err := validateConditionDispositions(v, intentContent); err != nil {
		return verdict{}, err
	}
	return v, nil
}

// validateConditionDispositions checks the scope-condition dispositions against
// the identities the RECORD carries, never against the payload's own claims —
// the conditions are read through ParseClaims (spc-55's single claim reader), so
// no second parser can disagree with the readiness gate about what a condition
// is.
//
// The two directions are separate refusals: a verdict disposing a condition the
// intent does not record is judging something the record does not claim, and a
// verdict disposing only some of them is the partial judgement the criteria
// check already refuses one level down. That symmetry is what makes the staged
// rollout safe — an intent shipped before the identity mint existed carries no
// conditions, so the check is vacuous rather than blocking.
func validateConditionDispositions(v verdict, intentContent string) error {
	conds := ParseClaims(intentContent).Conditions
	known := map[string]bool{}
	for _, c := range conds {
		// An unstamped condition has no identity for a disposition to attach to,
		// so accepting the verdict would leave it permanently undisposed — which
		// is exactly the absence itd-181 refuses. The readiness gate reports the
		// same fault, but it only reports: it is read-only, and it refuses a
		// shipped bucket outright — which is the only bucket the ingest ever
		// sees. So this is the gate, not a second opinion.
		if c.ID == "" {
			return fmt.Errorf("scope condition %d carries no minted identity, so no disposition can be keyed to it", c.Ordinal)
		}
		known[c.ID] = true
	}
	// Two bullets sharing one identity collapse into a single entry in `known`,
	// so the set-sized coverage check below would accept one disposition for two
	// conditions and leave the second silently undisposed. A copy-pasted bullet
	// keeps its invisible marker and nothing re-stamps a shipped record, so the
	// state is reachable. DuplicateConditionIDs is the canonical detector — the
	// same one the readiness gate reports with — never a second notion of it.
	if dupes := DuplicateConditionIDs(conds); len(dupes) > 0 {
		return fmt.Errorf("scope condition identity %q is carried by more than one condition, so a disposition cannot be keyed to either", dupes[0])
	}
	if len(known) == 0 {
		if len(v.ScopeConditions) != 0 {
			return fmt.Errorf("verdict disposes %d scope condition(s) but the intent records none", len(v.ScopeConditions))
		}
		return nil
	}

	seen := map[string]bool{}
	for i, c := range v.ScopeConditions {
		if !known[c.ConditionID] {
			return fmt.Errorf("scope_conditions[%d] id %q is not an identity the intent carries", i, c.ConditionID)
		}
		if seen[c.ConditionID] {
			return fmt.Errorf("scope condition %q is disposed more than once", c.ConditionID)
		}
		seen[c.ConditionID] = true
		if !dispositionEnum[c.Disposition] {
			return fmt.Errorf("scope condition %q has out-of-enum disposition %q", c.ConditionID, c.Disposition)
		}
		// `narrowing` is required on `narrowed` and empty everywhere else — the
		// rule the definition publishes, gated in both directions. A narrowing
		// carried by a `survived` condition renders a stated narrowing into the
		// record while the split reports no narrowed condition at all.
		if c.Disposition == dispositionNarrowed && strings.TrimSpace(c.Narrowing) == "" {
			return fmt.Errorf("scope condition %q is narrowed but states no narrowing", c.ConditionID)
		}
		if c.Disposition != dispositionNarrowed && strings.TrimSpace(c.Narrowing) != "" {
			return fmt.Errorf("scope condition %q is %s but states a narrowing; only a narrowed condition carries one", c.ConditionID, c.Disposition)
		}
		// `untested` is by definition the absence of evidence; every other
		// disposition is a claim about delivered reality and must cite one.
		if c.Disposition != dispositionUntested && !hasCitedEvidence(c.Evidence) {
			return fmt.Errorf("scope condition %q cites no evidence ref", c.ConditionID)
		}
	}
	if len(seen) != len(known) {
		return fmt.Errorf("verdict disposes %d of %d scope conditions (every condition must be disposed exactly once)", len(seen), len(known))
	}
	return nil
}

// deadLetter quarantines a bad-but-resolvable verdict: it retains the raw payload
// under the ephemeral reviews dir and replaces the parked marker with a
// DEAD_LETTER block recording all criteria INCONCLUSIVE. Never partial. It runs
// under the store lock ingestLocked holds, so its two writes — the retained
// payload and the record — land in one hold.
func deadLetter(repoRoot string, it Intent, content, rcp string, raw []byte, reason string, free proseField) (IngestVerdictResult, error) {
	if !rcpIDRe.MatchString(rcp) {
		return IngestVerdictResult{}, fmt.Errorf("intent: receipt id %q is malformed; refusing to dead-letter", rcp)
	}
	dir := filepath.Join(repoRoot, reviewsRelDir)
	dlRel := filepath.Join(reviewsRelDir, rcp+".deadletter.json")
	untested := untestedDispositions(content)
	block := deadLetterBlock(rcp, reason, dlRel, untested, free)
	// The quarantine's reason quotes the payload, so it is held to the same gate
	// as a verdict, before anything is retained or written.
	if err := checkReviewCitations(repoRoot, it, rcp, block); err != nil {
		return IngestVerdictResult{}, err
	}
	if err := ensureRecordDir(repoRoot, reviewsRelDir); err != nil {
		return IngestVerdictResult{}, err
	}
	if err := fsutil.WriteFileAtomic(filepath.Join(dir, rcp+".deadletter.json"), raw, 0o644); err != nil {
		return IngestVerdictResult{}, fmt.Errorf("intent: retaining dead-letter payload %s: %w", dlRel, err)
	}
	updated := upsertReviewBlock(content, rcp, block)
	if err := writeIntentFile(filepath.Join(repoRoot, it.Path), it.Path, updated); err != nil {
		return IngestVerdictResult{}, err
	}
	// The counts exist so a surface reports the split WITHOUT re-reading the
	// record, so they must agree with what was just written there: the quarantine
	// records every condition untested, and says so here too.
	return IngestVerdictResult{
		Status: "dead_letter", ReceiptID: rcp, IntentID: it.ID,
		Conditions: len(untested), Untested: len(untested),
		DeadLetterPath: dlRel, Reason: reason,
		ReadingOccasionedStanding: occasionedStanding(updated),
	}, nil
}

// UnresolvedCitation is one record id a fragment cites that the repository's
// record gate refuses, at its 1-based line in the fragment.
type UnresolvedCitation struct {
	Line int
	ID   string
}

// proseCitationGate is record-lint's prose_citation_resolves asked of a
// fragment before it is written into the record at rel: the ids it cites that
// the gate would refuse there. It is lint.UnresolvedProseCitationsInRecord,
// registered by the front doors (SetProseCitationGate), because this package
// cannot import core/lint: lint's own tests import this package, and Go refuses
// the cycle. The same seam lint.SetIssueReader is, from the other side.
var proseCitationGate func(repoRoot, rel, text string) ([]UnresolvedCitation, error)

// SetProseCitationGate registers the prose-citation gate every verdict ingest
// asks before it writes. Pass an adapter over lint.UnresolvedProseCitationsInRecord.
func SetProseCitationGate(fn func(repoRoot, rel, text string) ([]UnresolvedCitation, error)) {
	proseCitationGate = fn
}

// checkReviewCitations refuses a rendered review block that would cite a record
// id naming no record, in a repository whose record-lint gates prose citations
// in the intent store (iss-2609231036448320). The ingest validated the verdict
// against the rubric; this holds the block to the gate the record it lands in
// must pass, so a valid verdict can never produce an uncommittable record.
//
// It REFUSES rather than sanitises, and nothing is written. The verdict schema
// has no way to say an id is illustrative, and the cleaner every ingest routes
// prose through (termsafe) neutralises syntax, never a citation's meaning: a
// rewrite that dropped every id would erase the real citations an audit rests
// on, and one that dropped only the unresolvable ones would be a second
// sanitiser deciding what an auditor meant. The auditor re-words the prose to
// describe the record instead of citing an id that does not exist, and ingests
// again; the receipt stays in the state it was in.
//
// The gate is the repository's: a repository whose record-lint does not arm the
// rule over the intent store refuses nothing here. A front door that registered
// no gate is refused outright, since the check it owes cannot be made.
func checkReviewCitations(repoRoot string, it Intent, rcp, block string) error {
	if proseCitationGate == nil {
		return fmt.Errorf("intent: no prose-citation gate is registered, so the verdict for %s cannot be checked "+
			"against the record gate the intent record must pass; refusing to ingest (nothing written)", rcp)
	}
	rel := filepath.ToSlash(it.Path)
	cites, err := proseCitationGate(repoRoot, rel, block)
	if err != nil {
		return fmt.Errorf("intent: the prose-citation check over the rendered review block: %w", err)
	}
	if len(cites) == 0 {
		return nil
	}
	lines := strings.Split(block, "\n")
	named := make([]string, 0, len(cites))
	for _, c := range cites {
		at := ""
		if c.Line >= 1 && c.Line <= len(lines) {
			at = ": " + excerpt(lines[c.Line-1])
		}
		named = append(named, fmt.Sprintf("%s (line %d of the rendered block%s)", c.ID, c.Line, at))
	}
	return fmt.Errorf("intent: verdict %s cites a record id that names no record in this repository: %s; "+
		"record-lint's prose_citation_resolves refuses %s if it carries one, and a verdict has no way to mark an id "+
		"illustrative. Re-word the prose to describe the record rather than cite an id that does not exist, and "+
		"ingest again (nothing written)", rcp, strings.Join(named, "; "), rel)
}

// excerpt shortens one rendered line for a refusal. The line is already cleaned
// for the record, so only its length needs bounding.
func excerpt(s string) string {
	const limit = 120
	r := []rune(s)
	if len(r) <= limit {
		return s
	}
	return string(r[:limit]) + "…"
}

// occasionedStanding lists the condition-block dispositions the fold reports as
// standing in content, ordered by condition identity so the report is
// deterministic.
func occasionedStanding(content string) []condition.Disposition {
	standing := condition.Standing(content)
	ids := make([]string, 0, len(standing))
	for id, d := range standing {
		if d.Occasion != "" {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	out := make([]condition.Disposition, 0, len(ids))
	for _, id := range ids {
		out = append(out, standing[id])
	}
	return out
}

// ---------------------------------------------------------------------------
// Receipt resolution + Audit Notes surgery
// ---------------------------------------------------------------------------

// findIntentByReceipt scans every bucket for the intent whose Audit Notes carry a
// review marker for rcp. It returns the intent, its content, and the marker state
// (OWED/INGESTED/DEAD_LETTER). ok is false when no intent claims the receipt.
func findIntentByReceipt(repoRoot, rcp string) (Intent, string, string, bool, error) {
	corpus, err := Load(repoRoot)
	if err != nil {
		return Intent{}, "", "", false, err
	}
	for _, it := range corpus.Intents {
		data, err := readRepoFile(filepath.Join(repoRoot, it.Path), it.Path)
		if err != nil {
			return Intent{}, "", "", false, err
		}
		content := string(data)
		if state, ok := markerState(content, rcp); ok {
			return it, content, state, true, nil
		}
	}
	return Intent{}, "", "", false, nil
}

// reviewBlock is one review block as the record carries it: its marker's state
// and receipt, and the [start, end) lines it spans.
type reviewBlock struct {
	state, receipt string
	start, end     int
}

// readReviewBlocks is the ONE reader of an intent record's review blocks, in
// document order. Every question the audit asks of the record — which receipt
// it parked, what state a receipt is in, which lines a replacement rewrites —
// is answered from here, so the reader and the writer cannot disagree about
// where a block is.
//
// A marker counts only on a LIVE line of the live `## Audit Notes` section, as
// condition.AuditNotes reads it: a marker-shaped line in a fenced block, in an
// HTML comment span or under another heading is an example, not state, so a
// record quoting a marker cannot solicit, misroute or silence a verdict
// (iss-2609020529185438). termsafe's cleaner breaks the comment delimiters in
// every field the ingest writes; this reader is the other half, reading the
// record as a markdown reader parses it.
//
// A block's extent is reviewBlockEnd's.
func readReviewBlocks(content string) (lines []string, blocks []reviewBlock) {
	lines, mask, start, end, ok := condition.AuditNotes(content)
	if !ok {
		return lines, nil
	}
	for i := start; i < end; i++ {
		if mask[i] != 0 {
			continue
		}
		m := markerRe.FindStringSubmatch(strings.TrimRight(lines[i], "\r"))
		if m == nil {
			continue
		}
		blocks = append(blocks, reviewBlock{state: m[1], receipt: m[2], start: i,
			end: reviewBlockEnd(lines, mask, i, end, m[1], m[2])})
	}
	return lines, blocks
}

// reviewBlockEnd bounds the block whose marker is at line start, within a
// section body ending at sectionEnd:
//
//   - A block closes on its own closing line (condition.ReviewEndRe naming its
//     receipt), which every renderer writes; the block ends after it, so prose a
//     human writes below the block is never part of it (iss-2609251451434656).
//   - A block written before the closing line existed has none. An OWED stub
//     has one known shape — the marker and its one sentence — and ends there,
//     so a first ingest replaces the stub and nothing below it
//     (iss-2609251451432601).
//   - Any other block without one runs to the next live block marker of either
//     grammar, closing line or heading, as it always has; a condition block
//     written after it survives its replacement (spc-2609020626046252). A
//     trailing run of link-reference definitions is not the block's: no
//     renderer writes one, and a record parks them at the end of the section
//     (iss-2608210737265820), so a replacement leaves them where they are.
func reviewBlockEnd(lines []string, mask []uint8, start, sectionEnd int, state, rcp string) int {
	live := func(j int) (string, bool) {
		if mask[j] != 0 {
			return "", false
		}
		return strings.TrimRight(lines[j], "\r"), true
	}
	for j := start + 1; j < sectionEnd; j++ {
		t, ok := live(j)
		if !ok {
			continue
		}
		if m := condition.ReviewEndRe.FindStringSubmatch(t); m != nil {
			if m[1] == rcp {
				return j + 1
			}
			break
		}
		if condition.IsBlockMarker(t) {
			break
		}
	}
	if state == "OWED" && start+1 < sectionEnd {
		if t, ok := live(start + 1); ok && strings.HasPrefix(t, "Fidelity review OWED") {
			return start + 2
		}
	}
	end := sectionEnd
	for j := start + 1; j < sectionEnd; j++ {
		if t, ok := live(j); ok && (condition.IsBlockMarker(t) || condition.ReviewEndRe.MatchString(t)) {
			end = j
			break
		}
	}
	body := append([]string(nil), lines[start+1:end]...)
	if refs := mdrecord.PeelTrailingLinkRefs(&body); len(refs) > 0 {
		end = start + 1 + len(body)
	}
	return end
}

// reviewBlockFor returns the block for rcp, if the record carries one.
func reviewBlockFor(content, rcp string) ([]string, reviewBlock, bool) {
	lines, blocks := readReviewBlocks(content)
	for _, b := range blocks {
		if b.receipt == rcp {
			return lines, b, true
		}
	}
	return lines, reviewBlock{}, false
}

// existingMarker returns the receipt id and state of the FIRST review marker in
// content, if any. Emit reuses this parked receipt rather than recomputing one
// (see emitAuditForIntent's receipt-shift note).
func existingMarker(content string) (string, string, bool) {
	if _, blocks := readReviewBlocks(content); len(blocks) > 0 {
		return blocks[0].receipt, blocks[0].state, true
	}
	return "", "", false
}

// markerState returns the state of the review marker for rcp, if present.
func markerState(content, rcp string) (string, bool) {
	_, b, ok := reviewBlockFor(content, rcp)
	return b.state, ok
}

// upsertReviewBlock replaces the existing review block for rcp with newBlock, or
// appends newBlock to the Audit Notes section (creating the section if absent).
// The block's extent is readReviewBlocks'. The record it returns ends in a
// newline, whatever the record it was handed ended in (iss-2609231011136579).
func upsertReviewBlock(content, rcp, newBlock string) string {
	lines, b, ok := reviewBlockFor(content, rcp)
	if !ok {
		return appendToAuditNotes(content, newBlock)
	}
	// Keep the blank separator the old block ended with, so a block that
	// follows it is not glued to the replacement.
	sep := b.end
	for sep > b.start+1 && strings.TrimSpace(lines[sep-1]) == "" {
		sep--
	}
	out := make([]string, 0, len(lines))
	out = append(out, lines[:b.start]...)
	out = append(out, strings.Split(newBlock, "\n")...)
	out = append(out, lines[sep:]...)
	return withFinalNewline(strings.Join(out, "\n"))
}

// reviewBlockText is the review block for rcp as a renderer would have written
// it: its lines with the trailing blank separator trimmed.
func reviewBlockText(content, rcp string) (string, bool) {
	lines, b, ok := reviewBlockFor(content, rcp)
	if !ok {
		return "", false
	}
	return strings.TrimRight(strings.Join(lines[b.start:b.end], "\n"), "\r\n\t "), true
}

// sameReviewBlock reports whether the block on the record already says what
// rendered says. A block written before the closing line existed carries none,
// and says the same when it is rendered without it: an identical re-ingest over
// it is the noop it always was, not a rewrite that adds the line.
func sameReviewBlock(existing, rendered, rcp string) bool {
	if existing == rendered {
		return true
	}
	return !condition.ReviewEndRe.MatchString(lastLine(existing)) &&
		existing == strings.TrimSuffix(rendered, "\n"+condition.ReviewEndLine(rcp))
}

// lastLine is the final line of s.
func lastLine(s string) string {
	if i := strings.LastIndex(s, "\n"); i >= 0 {
		return s[i+1:]
	}
	return s
}

// withFinalNewline ends s in a newline, as every record writer in the tree ends
// its file.
func withFinalNewline(s string) string {
	if strings.HasSuffix(s, "\n") {
		return s
	}
	return s + "\n"
}

// appendToAuditNotes appends a block to the `## Audit Notes` section, creating
// the section at end of file if it is absent.
func appendToAuditNotes(content, block string) string {
	// The section is the live one readReviewBlocks reads: a fenced or commented
	// `## Audit Notes` is an example, and a block appended under it is a block
	// the reader never finds again.
	lines, _, bodyStart, end, ok := condition.AuditNotes(content)
	if !ok {
		body := strings.TrimRight(content, "\n")
		return body + "\n\n## Audit Notes\n\n" + block + "\n"
	}
	head := bodyStart - 1
	// Copy the section out (never alias the backing array) and drop the template
	// placeholder line, so the first real review block replaces the "Empty" claim
	// rather than sitting beneath it.
	section := make([]string, 0, end-head)
	for _, ln := range lines[head+1 : end] {
		if auditPlaceholderRe.MatchString(strings.TrimRight(ln, "\r")) {
			continue
		}
		section = append(section, ln)
	}
	// Drop blank lines at both ends of the section, then re-add one separator on
	// each side: the heading's blank line is written below, so a leading one kept
	// here would open the section with two.
	for len(section) > 0 && strings.TrimSpace(section[0]) == "" {
		section = section[1:]
	}
	for len(section) > 0 && strings.TrimSpace(section[len(section)-1]) == "" {
		section = section[:len(section)-1]
	}
	// Peel a trailing run of link-reference definitions (and any blanks among them)
	// off the section so the new block is inserted ABOVE them: a `[ref]: url`
	// definition parked at the end of the Audit Notes belongs below the review
	// prose, and appending the block after it detaches the block from the section
	// it documents (iss-2608210737265820).
	trailingRefs := mdrecord.PeelTrailingLinkRefs(&section)
	rebuilt := make([]string, 0, len(lines)+8)
	rebuilt = append(rebuilt, lines[:head+1]...)
	rebuilt = append(rebuilt, "")
	rebuilt = append(rebuilt, section...)
	if len(section) > 0 {
		rebuilt = append(rebuilt, "")
	}
	rebuilt = append(rebuilt, strings.Split(block, "\n")...)
	if len(trailingRefs) > 0 {
		rebuilt = append(rebuilt, "")
		rebuilt = append(rebuilt, trailingRefs...)
	}
	rebuilt = append(rebuilt, "")
	rebuilt = append(rebuilt, lines[end:]...)
	return withFinalNewline(strings.Join(rebuilt, "\n"))
}

// ---------------------------------------------------------------------------
// Block rendering (deterministic; no timestamps)
// ---------------------------------------------------------------------------

func owedBlock(rcp string) string {
	return fmt.Sprintf("<!-- abcd-review: OWED receipt=%s -->\nFidelity review OWED (receipt %s).\n%s",
		rcp, rcp, condition.ReviewEndLine(rcp))
}

// deadLetterBlock renders the quarantine block. conds are the record's own scope
// conditions, every one of them recorded `untested`: the acceptance vocabulary
// already says INCONCLUSIVE here, and the disposition vocabulary's word for the
// same state is `untested`, so the quarantine stays honest in both without
// inventing a fifth value.
func deadLetterBlock(rcp, reason, dlRel string, conds []verdictCondition, free proseField) string {
	var b strings.Builder
	// reason is derived from untrusted payload content (e.g. an out-of-enum token
	// quoted back), so it is free text and goes through the same redact-then-
	// neutralise path as the rest: a quarantine that leaks is still a leak.
	fmt.Fprintf(&b, "<!-- abcd-review: DEAD_LETTER receipt=%s -->\n"+
		"Fidelity review DEAD_LETTER (receipt %s): %s. Raw payload retained at %s. "+
		"All criteria recorded INCONCLUSIVE.\n", rcp, rcp, free(reason), dlRel)
	renderDispositions(&b, conds, free)
	return closeReviewBlock(&b, rcp)
}

// closeReviewBlock ends a rendered block on its closing line, the line
// readReviewBlocks bounds it by.
func closeReviewBlock(b *strings.Builder, rcp string) string {
	return strings.TrimRight(b.String(), "\n") + "\n" + condition.ReviewEndLine(rcp)
}

// untestedDispositions is every identified scope condition the record carries,
// recorded `untested`. A condition with no minted identity is skipped: there is
// nothing to key a disposition on, and the ingest refuses such a record anyway.
func untestedDispositions(intentContent string) []verdictCondition {
	conds := ParseClaims(intentContent).Conditions
	out := make([]verdictCondition, 0, len(conds))
	for _, c := range conds {
		if c.ID == "" {
			continue
		}
		out = append(out, verdictCondition{ConditionID: c.ID, Disposition: dispositionUntested})
	}
	return out
}

func ingestedBlock(rcp string, v verdict, rollup map[string]int, free proseField) string {
	var b strings.Builder
	fmt.Fprintf(&b, "<!-- abcd-review: INGESTED receipt=%s -->\n", rcp)
	fmt.Fprintf(&b, "Fidelity review — receipt %s (verifier %s %s).\n\n",
		rcp, orFree(v.Verifier.ID, free), orFree(v.Verifier.Version, free))
	// Pinned provenance: the verifier identity, the policy hashes it attested to,
	// and every input attestation. All of it is untrusted, and the split is by
	// whether the contract gives the field a SHAPE. The two hashes and a digest
	// are validated `sha256:<64 hex>` by the time this runs, so they carry
	// nothing to redact and keep oneLine alone. The verifier identity, its
	// version, and an attestation's kind and ref have no declared shape — a real
	// ref is a commit range with prose beside it — so they are free text and go
	// through the same redact-then-neutralise path the rationales use.
	fmt.Fprintf(&b, "Provenance: %s@%s · rubric_hash %s · prompt_hash %s\n",
		orFree(v.Verifier.ID, free), orFree(v.Verifier.Version, free),
		orDash(v.Policy.RubricHash), orDash(v.Policy.PromptHash))
	if len(v.InputAttestations) > 0 {
		b.WriteString("Input attestations:")
		for _, a := range v.InputAttestations {
			fmt.Fprintf(&b, " %s:%s@%s;", orFree(a.Kind, free), orFree(a.Ref, free), orDash(a.Digest))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	fmt.Fprintf(&b, "Acceptance rollup: MET %d · MET_WITH_CONCERNS %d · NOT_MET %d · INCONCLUSIVE %d\n\n",
		rollup["MET"], rollup["MET_WITH_CONCERNS"], rollup["NOT_MET"], rollup["INCONCLUSIVE"])

	b.WriteString("Per-criterion verdicts:\n")
	for _, c := range v.Criteria {
		fmt.Fprintf(&b, "- %s — %s: %s\n", c.CriterionID, c.Verdict, free(c.Rationale))
		for _, e := range c.Evidence {
			fmt.Fprintf(&b, "  evidence: %s\n", renderEvidence(e, free))
		}
	}
	b.WriteString("\nGap audit:\n")
	renderBucket(&b, "honoured", v.GapAudit.Honoured, free)
	renderBucket(&b, "diverged", v.GapAudit.Diverged, free)
	renderBucket(&b, "missing", v.GapAudit.Missing, free)
	renderDispositions(&b, v.ScopeConditions, free)
	return closeReviewBlock(&b, rcp)
}

// renderDispositions writes the scope-condition disposition block — the ONE
// renderer both the INGESTED and the DEAD_LETTER path use, so the two can never
// disagree about the shape of the surface. An intent that records no conditions
// gets no block at all: a heading over nothing asserts a surface the record does
// not carry, and its absence is what keeps the staged rollout invisible to every
// intent shipped before the identity mint existed.
//
// Every field is agent-produced and lands in a committed record, so all of them
// go through oneLine — the same neutraliser the per-criterion render uses, so no
// payload can forge an `<!-- abcd-review: … -->` marker and spoof review state.
// The FREE-TEXT ones go through free, which redacts first: an identifier and an
// enum token are validated shapes that carry nothing to redact, while a
// rationale and a narrowing are prose an agent wrote.
func renderDispositions(b *strings.Builder, conds []verdictCondition, free proseField) {
	if len(conds) == 0 {
		return
	}
	b.WriteString("\nScope-condition dispositions:\n")
	for _, c := range conds {
		writeDispositionBullet(b, oneLine(c.ConditionID), oneLine(c.Disposition), free(c.Rationale), free(c.Narrowing))
		for _, e := range c.Evidence {
			fmt.Fprintf(b, "  evidence: %s\n", renderEvidence(e, free))
		}
	}
}

func renderBucket(b *strings.Builder, name string, entries []verdictGapEntry, free proseField) {
	if len(entries) == 0 {
		fmt.Fprintf(b, "- %s: (none)\n", name)
		return
	}
	fmt.Fprintf(b, "- %s:\n", name)
	for _, e := range entries {
		fmt.Fprintf(b, "  - %s\n", free(e.Claim))
		for _, ev := range e.Evidence {
			fmt.Fprintf(b, "    evidence: %s\n", renderEvidence(ev, free))
		}
	}
}

// renderEvidence writes one evidence pointer. The quote is delimited with plain
// quotation marks rather than %q, and that is not a style choice: %q REWRITES the
// cleaned bytes — it doubles every backslash — and the cleaner's guarantees are
// stated over the exact string it returned (see the invariant note in
// internal/termsafe/prose.go). The backslash the cleaner writes to escape a stray
// backtick came back through %q doubled — an escaped backslash followed by a LIVE
// backtick — putting an unpaired run into a committed record and reopening the
// re-pairing hole this file's other embeddings close. Both fields are already
// newline-free and control-rune-free, which is all %q was buying here.
func renderEvidence(e verdictEvidence, free proseField) string {
	ref := free(e.Ref)
	if q := free(e.Quote); q != "" {
		return fmt.Sprintf(`%s — "%s"`, ref, q)
	}
	return ref
}

// ---------------------------------------------------------------------------
// Small helpers
// ---------------------------------------------------------------------------

// sectionBody returns the text of the section introduced by the first heading
// matching headRe, up to the next heading or end of file. It reads the section
// through mdrecord.SectionLineRange, the single notion of where a section
// starts and stops; an absent section and an empty one both read as ""
// here, and a caller that must tell them apart asks for the bounds directly.
func sectionBody(content string, headRe *regexp.Regexp) string {
	lines := strings.Split(content, "\n")
	start, end, ok := mdrecord.SectionLineRange(lines, headRe)
	if !ok {
		return ""
	}
	return strings.Join(lines[start:end], "\n")
}

// countAcceptanceCriteria counts the top-level list bullets in the intent's
// `## Acceptance Criteria` section — the positional authority ac-1..ac-K.
func countAcceptanceCriteria(content string) int {
	n := 0
	for _, ln := range strings.Split(sectionBody(content, acHeadingRe), "\n") {
		if mdrecord.IsTopLevelBullet(strings.TrimRight(ln, "\r")) {
			n++
		}
	}
	return n
}

func countVerdicts(v verdict) map[string]int {
	m := map[string]int{"MET": 0, "MET_WITH_CONCERNS": 0, "NOT_MET": 0, "INCONCLUSIVE": 0}
	for _, c := range v.Criteria {
		m[c.Verdict]++
	}
	return m
}

// countDispositions is the per-value split of the scope-condition dispositions,
// so a surface reports it without re-reading the record.
func countDispositions(v verdict) map[string]int {
	m := map[string]int{"survived": 0, dispositionNarrowed: 0, "falsified": 0, dispositionUntested: 0}
	for _, c := range v.ScopeConditions {
		m[c.Disposition]++
	}
	return m
}

func hasCitedEvidence(ev []verdictEvidence) bool {
	for _, e := range ev {
		if strings.TrimSpace(e.Ref) != "" {
			return true
		}
	}
	return false
}

// maxNoteFieldBytes caps one untrusted verdict field rendered into the committed
// Audit Notes. It matches the release ingest's per-entry cap: a rationale, a
// claim or a quoted line is a sentence or a few, and an unbounded field is the
// one a hostile verdict uses to bury the record.
const maxNoteFieldBytes = 4096

// oneLine sanitises an untrusted verdict string before it is rendered into the
// committed Audit Notes. It is termsafe.CleanProseLine under this package's cap,
// NOT a second sanitiser: that package is the canonical home for the
// untrusted-prose cleaner every host-delegated ingest boundary needs, and routing
// through it is what gives this record the same guarantees the others have —
// newlines collapse (so injected content cannot break out of its line), HTML
// comment delimiters are broken apart (so untrusted content can never forge an
// `<!-- abcd-review: <STATE> receipt=<rcp> -->` marker to spoof review state,
// misroute a future ingest, or poison idempotency into a false no-op), raw HTML
// cannot open, terminal-display attack runes are masked, and markdown link
// syntax is neutralised so a faithful quotation of code such as
// `items[0](itm-0001)` cannot trip the links_resolve gate on the record this
// ingest just wrote (iss-2608311504353427). Every untrusted field rendered into
// the record passes through here.
func oneLine(s string) string {
	return termsafe.CleanProseLine(s, maxNoteFieldBytes)
}

// proseField renders one FREE-TEXT verdict field into the committed Audit Notes:
// privacy redaction first, then oneLine's neutralisation.
//
// The two do different jobs and both are needed. oneLine protects the RECORD's
// structure — a payload cannot forge a review marker or open raw HTML through it
// — and it has always run here. It knows nothing about privacy, so a rationale,
// a narrowing, a gap-audit claim or an evidence pointer carrying an absolute
// home path, a hostname or a person's name was written into the shipped intent
// verbatim, with only the committed-file privacy lint downstream to notice
// (iss-2608300924205748). AGENTS.md's rule governs what lands in a committed
// file, and framework 7.1 puts Audit Notes squarely there: a verdict is revision
// history carried by the intent record.
//
// The ORDER is load-bearing in both directions. Redacting first means the
// detector sees the agent's bytes rather than a neutralised paraphrase of them;
// neutralising last means the final bytes still carry oneLine's guarantees, and
// the masks it is handed (`[redacted-path]` and its siblings) are inert to every
// rule oneLine applies.
//
// It is applied to free text ALONE, and what counts as free text is decided by
// whether a VALIDATOR constrains the field — never by whether its name sounds
// structural. A criterion id, an enum verdict, a disposition value, a condition
// id, and the two policy hashes and an attestation digest under sha256FieldRe
// are validated shapes with nothing to redact, and running a name matcher over
// them would corrupt a value the record is keyed on rather than protect
// anything. Those keep oneLine by itself.
//
// Everything else is free text, the verifier's id and version and an
// attestation's kind and ref included. Those four were once excused here as
// validated shapes while nothing validated them, and an attestation ref is prose
// by construction — the contract's own example is a commit range with a
// parenthetical beside it (iss-2609022002241168).
type proseField func(string) string

// newVerdictProse builds the free-text renderer for one ingest, failing closed
// on a degraded scanner before any block is composed.
func newVerdictProse(repoRoot string) (proseField, error) {
	redact, err := newIntentRedactor(repoRoot)
	if err != nil {
		return nil, err
	}
	return func(s string) string {
		redacted, _ := redact(s)
		return oneLine(redacted)
	}, nil
}

func orDash(s string) string {
	if s = oneLine(s); s == "" {
		return "-"
	}
	return s
}

// orFree is orDash for a FREE-TEXT field: the same em-dash for an empty value,
// with proseField's privacy redaction ahead of oneLine's neutralisation. It is
// what a provenance field takes when the contract gives it no shape.
func orFree(s string, free proseField) string {
	if s = free(s); s == "" {
		return "-"
	}
	return s
}

// atoiPositive parses a non-negative decimal string (already ^[0-9]+$ via regex).
func atoiPositive(s string) int {
	n := 0
	for _, r := range s {
		n = n*10 + int(r-'0')
		if n > 1_000_000 {
			return n // clamp; huge indices are out-of-range anyway
		}
	}
	return n
}
