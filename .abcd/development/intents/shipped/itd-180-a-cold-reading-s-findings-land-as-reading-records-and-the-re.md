---
id: itd-180
slug: a-cold-reading-s-findings-land-as-reading-records-and-the-re
spec_id: spc-58
kind: standalone
suggested_kind: standalone
reclassification_history: []
builds_on: [itd-86]
severity: major
impact: additive
---

# A cold reading's findings land as reading records, and the researcher's response is a separate disposition record — two acts, two writes, never collapsed

Typed links: `refines` [itd-86](../drafts/itd-86-cold-reading-surface.md) (the
detection shape); the hold state's axes are shared with the itd-142 hold
register and the iss-2608220750029991 triage-route seed.

## Press Release

> **A reading record is not a capture with a flag, and a disposition is not an
> issue state.** A capture is something a person noticed; a reading record is
> something an instrument returned under a recorded visible world — it
> carries a run-scoped identifier, a run identifier, a manifest reference,
> its position and regime value, and a position-typed body (for the
> detection pass: tension, constraint in play, why it is a tension). The researcher's
> response is a second record, keyed to the item identifier, from a
> five-state vocabulary whose availability varies by position: `accepted`
> (all positions — at the widening position it IS admission), `rejected`
> (a testable purpose is asserted; never at the widening position),
> `declined` (widening only — the proposal was admissible and the
> researcher chose otherwise, asserting nothing testable), `held`
> (directional, with an epistemic exit condition). The record can therefore always show that a
> reading record existed before it was dispositioned.

## What's In Scope

- The reading record type (renamed from "detection", ruled 2026-08-28 —
  the registrative body and the Step-6 instrument keep the detection
  name; the record type is the reading record): a common envelope the
  instrument stamps — identifier (run-scoped, adr-45 mint, never
  content-derived, else a re-raise is indistinguishable from its first
  appearance and the recurrence signal dies), run identifier, manifest
  reference, position and regime value from the definition, and the
  pattern named — plus a position-typed body from the reading.
- The bodies, one per position — the pattern named sits in the envelope,
  never in a body, since a universal core condition must not live in a
  variant part: registrative — tension · constraint in play · why it is a
  tension; generative — configuration · what admits it; explicative —
  claim surfaced · claim type · what implies it; evaluative — candidate
  identifier · criterion · characterisation. One record type: one lint,
  one disposition surface, one identifier scheme; four types are
  rejected. Fallback if the discriminated union proves awkward in build:
  one type, untyped body, and a lint asserting the required fields per
  position — weaker, and adequate.
- The disposition record, written separately (ruled 2026-08-28, R7):
  five states, availability varying by position — `accepted` (all
  positions; at the widening position acceptance IS admission, since a
  state encoding position would duplicate the envelope), `rejected`
  (explicative/evaluative/registrative only: it asserts a purpose the
  closing run tests), `declined` (widening only: the proposal was
  admissible and the researcher chose otherwise — forcing that into
  `rejected` would manufacture a principle never at stake), `held`
  (exit condition required; availability at the widening position still
  open). The grounds field is `disposition_grounds`, required on every
  state except `held`; what it must contain varies by state, enforced by
  lint rather than by four fields. Free text, not enumerations. Nothing
  meaning "already covered" exists in any position; an undispositioned
  item is reported as outstanding, not named as a state. The disposition
  record reads the envelope's position to validate its own state — a
  coupling the schema carries and the lint checks — and the
  admitted-against-declined count at the widening position is the
  ownership evidence, queryable without reading prose.
- The recurrence link, on the warm side: a disposition may cite prior
  item identifiers (`recurs`), and a re-acceptance or re-rejection
  made against evidence of persistence carries that citation — the
  stronger record recurrence-is-signal describes, and the answer to the
  duplicate case (re-disposition with citation, not a fourth state). The
  reading never sees it: the assembler excludes dispositions, so the link
  lives entirely on the ledger side. **Adopted (2026-08-28):** the citation is the recorded form of the
  researcher's warm recognition — new material relative to the governing
  design, adopted, and routed back as an amendment to it.
- On-disk shape: reading records land under `issues/readings/<run-id>/`
  and dispositions under `issues/dispositions/`, keyed by the item
  identifier. The status signal is the presence of the keyed disposition —
  never folder membership — and RS001–RS003, `capture resolve`, and the
  changelog derivation are taught in the same change to scope to
  open/resolved/wontfix and ignore both new types.
