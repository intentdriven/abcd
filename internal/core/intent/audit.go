package intent

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
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
	auditHeadingRe = regexp.MustCompile(`^#{1,6}\s+Audit Notes\s*$`)
	// markerRe matches a parked review marker LINE inside the Audit Notes. It is
	// line-anchored and whole-line on purpose: the marker is the ledger's own
	// review state, and an unanchored pattern would find one anywhere in the
	// record's bytes — mid-sentence inside a rendered verdict field, for instance,
	// where an untrusted payload put it. termsafe's cleaner is the primary defence
	// (it breaks `<!` and `-->` in every field it writes, code span or not); this
	// is the second, so a marker has to occupy a line of its own to count.
	//
	// It is still a byte pattern rather than a grammar: it does not know a fenced
	// block from prose, so a marker-shaped line inside a fence still matches
	// (iss-2609020529185438). Both defences are needed; neither is sufficient.
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
type AuditEmitResult struct {
	ReceiptID   string `json:"receipt_id"`
	IntentID    string `json:"intent_id"`
	Status      string `json:"status"` // owed | already_owed | already_ingested | already_dead_letter
	RequestPath string `json:"request_path"`
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
		res.RequestPath = filepath.Join(reviewsRelDir, rcp+".request.md")
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
		}
		return res, nil
	}

	rcp := receiptFor(it.ID, it.SpecID, content)
	res := AuditEmitResult{ReceiptID: rcp, IntentID: it.ID}
	block := owedBlock(rcp)
	updated := upsertReviewBlock(content, rcp, block)
	if err := writeIntentFile(abs, it.Path, updated); err != nil {
		return AuditEmitResult{}, err
	}
	if err := writeAuditRequest(repoRoot, it, rcp, updated, opts); err != nil {
		return AuditEmitResult{}, err
	}
	res.Status = "owed"
	res.RequestPath = filepath.Join(reviewsRelDir, rcp+".request.md")
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
	b.WriteString("## Acceptance Criteria (authority; numbered ac-1..ac-K in order)\n\n")
	if ac == "" {
		b.WriteString("(none found)\n")
	} else {
		b.WriteString(ac + "\n")
	}
	b.WriteString("\n## Rubric (authority; the contract the ingest enforces)\n\n")
	b.WriteString(rubricText())
	b.WriteString("\nRun the intent-auditor agent over the criteria and the delivered\n")
	b.WriteString("diff, then ingest its verdict JSON:\n\n")
	fmt.Fprintf(&b, "    abcd intent audit ingest --verdict-json <path>   # receipt %s\n", rcp)
	return b.String()
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
// owed a verdict, and a bad re-ingest must never replace a good one.
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
	if existing, ok := reviewBlockText(content, rcp); ok && existing == block {
		return IngestVerdictResult{Status: "noop", ReceiptID: rcp, IntentID: it.ID}, nil
	}
	if err := checkIssuedPolicy(repoRoot, raw, it, rcp, content); err != nil {
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
// DEAD_LETTER block recording all criteria INCONCLUSIVE. Never partial.
func deadLetter(repoRoot string, it Intent, content, rcp string, raw []byte, reason string, free proseField) (IngestVerdictResult, error) {
	if !rcpIDRe.MatchString(rcp) {
		return IngestVerdictResult{}, fmt.Errorf("intent: receipt id %q is malformed; refusing to dead-letter", rcp)
	}
	dir := filepath.Join(repoRoot, reviewsRelDir)
	if err := ensureRecordDir(repoRoot, reviewsRelDir); err != nil {
		return IngestVerdictResult{}, err
	}
	dlRel := filepath.Join(reviewsRelDir, rcp+".deadletter.json")
	if err := fsutil.WriteFileAtomic(filepath.Join(dir, rcp+".deadletter.json"), raw, 0o644); err != nil {
		return IngestVerdictResult{}, fmt.Errorf("intent: retaining dead-letter payload %s: %w", dlRel, err)
	}
	untested := untestedDispositions(content)
	block := deadLetterBlock(rcp, reason, dlRel, untested, free)
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

// existingMarker returns the receipt id and state of the FIRST parked review
// marker in content, if any. Emit reuses this parked receipt rather than
// recomputing one (see emitAuditForIntent's receipt-shift note).
func existingMarker(content string) (string, string, bool) {
	if m := markerRe.FindStringSubmatch(content); m != nil {
		return m[2], m[1], true
	}
	return "", "", false
}

// markerState returns the state of the review marker for rcp, if present.
func markerState(content, rcp string) (string, bool) {
	for _, m := range markerRe.FindAllStringSubmatch(content, -1) {
		if m[2] == rcp {
			return m[1], true
		}
	}
	return "", false
}

// upsertReviewBlock replaces the existing review block for rcp with newBlock, or
// appends newBlock to the Audit Notes section (creating the section if absent). A
// review block runs from its marker line to the next block marker of EITHER
// grammar (condition.IsBlockMarker), the next heading, or end of file — so a
// condition block written after an OWED stub survives the stub's replacement
// rather than being swallowed as part of it (spc-2609020626046252).
func upsertReviewBlock(content, rcp, newBlock string) string {
	lines := strings.Split(content, "\n")
	if start, end, ok := reviewBlockRange(lines, rcp); ok {
		// Keep the blank separator the old block ended with, so a block that
		// follows it is not glued to the replacement.
		sep := end
		for sep > start+1 && strings.TrimSpace(lines[sep-1]) == "" {
			sep--
		}
		out := make([]string, 0, len(lines))
		out = append(out, lines[:start]...)
		out = append(out, strings.Split(newBlock, "\n")...)
		out = append(out, lines[sep:]...)
		return strings.Join(out, "\n")
	}
	return appendToAuditNotes(content, newBlock)
}

// reviewBlockRange locates the review block for rcp in lines: from its marker
// line to the next block marker of either grammar, the next heading, or end of
// file. It is the one notion of a review block's extent, so the replacement and
// the idempotency comparison cannot disagree about where a block ends.
func reviewBlockRange(lines []string, rcp string) (start, end int, ok bool) {
	for i, ln := range lines {
		m := markerRe.FindStringSubmatch(strings.TrimRight(ln, "\r"))
		if m == nil || m[2] != rcp {
			continue
		}
		end = len(lines)
		for j := i + 1; j < len(lines); j++ {
			t := strings.TrimRight(lines[j], "\r")
			if condition.IsBlockMarker(t) || mdrecord.IsHeading(t) {
				end = j
				break
			}
		}
		return i, end, true
	}
	return 0, 0, false
}

// reviewBlockText is the review block for rcp as a renderer would have written
// it: its lines with the trailing blank separator trimmed.
func reviewBlockText(content, rcp string) (string, bool) {
	lines := strings.Split(content, "\n")
	start, end, ok := reviewBlockRange(lines, rcp)
	if !ok {
		return "", false
	}
	return strings.TrimRight(strings.Join(lines[start:end], "\n"), "\r\n\t "), true
}

// appendToAuditNotes appends a block to the `## Audit Notes` section, creating
// the section at end of file if it is absent.
func appendToAuditNotes(content, block string) string {
	lines := strings.Split(content, "\n")
	head := -1
	for i, ln := range lines {
		if auditHeadingRe.MatchString(strings.TrimRight(ln, "\r")) {
			head = i
			break
		}
	}
	if head < 0 {
		body := strings.TrimRight(content, "\n")
		return body + "\n\n## Audit Notes\n\n" + block + "\n"
	}
	// Find the end of the Audit Notes section (next heading or EOF).
	end := len(lines)
	for j := head + 1; j < len(lines); j++ {
		if mdrecord.IsHeading(strings.TrimRight(lines[j], "\r")) {
			end = j
			break
		}
	}
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
	return strings.Join(rebuilt, "\n")
}

// ---------------------------------------------------------------------------
// Block rendering (deterministic; no timestamps)
// ---------------------------------------------------------------------------

func owedBlock(rcp string) string {
	return fmt.Sprintf("<!-- abcd-review: OWED receipt=%s -->\nFidelity review OWED (receipt %s).", rcp, rcp)
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
	return strings.TrimRight(b.String(), "\n")
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
	return strings.TrimRight(b.String(), "\n")
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
