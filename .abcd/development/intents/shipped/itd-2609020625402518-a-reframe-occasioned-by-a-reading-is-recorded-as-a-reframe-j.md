---
id: itd-2609020625402518
slug: a-reframe-occasioned-by-a-reading-is-recorded-as-a-reframe-j
spec_id: spc-2609020626048705
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-180, itd-189]
severity: minor
impact: additive
origin: researcher-authored
production_mode: dictated-and-formatted
---

# A reframe occasioned by a reading is recorded as a reframe, joined to what occasioned it, without carrying the construal it replaced

Typed links: `builds_on` [itd-180](itd-180-a-cold-reading-s-findings-land-as-reading-records-and-the-re.md) (record families in the issue tier), [itd-189](itd-189-what-the-widening-reading-proposes-is-admitted-or-declined-o.md) (the surprise entry as its own act); `refines` [adr-55](../../decisions/adrs/0055-the-construal-stands-in-the-record-its-history-does-not.md) (a reframe record beside the construal, adopted as [adr-2609021016288378](../../decisions/adrs/2609021016288378-a-reframe-occasioned-by-a-reading-is-a-committed-pointer-to.md) in its three-surface form).

## Press Release

> **A change to the frame is a record, not a diff.** `abcd capture reframe --occasioned-by <rdi-N|dsp-N|srp-N> --grounds "<why>"` writes one frame-level revision record when the frame is rewritten because of a reading, naming what occasioned it, the content fingerprint of each of the frame's three surfaces before and after (the construal section, the committed glossary terms and the committed scope), which of them changed, and the grounds. The prior text of any surface passes to ledger content on the local side, as adr-55 requires, so the committed record shows that a reframe happened, when, why and where, without committing the framing it abandoned. `abcd <id>` on a reframe reports its occasion, the fingerprints and which surfaces moved.

> "When a detection sends me back to the frame rather than to the artefact, I need the record to say that is what happened," said an AI/agent researcher who keeps their own design record. "Otherwise a reframe looks like an edit to a paragraph, and the reading that caused it gets no credit and no blame."

## Why This Matters

The cold-reading design lists where an accepted detection may land: an intent, a discipline, an ADR, a brief passage, the construal section, and the frame. Every landing exists except the last, which the design schedules for Iteration 2 as a frame-level revision record, "without which a reframe occasioned by a reading cannot be recorded as a reframe".

[adr-55](../../decisions/adrs/0055-the-construal-stands-in-the-record-its-history-does-not.md) rules that the construal as it presently stands is committed record and that declined construals, superseded terms and the reasoning that settled a dispute stay on the local side and are read by nothing automated. The brief's framing chapter keeps no history in the section. A frame-level record therefore cannot carry the prior construal's text, and it need not: what is wanted is the join, from a detection to the reframe it occasioned and from a reframe to the reading that caused it, and the fact that the frame moved rather than the artefact.

Without it, Iteration 2's closing run cannot distinguish a tension that was answered by a reframe from one that was answered by a build, and the design's purpose-durability and convergence readings both depend on knowing which.

## Decisions flagged for the maintainer

Both were adopted by the maintainer on 2026-09-02 as [adr-2609021016288378](../../decisions/adrs/2609021016288378-a-reframe-occasioned-by-a-reading-is-a-committed-pointer-to.md), in the three-surface form, after the single-section draft was found to contradict the design's two landings and adr-55's enumeration of the framing's committed surfaces.

- **The frame is the framing as it presently stands, which adr-55 enumerates as three committed surfaces.** The narrowing to the construal section alone, which this intent's first draft flagged, is withdrawn. The record fingerprints the framing chapter's construal section, the committed glossary terms and the committed scope, before and after, and shows which of the three changed; a construal-only rewrite is one instance, not the definition.
- **A reframe record beside the construal refines adr-55.** adr-55 is silent on recording that a rewrite happened; this record adds a committed pointer to an event whose content stays local. Only a reframe a reading occasioned carries a record; a researcher's own rewrite of any surface stays as adr-55 leaves it, and a rule over every frame edit is adoptable later without changing the record's shape.

## What's In Scope

- **A reframe record family** in the working tier beside the other reading families, one record per reframe, minted on the timestamp seam, carrying: the occasion (a reading item, a disposition or a surprise), the content fingerprint of each of the three frame surfaces before and after, which of them changed, and the grounds. It carries no construal text, no term text and no scope text. It is warm and is excluded from every reading by positive inclusion, and the manifest asserts the exclusion.
- **`abcd capture reframe`** as the only writer, refusing an occasion that does not resolve, a ground below the substance floor, and a before-fingerprint it cannot find in the surfaces' history. The verb reads all three surfaces' committed content itself; the operator supplies no hash. The join to the occasion is operator-asserted, with one check: the occasion predates the rewrite.
- **Dispatch** on the reframe id, reporting the occasion, the fingerprints and which surfaces changed.
- The capture surface page documents the verb.

## What's Out of Scope