- The output contract that "validated" refers to is owned by the
  cold-reading output-contract intent; this intent consumes it and adds no
  second validation path.
- The surprise entry (per the recording-obligations ruling): a distinct record shape, schema reserved in
  this family now and populated in Iteration 2 — the reading's output, the
  researcher's disposition, and the surprise that occasions abduction are
  three acts, three records. The admission-side records (grounds on
  admission, declined-proposal dispositions) live in the
  step2-admission-records draft.
- The reserved two-axis hold field (frame-location × MoSCoW), present in
  the schema and unpopulated — deferred by decision; reserving costs
  nothing, retrofitting is expensive. Defined-and-dormant: the value
  grammars are stated (frame-location: free text naming the frame element;
  MoSCoW: must / should / could / wont) and a populated value is refused
  until activation is ruled — refusal, not silent acceptance, so the
  reservation is a behaviour rather than a comment.
- The lint: every reading record in a run either carries a disposition or is
  reported as outstanding — report-only, on the capture status board and
  the lint summary; it never gates preflight or CI, so a reading can never
  block unrelated pushes. Open holds render on the same board with their
  exit conditions; a hold exits only through a superseding disposition
  that cites it — never by expiry, and never silently.
- Routing on acceptance: action is a separate admission joined by the
  item identifier stamped forward on `promoted_to` and back in
  `origin`. Item-to-intent without a disposition is the collapse this
  record family exists to prevent: `capture promote` refuses an item
  identifier that carries no disposition, and a circumvention is a
  lapse-log entry.

## Ruled

- **Ruled (maintainer, 2026-08-28; decision log):** the reading record is a distinct record type in
  the existing issue tier, with a separate disposition record and its own
  disposition vocabulary — the five-state, availability-by-position slate
  (R7) this draft implements. Ground: the
  surprise entry and the disposition are different acts and must be
  distinguishable records; reusing the issue states collapses them and
  misdescribes all three.
- **Recurrence matching is warm work (per the closing-run ruling):** run-scoped identifiers
  join nothing mechanically; the researcher recognises a recurrence
  against the ledger, and the recognition is itself a disposition
  judgement — the `recurs` citation in scope is that recognition's
  recorded form.
- **Where an accepted item goes (per the acceptance-routing ruling):** acceptance is one
  record; the action is a separate admission and build, joined by the
  item identifier (forward on `promoted_to`, back in `origin` with
  the run identifier). The landings are enumerated — artefact via the
  intent lifecycle, cross-cutting rule via a discipline, redecision via a
  superseding ADR, the brief's description via the delivering change, the
  construal via a section rewrite with the prior construal passing to
  ledger content; the frame-level record is iteration 2, the fourth
  verdict deferred.

## Open (maintainer readings design, 2026-08-28)

- Whether `held` is available at the widening position: a configuration
  held with an exit condition is a candidate deferred, which is what
  `deferred` already means in the selection-grounds vocabulary
  (pursued / deferred / declined) — two words for one act is the drift
  the design avoids elsewhere. The alternative routes a deferred
  configuration through the selection surface instead of the disposition
  surface. Reaches the selection vocabulary as well; does not gate the
  build.

## What's Out of Scope

- Reusing open/resolved/wontfix: `resolved` means fixed, but an accepted
  detection may be deliberately not acted on; `wontfix` means will-not-act,
  whereas rejection asserts an intentional constraint; `open` is a parking
  space, whereas held is directional.
- Passing dispositions to a reading — warm by definition; the assembler
  excludes them and the read-block eval asserts it.

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** a validated reading output, **when** it is ingested, **then**
  one reading record is written per item, each with a run-scoped
  identifier.
- **Given** two runs producing the same tension, **when** both are
  ingested, **then** the two records carry different identifiers.
- **Given** any disposition with empty `disposition_grounds` (or a hold
  with an empty exit condition), **when** the command runs, **then** it
  refuses.
- **Given** a disposition whose state is unavailable at the item's
  position (a `declined` on a detection, a `rejected` on a widening
  configuration), **when** the command runs, **then** it refuses and
  names the availability rule.
- **Given** a run's reading records, **when** the lint runs, **then** every
  record either carries a disposition or is reported as outstanding.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-6d6ab907bc1c -->
Fidelity review — receipt rcp-6d6ab907bc1c (verifier abcd:intent-auditor claude-fable-5).

Provenance: abcd:intent-auditor@claude-fable-5 · rubric_hash sha256:f3bba86a84b329fcbfbd3df64ec5d77d382dda9b4b1a61baf887e316bc41f38e · prompt_hash sha256:77eb6fcc7ae29cb828c8e98afa18d6d0dbd26ff84c66f2d977779b071277023d
Input attestations: diff:932629f9^1...932629f9^2 (merge 932629f9fe1334a333067cc6835e90e1fcfb7a2e, build/itd-180 into experiment/cold-reading)@sha256:4fe10729c6a205a09991d684974d37a5260d966349818a8c8fb6b7e0f436f956;

Acceptance rollup: MET 3 · MET_WITH_CONCERNS 2 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: capture.IngestReading mints one rdi-N per item under readings/<run-id>/ with `run` in every envelope and TestIngestWritesOneRecordPerItem proves it (go test passes), but no CLI or plugin verb calls IngestReading — by design the not-yet-built spc-63 ingest verb is the only door — so 'when it is ingested' cannot happen from a production entry point this cycle
  evidence: internal/core/capture/reading.go:144 — "func IngestReading(req IngestReadingRequest) (IngestReadingResult, error) {"
  evidence: internal/core/capture/reading.go:236 — "result.Records = append(result.Records, ReadingRecordRef{"
  evidence: internal/core/capture/ingest_reading_test.go:21 — "func TestIngestWritesOneRecordPerItem(t *testing.T) {"
  evidence: internal/core/capture/ingest_reading_test.go:59 — "strings.Contains(string(content), \"run: \\\"\"+run+\"\\\"\")"
  evidence: .abcd/development/brief/04-surfaces/06-capture.md:139 — "the only caller of `capture.IngestReading`, so until it lands the two reading sub-verbs have no item to act on"
- ac-2 — MET: ids are minted (adr-45 timestamp plus entropy), never content-derived, and the mint probes every readings/rdg-*/ under the ledger lock and redraws on a hit; TestTwoRunsSameTensionMintDistinctIDs and TestTwoRunsAtOneInstantMintDistinctIDs (pinned same-second, same-suffix minter) both pass
  evidence: internal/core/capture/reading.go:856 — "func mintUnusedItemID(issuesRoot string, minted map[string]bool) (string, error) {"
  evidence: internal/core/capture/reading.go:201 — "id, err := mintUnusedItemID(issuesRoot, minted)"
  evidence: internal/core/capture/ingest_reading_test.go:68 — "func TestTwoRunsSameTensionMintDistinctIDs(t *testing.T) {"
  evidence: internal/core/capture/ingest_reading_test.go:178 — "func TestTwoRunsAtOneInstantMintDistinctIDs(t *testing.T) {"
  evidence: .abcd/work/issues/resolved/iss-2608300227228575-reading-item-ids-collide-across-runs.md:10 — "The ingest mint now probes every readings/rdg-*/ under the ledger lock and redraws on a hit"
- ac-3 — MET: validateDispositionStrict requires a non-blank disposition_grounds on every state except held and a non-blank exit_condition on held, both refusals reached from the wired `abcd capture disposition` verb; TestDispositionRefusesEmptyGrounds, TestHeldDispositionRefusesEmptyExitCondition and the CLI-level TestCaptureDispositionRefusesEmptyGroundsAndWritesNothing pass
  evidence: internal/core/capture/reading.go:579 — "if err := requireNonBlankString(fm, \"exit_condition\"); err != nil {"
  evidence: internal/core/capture/reading.go:583 — "if err := requireNonBlankString(fm, \"disposition_grounds\"); err != nil {"
  evidence: internal/core/capture/reading_test.go:97 — "func TestDispositionRefusesEmptyGrounds(t *testing.T) {"
  evidence: internal/core/capture/reading_test.go:116 — "func TestHeldDispositionRefusesEmptyExitCondition(t *testing.T) {"
  evidence: internal/surface/cli/cli.go:2632 — "dispositionCmd := &cobra.Command{"
  evidence: internal/surface/cli/capture_surface_test.go:559 — "func TestCaptureDispositionRefusesEmptyGroundsAndWritesNothing(t *testing.T) {"