- The prior text of any surface. It passes to the local side under adr-55 and this record does not carry it.
- Reframes not occasioned by a reading, and any lint over frame edits. adr-2609021016288378 adds a pointer for the reading-occasioned case and no rule over every edit.
- The fourth audit verdict. A reframe changes what is built next; nothing here puts a delivered promise in question, which the design defers by decision.

## Mechanism

We expect a record keyed to the committed fingerprints of the three frame surfaces to make reframes countable without committing framing traces because the fingerprints identify which frame a reading saw and which replaced it, and all three surfaces are already committed content, while the reasoning between them is not. It fails if a rewrite and its record cannot be paired by fingerprint, which happens when a surface is rewritten twice between records; the verb refuses a before-fingerprint it cannot find in the surfaces' history, so the failure is loud.

## Scope Conditions

- The frame is three committed surfaces at fixed paths: the construal section of the framing chapter, the glossary directory's committed content, and the scope chapter of the brief. A repository whose frame lives elsewhere is outside this scope. <!-- cond: cond-2609020626047674 -->
- The verb reads the surfaces at HEAD. A reframe recorded before the rewrite is committed carries the pre-rewrite fingerprints as its before-fingerprints and is completed by a second write after the commit; the verb says which half it wrote. <!-- cond: cond-2609020626049770 -->

## Acceptance Criteria

- **Given** a reading item and any of the three frame surfaces rewritten and committed, **when** `capture reframe` runs with the item as occasion and a ground, **then** one reframe record exists carrying the occasion, a before-fingerprint of each surface matching its previously committed state, an after-fingerprint of each matching its current state, which surfaces changed, and the ground.
- **Given** a frame whose committed surfaces match no known prior state, **when** the verb runs, **then** it refuses and names the mismatch.
- **Given** an occasion that does not resolve to a reading item, a disposition or a surprise, **when** the verb runs, **then** it refuses.
- **Given** a repository holding reframe records, **when** any reading is assembled, **then** no reframe record reaches the bundle and the manifest asserts the family's exclusion.
- **Given** `abcd <reframe-id>`, **when** it runs, **then** it reports the occasion, the fingerprints and which surfaces changed.

## Prior Art

- [adr-55](../../decisions/adrs/0055-the-construal-stands-in-the-record-its-history-does-not.md); the brief's framing chapter; [itd-180](itd-180-a-cold-reading-s-findings-land-as-reading-records-and-the-re.md) and [itd-189](itd-189-what-the-widening-reading-proposes-is-admitted-or-declined-o.md).
- The cold-reading rulings of 2026-08-28 in the decision log.

## Open Questions

None. The flagged decisions are adopted as adr-2609021016288378; the family's identifier prefix is the spec's to fix.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-987317795e99 -->
Fidelity review — receipt rcp-987317795e99 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:a30a0680da0b947809e8292c496f6c476ca7cc55eb34a4f0c4a42671319f65f7
Input attestations: diff:internal/core/capture/reframe.go, internal/core/reading/include.go, internal/core/record/record.go, internal/core/issueschema/, commands/capture.md at 4001d807 (git ls-tree -r)@sha256:2cf4f0f8ca16c675db08e24bf55dc586a85272ba8aa30d02ce0d3be9690bcee3;

Acceptance rollup: MET 5 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: the whole write fingerprints the frame at HEAD, walks the surfaces' history to the previous distinct committed triple, and writes occasion, the three before and three after fingerprints, changed and grounds; TestReframeRecordsACommittedRewrite asserts every field against the fixture's before/after frames and that no surface text reaches the record, with glossary and scope rewrites covered by their own tests
  evidence: internal/core/capture/reframe.go:510 — "head, err := frameAtCommit(repoRoot, "HEAD", cache)"
  evidence: internal/core/capture/reframe.go:558 — "result.Half, result.Before, result.After = ReframeHalfWhole, before, head"
  evidence: internal/core/capture/reframe_test.go:257 — "func TestReframeRecordsACommittedRewrite"
  evidence: internal/core/capture/reframe_test.go:301 — "func TestReframeRecordsAGlossaryRewrite"
  evidence: internal/core/capture/reframe_test.go:317 — "func TestReframeRecordsAScopeRewrite"
- ac-2 — MET: when every fingerprintable state in the walk equals HEAD's the verb refuses naming that no prior committed state matches and how far the history was searched; TestReframeRefusesAFrameWithNoPriorState and TestReframeRefusesNamingHowFarTheFingerprintableHistoryReaches assert the refusal and an unchanged ledger
  evidence: internal/core/capture/reframe.go:548 — "the frame at HEAD matches no prior committed state, so there is no reframe to record"
  evidence: internal/core/capture/reframe_test.go:332 — "func TestReframeRefusesAFrameWithNoPriorState"
  evidence: internal/core/capture/reframe_test.go:349 — "func TestReframeRefusesNamingHowFarTheFingerprintableHistoryReaches"