- ac-4 — MET_WITH_CONCERNS: the verb reads position off the keyed reading record, consults issueschema.DispositionAvailability and refuses a ruled-unavailable pair with a message naming the state, the position and the states available there; TestDispositionRefusesStateUnavailableAtPosition covers exactly the two named cases (declined on detection, rejected on widening) and passes — the concern is that `held` at the widening position ships as an unruled row and is accepted silently rather than refused, a signed-off deferral (spec Out of scope) but a combination the availability rule does not judge
  evidence: internal/core/capture/reading.go:596 — "available, ruled := issueschema.DispositionStateAvailable(position, state)"
  evidence: internal/core/capture/reading.go:598 — "\"%w: %q is not available at the %s position (available there: %s); the disposition record validates its state against the envelope's position\""
  evidence: internal/core/issueschema/reading.go:188 — "// DispositionHeld: deferred, deliberately absent."
  evidence: internal/core/capture/reading_test.go:134 — "func TestDispositionRefusesStateUnavailableAtPosition(t *testing.T) {"
  evidence: .abcd/development/specs/closed/spc-58-a-cold-reading-s-findings-land-as-reading-records-and-the-re.md:246 — "the availability table ships with that row unfilled, so the refusal for that one combination is not armed"
- ac-5 — MET: the reading_outstanding rule walks every readings/<run>/rdi-*.md and reports each item with no standing disposition as outstanding (severity pinned to info in code), is enabled in .abcd/record-lint.json, is registered in lint.go and rides the bare `abcd capture` board through the same ReadReadingOutstanding function; TestOutstandingReportNamesUndispositionedItems, TestOutstandingReportSeverityIsInfoNotBlocker and TestCaptureBoardCarriesTheOutstandingRoster pass
  evidence: internal/core/lint/readingoutstanding.go:136 — "func ReadReadingOutstanding(repoRoot, issuesDir string) (OutstandingReadings, error) {"
  evidence: internal/core/lint/readingoutstanding.go:225 — "case answer.standing == nil:"
  evidence: internal/core/lint/lint.go:535 — "if roCfg, ok := cfg.Rules[ruleReadingOutstanding]; ok && roCfg.Enabled {"
  evidence: .abcd/record-lint.json:237 — "\"reading_outstanding\": {"
  evidence: internal/core/lint/reading_outstanding_test.go:49 — "func TestOutstandingReportNamesUndispositionedItems(t *testing.T) {"
  evidence: internal/surface/cli/capture_surface_test.go:614 — "func TestCaptureBoardCarriesTheOutstandingRoster(t *testing.T) {"

Gap audit:
- honoured:
  - reading record and disposition are two records, two writes, keyed by the item id
    evidence: internal/core/capture/reading.go:260 — "func Disposition(req DispositionRequest) (DispositionResult, error) {"
    evidence: internal/core/capture/reading.go:299 — "itemDir := filepath.Join(issuesRoot, issueschema.DispositionsDir, req.Item)"
  - envelope carries run-scoped id, run, manifest, position, regime and pattern; pattern is never in a body
    evidence: internal/core/issueschema/reading.go:117 — "var ReadingRequired = []string{ \"schema_version\", \"id\", \"run\", \"manifest\", \"position\", \"regime\", \"pattern\" }"
  - one record type, four position-typed bodies held as data (the intent's stated fallback)
    evidence: internal/core/issueschema/reading.go:99 — "var ReadingPositions = []ReadingPosition{"
  - the disposition reads the envelope's position to validate its own state
    evidence: internal/core/capture/reading.go:278 — "position, err := readingItemPosition(issuesRoot, req.Item)"
  - a hold exits only through a superseding disposition that cites it; the superseded record stays in place
    evidence: internal/core/capture/reading.go:328 — "if req.Supersedes == \"\" && len(standing) > 0 {"
    evidence: internal/core/capture/ingest_reading_test.go:127 — "the superseded record must stay in place"
  - recurs citation lives on the ledger side as a disposition field, never a state
    evidence: internal/core/issueschema/reading.go:305 — "\"recurs\": true,"
  - reserved two-axis hold field and surprise key: schema-present, populated value refused
    evidence: internal/core/issueschema/reading.go:233 — "var ReservedHoldFields = []string{\"hold_frame_location\", \"hold_moscow\"}"
    evidence: internal/core/capture/reading_test.go:170 — "func TestPopulatedHoldAxisRefused(t *testing.T) {"
  - the lint is report-only and never gates: severity pinned to info in code, on the lint findings and the capture board
    evidence: internal/core/lint/readingoutstanding.go:39 — "const severityInfo = \"info\""
    evidence: internal/core/lint/reading_outstanding_test.go:82 — "func TestOutstandingReportSeverityIsInfoNotBlocker(t *testing.T) {"
  - open holds render on the board with their exit conditions
    evidence: internal/core/lint/reading_outstanding_test.go:107 — "func TestOpenHoldRendersItsExitCondition(t *testing.T) {"
  - capture promote refuses an item identifier that carries no disposition, and stamps promoted_to / origin on success
    evidence: internal/core/capture/promote.go:412 — "carries no disposition; an item is answered before it is acted on"
    evidence: internal/core/capture/promote_test.go:1 — "TestPromoteRefusesUndispositionedReadingItem / TestPromoteStampsReadingItemPromotedTo"
  - RS001-RS003 and record_schema taught to scope to open/resolved/wontfix and ignore both new families
    evidence: scripts/check-issue-resolution.sh:77 — "STATUS_DIRS=(open resolved wontfix)"
    evidence: internal/core/issueschema/reading.go:36 — "They are deliberately NOT in StatusDirs"
  - on-disk shape readings/<run-id>/ and dispositions/<item-id>/
    evidence: internal/core/issueschema/reading.go:43 — "ReadingsDir = \"readings\""
    evidence: internal/core/issueschema/reading.go:50 — "DispositionsDir = \"dispositions\""
- diverged:
  - a five-state disposition vocabulary
    evidence: internal/core/issueschema/reading.go:142 — "itd-180 and ruling (19) both say FIVE states and both enumerate four"
    evidence: internal/core/issueschema/reading.go:169 — "var DispositionStates = []string{ DispositionAccepted, DispositionRejected, DispositionDeclined, DispositionHeld }"
  - availability of every state at every position is ruled by the table (held at widening left open by the intent)
    evidence: internal/core/issueschema/reading.go:183 — "var DispositionAvailability = map[string]map[string]bool{"
    evidence: internal/core/capture/reading.go:597 — "if ruled && !available {"
  - promote refuses an item that carries no disposition — the build refuses every standing state but accepted, and refuses under contest
    evidence: internal/core/capture/promote.go:421 — "only %q licenses an action; supersede it with a new disposition first"
    evidence: internal/core/capture/promote.go:370 — "ErrInvariantViolation, item, len(standing), renderList(standing))"
  - a second disposition always supersedes the standing one — the verb instead refuses when more than one answer stands and prescribes a hand repair
    evidence: internal/core/capture/reading.go:318 — "if len(standing) > 1 {"
    evidence: .abcd/work/issues/resolved/iss-2608300835066901-itd-180-fifth-round-security-findings.md:10 — "capture disposition refuses under contest exactly as promote does and names the hand repair"
  - every record either carries a disposition or is outstanding — the report also names contested, cyclic, unreadable and unsafe items as distinct classes
    evidence: internal/core/lint/readingoutstanding.go:210 — "switch {"
    evidence: internal/core/lint/readingoutstanding.go:87 — "Contested []ContestedItem `json:\"contested,omitempty\"`"
  - changelog derivation taught in the same change to scope to the status folders
    evidence: internal/core/release/emit_test.go:22 — "resolvedDir = \".abcd/work/issues/resolved/\""
    evidence: .abcd/development/specs/closed/spc-58-a-cold-reading-s-findings-land-as-reading-records-and-the-re.md:158 — "scripts/check-issue-resolution.sh replaces its single $ISSUES_DIR pathspec"
- missing:
  - a validated reading output can be ingested from a production entry point (CLI or plugin verb) — no front door calls IngestReading; no rdi-N can be minted through the surface this cycle
    evidence: .abcd/development/brief/04-surfaces/06-capture.md:139 — "the only caller of `capture.IngestReading`, so until it lands the two reading sub-verbs have no item to act on"
    evidence: commands/capture.md:36 — "until that verb lands there is no reading item to answer, and these two sub-verbs have nothing to act on"
  - the committed-tree gate judges reading and disposition content (one lint over the record type) — record_schema declares no required fields for the three families, so a hand-written record the writer would refuse still passes the tree gate
    evidence: internal/core/capture/reading.go:16 — "(record_schema) declares no required fields for these two families yet, so it judges their SHAPE"
    evidence: .abcd/work/issues/resolved/iss-2608300257421526-itd-180-second-round-residue.md:10 — "schema.go states the reading stores' missing required fields as a gap the writer and review cover"
  - a fifth disposition state
    evidence: internal/core/issueschema/reading.go:146 — "The discrepancy is carried here, not resolved."

## Grounds

- pursued: Pursued now because the instrument's first run happens in this cycle, and a record written after the fact is precisely the retrospective reconstruction the ledger exists to prevent: unless the reading record and the disposition are two writes from the start, nothing can later show that a finding existed before it was answered. Ruling (2) settled the shape; the only remaining question was whether to build it before or after the first run, and after is too late.