- ac-3 — MET: the occasion is held to the closed rdi/dsp/srp form and resolved through the shared occasion resolver before anything is minted and again under the lock; TestReframeRefusesAnUnresolvableOccasion covers an absent id of each family, a fourth family, prose and empty, each with the ledger unchanged
  evidence: internal/core/capture/reframe.go:492 — "if !issueschema.ValidReframeOccasion(occasion) {"
  evidence: internal/core/capture/reframe.go:500 — "occPath, err := resolveReframeOccasion(repoRoot, occasion)"
  evidence: internal/core/capture/reframe_test.go:710 — "func TestReframeRefusesAnUnresolvableOccasion"
- ac-4 — MET: the reframes directory is a denied path in the positive walk and the exclusion floor carries a row naming the reframe record at every position; TestReframeRecordsNeverReachTheBundle assembles every position over a committed rfm-1 and asserts neither the bundle nor the manifest carries it
  evidence: internal/core/reading/include.go:699 — "{Rule: "absent from the positive walk", Signal: "record type in a denied path", Detail: "the reframe record"},"
  evidence: internal/core/reading/reframe_test.go:19 — "func TestReframeRecordsNeverReachTheBundle"
  evidence: internal/core/reading/reframe_test.go:62 — "func TestExclusionFloorNamesTheReframeRecord"
- ac-5 — MET: the dispatcher routes rfm-N to describeReframe, which reports occasioned_by and its path, the three before fingerprints, the three after fingerprints and changed once complete, and the open status with its next move otherwise; TestDescribeReframeReportsOccasionAndFingerprints asserts each link
  evidence: internal/core/record/record.go:117 — "case "rfm":"
  evidence: internal/core/record/record.go:562 — "func describeReframe(repoRoot, id string) (Description, error) {"
  evidence: internal/core/record/record_test.go:846 — "func TestDescribeReframeReportsOccasionAndFingerprints"

Gap audit:
- honoured:
  - the record carries no construal, term or scope text, only fingerprints
    evidence: internal/core/capture/reframe_test.go:294 — "if strings.Contains(raw, fxOldPhrase) || strings.Contains(raw, "record of judgement") {"
  - the operator supplies no hash: the verb reads all three surfaces itself at HEAD and in their history
    evidence: internal/core/capture/reframe.go:289 — "func frameAtCommit(repoRoot, rev string, cache blobCache) (Frame, error) {"
    evidence: internal/core/capture/reframe.go:418 — "func frameHistory(repoRoot string, cache blobCache, mainline bool) (*frameWalk, error) {"
  - the join to the occasion is operator-asserted with one check, that the occasion predates the rewrite
    evidence: internal/core/capture/reframe.go:766 — "func requirePredates"
    evidence: internal/core/capture/reframe_test.go:748 — "func TestReframeRefusesAnOccasionCommittedAfterTheRewrite"
  - a ground below the substance floor refuses, and the ground is redacted
    evidence: internal/core/capture/reframe.go:496 — "ground, redacted, degraded, err := requireFreeGrounds(repoRoot, "reframe", req.Grounds)"
    evidence: internal/core/capture/reframe_test.go:672 — "func TestReframeHoldsTheGroundToTheFloor"
  - the capture surface page documents the verb and its three halves
    evidence: commands/capture.md:508 — "## Record a reframe"
    evidence: commands/capture.md:529 — "A reframe is written in one of three halves, and every render names which:"
  - the family is warm and excluded from every reading, with the manifest asserting the exclusion
    evidence: internal/core/reading/include.go:72 — "It goes 1.8.0 to 1.9.0 with the reframe record: Exclusions gains the row"
- diverged: (none)
- missing: (none)

Scope-condition dispositions:
- cond-2609020626047674 — survived: FrameSurfaces fixes the three paths and the history walk is limited to them; ConstrualFingerprint refuses a framing chapter without exactly one Construal section, so a frame living elsewhere is refused rather than misread
  evidence: internal/core/capture/reframe.go:54 — "FrameSurfaces"
  evidence: internal/core/capture/reframe.go:257 — "func frameSurfacePaths() []string {"
  evidence: internal/core/capture/reframe_test.go:174 — "func TestConstrualFingerprintRefusesAChapterWithoutTheSection"
- cond-2609020626049770 — survived: the verb reads HEAD, refuses an uncommitted rewrite without --open, writes the before half from HEAD's triple under --open, completes it with --complete once HEAD differs, and every result names its half
  evidence: internal/core/capture/reframe.go:521 — "result.Half, result.Before = ReframeHalfOpen, head"
  evidence: internal/core/capture/reframe_test.go:444 — "func TestReframeRefusesUncommittedChangesWithoutOpen"
  evidence: internal/core/capture/reframe_test.go:469 — "func TestReframeOpensAHalfBeforeTheCommit"
  evidence: internal/core/capture/reframe_test.go:493 — "func TestCompleteFinishesAnOpenRecord"

## Grounds

- pursued: we expect a reframe record keyed to the committed fingerprints of the three frame surfaces, the construal section, the committed glossary terms and the committed scope, to make reframes countable and joinable to the reading that occasioned them without committing framing traces; a reframe the record cannot pair with its before and after fingerprints of the three surfaces, or a prior surface's text reaching the committed record, would show it wrong
