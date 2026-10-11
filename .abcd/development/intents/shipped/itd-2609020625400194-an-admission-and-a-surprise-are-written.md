---
id: itd-2609020625400194
slug: an-admission-and-a-surprise-are-written
spec_id: spc-2609020626040342
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-189, itd-180, itd-185]
severity: minor
impact: additive
origin: researcher-authored
production_mode: dictated-and-formatted
---

# An admission and a surprise are written by a verb, and the order the design fixes is a refusal

Typed links: `builds_on` [itd-189](itd-189-what-the-widening-reading-proposes-is.md) (the admission and surprise schemas), [itd-180](itd-180-a-cold-reading-s-findings-land-as.md) (dispositions), [itd-185](itd-185-one-ingest-verb-validates-every-cold.md) (the ingest verb); `refines` [itd-180](itd-180-a-cold-reading-s-findings-land-as.md) (admission is the `accepted` disposition plus the admission record, written as one act, flagged for the maintainer).

## Press Release

> **What the widening reading proposes is admitted or declined through a command.** `abcd capture admit <rdi-N> --grounds "<why>"` records an admission as one act: the item's `accepted` disposition, carrying the grounds, and the admission record that joins it to the run's candidate set. `abcd capture surprise --occasioned-by <id> "<what>"` writes one surprise entry as its own record. Declining stays what it already is, a disposition in the `declined` state. An admission is refused until the comparative reading over that run has been ingested or recorded as not exercised, because the design characterises first and admits second. `abcd <id>` dispatches on `adm-N` and `srp-N`, which no record dispatch reaches today, and the outstanding report says at any moment which widening items carry neither an admission nor a `declined` or `held` disposition.

> "Every proposal I took into the candidate set carries the reason I took it, written at the moment I took it, and every one I passed over says I passed over it," said a facilitator who runs the loop for a researcher-developer. "The verb refuses to let me admit before the characterisation is in, so the record shows the order happened the way the design says it must."

## Why This Matters

[itd-189](itd-189-what-the-widening-reading-proposes-is.md) places the recording burden on admission: rejecting a proposal costs nothing epistemically, and admitting one into the candidate set is where the frame is engaged. Its scope text says the enforcement is hand-run in Iteration 1, no reading running to produce proposals, and enforced at the command in Iteration 2. Iteration 1 delivered the schemas: the admission family, the surprise family, the store layout and the gate that refuses a blank ground on a committed record. It delivered no verb, and its fidelity verdict named filing the enforcement intent as "the concrete next step this verdict asks for".

The gate that reads committed records has two open defects that a verb closes for the records it writes: its blank refusal decides on literal spellings rather than on the YAML null and empty class ([iss-2608301808198621](../../../work/issues/resolved/iss-2608301808198621-isabsentvalue-decides-on-literal-strings.md)), and a trailing comment on a key defeats every spelling it refuses ([iss-2608301744268001](../../../work/issues/resolved/iss-2608301744268001-a-trailing-comment-on-a-frontmatter-key.md)). Both belong where the scanner lives and are fixed there, in their own change; a verb that cannot write a blank is the other half.

The ordering is ruled: Step 2 precedes Step 4, and admission is performed after the comparative reading. Under the rule that commands are the write path, a fixed order is a refusal in the verb, not a sentence in a protocol.

## Decisions flagged for the maintainer

- **Admission is one act with two records.** itd-180's ruling is that at the widening position acceptance is admission, and itd-189 shipped an admission record carrying grounds. Today an item with an `accepted` disposition is still reported as unadmitted. This intent makes `capture admit` write the `accepted` disposition and the admission record together under the ledger lock, the disposition carrying the grounds; where `accepted` already stands, it writes the admission record alone. This refines itd-180 and is flagged rather than assumed.

## What's In Scope

- **`abcd capture admit <rdi-N> --grounds "<text>"`**, writing the `accepted` disposition through the existing disposition validator and `admissions/<run-id>/adm-N.md` with the proposal, the run and the grounds, minted on the timestamp seam, under the ledger lock. It refuses an item that is not a widening item, an item already admitted, an item carrying a standing disposition other than `accepted`, and a ground that does not meet the substance floor the grounds primitive already applies.
- **The ordering as one gate in the shared disposition writer.** At the widening position, any disposition (accepted, declined or held) and any admission refuses until a committed comparative run names the item's run, and names what it is waiting for. The gate lives in the writer every verb routes through, so neither `capture disposition` nor the scribe's ingest can land an acceptance before characterisation.
- **`abcd capture surprise --occasioned-by <rdi-N|adm-N|dsp-N> "<text>"`**, writing `surprises/srp-N.md` as its own record, never as a field on a disposition, and refusing an occasion that does not resolve.
- **Dispatch.** `abcd adm-N` and `abcd srp-N` report the record and its joins. The dispatcher covers issues, intents, specs and ADRs today; the reading families (`rdi`, `dsp`, `rdg`) stay outside it and are named as out of scope.
- **The outstanding report** names every widening item of a run that carries neither an admission nor a `declined` or `held` disposition, so the admitted-against-declined count is a query rather than an inspection.
- The plugin surface page for capture documents the two verbs and the ordering.

## What's Out of Scope

- The scanner fix for absence as a class. It is two recorded issues and ships in its own change with its own trailers.
- A change to the disposition vocabulary. Declining is the `declined` state and stays so.
- Enforcing that a session ended. The report answers the question whenever it is run.
- The comparative channel itself, which is its own intent; this verb only reads whether that run's outcome exists.
- Dispatch on `rdi-N`, `dsp-N` and `rdg-N`, which spc-67 records as a residual and which this intent does not close.

## Mechanism

We expect a verb that refuses a blank ground to make the admission asymmetry legible because the count that evidences ownership is admitted against declined, and a record that can only be written with a stated reason turns that count into something a query answers rather than something a reader reconstructs. It fails if the verb is bypassed by hand-written records, which the gate exists to catch and which the scanner fix makes it catch as a class.

## Scope Conditions

- The ordering gate reads the comparative run the channel commits, which the comparative-channel intent defines (an empty item set records the not-exercised outcome), and the channel lands in Phase A before this verb. <!-- cond: cond-2609020626047113 -->
- Grounds on admission use the same substance floor as every other grounds primitive. A different floor for admissions is a ruling this intent does not make. <!-- cond: cond-2609020626042968 -->
- The verb is written for the widening position only, because admission is the widening reading's warm act by design. <!-- cond: cond-2609020626040151 -->

## Acceptance Criteria

- **Given** a widening item with no admission and no disposition, and a comparative run ingested over its run, **when** `capture admit` runs with a ground meeting the floor, **then** an `accepted` disposition and one admission record exist naming the item and the run, and a second `capture admit` on the same item refuses.
- **Given** a widening item with a standing `accepted` disposition and no admission, **when** `capture admit` runs, **then** one admission record is written and the disposition is untouched.
- **Given** the same item before any comparative run names its run, **when** `capture admit` or `capture disposition` runs on it, **then** it refuses and names what it is waiting for.
- **Given** a widening item with a standing `declined` disposition, **when** `capture admit` runs, **then** it refuses and names the disposition.
- **Given** `capture admit` with a blank, whitespace or degenerate ground, **when** it runs, **then** it refuses and nothing is written.
- **Given** `capture surprise` with an occasion that resolves, **when** it runs, **then** one surprise record exists as its own file, and no disposition record was touched.
- **Given** a run with four widening items of which one is admitted, one declined and one held, **when** the outstanding report runs, **then** it names the fourth item, which carries neither an admission nor a `declined` or `held` disposition, and no other.
- **Given** `abcd adm-N` or `abcd srp-N`, **when** it runs, **then** it reports the record and the records it joins to.

## Prior Art

- [itd-189](itd-189-what-the-widening-reading-proposes-is.md) and its spec (the schemas), [itd-180](itd-180-a-cold-reading-s-findings-land-as.md) (dispositions), adr-56 (absence as a class, ruled for the exclusion floor).
- The cold-reading rulings of 2026-08-28 in the decision log.

## Open Questions

None beyond the flagged decision above.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-77c7c87559c2 -->
Fidelity review — receipt rcp-77c7c87559c2 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:74f6cc68397cb66a0c07dd9ac2c494d47c595790bfdf769e0d66af279d595483
Input attestations: diff:internal/core/capture/{admit,surprise,reading,itemfate}.go, internal/core/lint/readingoutstanding.go, internal/core/record/record.go, commands/capture.md at 4001d807 (git ls-tree -r)@sha256:22d51985481877746861848e96f958139e71e639b95589d4c2f753fe44bcac92;

Acceptance rollup: MET 8 · MET_WITH_CONCERNS 0 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: Admit writes the accepted disposition through the shared writer and the admission record under one ledger lock, both naming the item and the run, and a second Admit refuses on the standing admission; TestAdmitWritesBothRecordsAndRefusesTwice passes at BASE
  evidence: internal/core/capture/admit.go:124 — "written, err := writeDispositionLocked(repoRoot, issuesRoot, head, DispositionRequest{"
  evidence: internal/core/capture/admit.go:114 — "if len(fate.Admissions) > 0 {"
  evidence: internal/core/capture/admit_test.go:82 — "func TestAdmitWritesBothRecordsAndRefusesTwice"
- ac-2 — MET: with exactly one standing disposition Admit reads it through requireStandingAcceptance and writes the admission alone, and the test asserts the disposition's bytes are untouched
  evidence: internal/core/capture/admit.go:131 — "case 1:"
  evidence: internal/core/capture/admit.go:187 — "func requireStandingAcceptance(issuesRoot, item, id, ground string)"
  evidence: internal/core/capture/admit_test.go:115 — "func TestAdmitWritesTheAdmissionAloneOverAStandingAcceptance"
- ac-3 — MET: requireCharacterised sits in writeDispositionLocked, the writer both Disposition and Admit route through, and the admission-alone branch calls it too; the refusal names the run and the comparative reading it waits for, and both TestDispositionRefusesBeforeTheComparativeRun and TestAdmitRefusesBeforeTheComparativeRun assert ErrNotCharacterised with nothing written
  evidence: internal/core/capture/reading.go:454 — "if err := requireCharacterised(repoRoot, head); err != nil {"
  evidence: internal/core/capture/reading.go:497 — "func requireCharacterised(repoRoot string, head itemHead) error {"
  evidence: internal/core/capture/admit.go:138 — "if err := requireCharacterised(repoRoot, head); err != nil {"
  evidence: internal/core/capture/reading_test.go:581 — "func TestDispositionRefusesBeforeTheComparativeRun"
  evidence: internal/core/capture/admit_test.go:180 — "func TestAdmitRefusesBeforeTheComparativeRun"
- ac-4 — MET: a standing disposition in any state but accepted refuses naming the disposition id and its state; TestAdmitRefusesAStandingNonAcceptance covers rejected, declined and held
  evidence: internal/core/capture/admit.go:198 — "if state := asString(fm["state"]); state != issueschema.DispositionAccepted {"
  evidence: internal/core/capture/admit_test.go:219 — "func TestAdmitRefusesAStandingNonAcceptance"
- ac-5 — MET: the ground is settled through requireFreeGrounds (redact, fold, grounds.ValidateText) before the ledger is touched, so a blank, whitespace or degenerate ground refuses with nothing written; TestAdmitRefusesADegenerateGround asserts the ledger digest is unchanged
  evidence: internal/core/capture/admit.go:76 — "ground, redacted, degraded, err := requireFreeGrounds(repoRoot, "admit", req.Grounds)"
  evidence: internal/core/capture/admit.go:292 — "if err := grounds.ValidateText(folded); err != nil {"
  evidence: internal/core/capture/admit_test.go:263 — "func TestAdmitRefusesADegenerateGround"
- ac-6 — MET: Surprise writes surprises/srp-N.md as its own file after resolving the occasion under the lock, and TestSurpriseIsItsOwnRecord asserts the dispositions tree digest is unchanged; the CLI test dispatches abcd srp-N to it
  evidence: internal/core/capture/surprise.go:99 — "path := filepath.Join(dir, id+".md")"
  evidence: internal/core/capture/surprise_test.go:38 — "func TestSurpriseIsItsOwnRecord"
  evidence: internal/surface/cli/capture_admit_test.go:113 — "func TestCaptureSurpriseRequiresAnOccasion"
- ac-7 — MET: ReadReadingOutstanding renders one WideningRun per run with the admitted/declined/held counts and the outstanding list; TestWideningRunSummaryNamesTheOutstandingItem builds exactly the four-item run of the criterion and asserts the fourth item alone is named, and the capture board renders the count
  evidence: internal/core/lint/readingoutstanding.go:140 — "type WideningRun struct {"
  evidence: internal/core/lint/reading_outstanding_test.go:1097 — "func TestWideningRunSummaryNamesTheOutstandingItem"
  evidence: internal/surface/cli/capture_admit_test.go:106 — "1 proposal(s): 1 admitted, 0 declined, 0 held, 0 outstanding"
- ac-8 — MET: the record dispatcher's IDRe admits adm and srp and routes them to describeAdmission (run, proposal, proposal_path, standing disposition) and describeSurprise (occasioned_by, occasion_path); TestDescribeAdmission and TestDescribeSurprise assert the joins and the CLI test dispatches a minted srp-N
  evidence: internal/core/record/record.go:36 — "var IDRe = regexp.MustCompile(`^(iss|itd|spc|adr|adm|srp|rfm)-[0-9]+$`)"
  evidence: internal/core/record/record.go:113 — "case "adm":"
  evidence: internal/core/record/record_test.go:786 — "func TestDescribeAdmission"
  evidence: internal/core/record/record_test.go:818 — "func TestDescribeSurprise"
  evidence: internal/surface/cli/capture_admit_test.go:142 — "// And `abcd srp-N` dispatches to it."

Gap audit:
- honoured:
  - admission is one act with two records under the ledger lock, and where accepted stands the admission is written alone on the standing ground
    evidence: internal/core/capture/admit.go:94 — "err = withLedgerLock(repoRoot, issuesRoot, func() error {"
    evidence: internal/core/capture/admit_test.go:342 — "func TestAdmissionAndDispositionCarryOneGround"
  - the ordering gate lives in the shared writer, so neither capture disposition nor the scribe's ingest lands an acceptance before characterisation
    evidence: internal/core/capture/reading_test.go:677 — "func TestDispositionAndAdmitShareOneWritePath"
    evidence: internal/core/scribe/ingest_test.go:385 — "if !errors.Is(err, capture.ErrNotCharacterised)"
  - a comparative run committed with an empty item set (not exercised) satisfies the gate
    evidence: internal/core/capture/admit_test.go:208 — "func TestAdmitProceedsOnAnEmptyComparativeRun"
  - both-or-neither: a failed admission write removes the disposition the act wrote
    evidence: internal/core/capture/admit.go:157 — "if rmErr := removeContained(ledgerBase(repoRoot, issuesRoot), dispPath); rmErr != nil {"
    evidence: internal/core/capture/admit_test.go:300 — "func TestAdmitRemovesTheDispositionWhenTheAdmissionWriteFails"
  - the plugin surface page documents both verbs and the ordering
    evidence: commands/capture.md:435 — "**At the widening position, characterise first and admit second.**"
    evidence: commands/capture.md:468 — "## Record a surprise"
  - surprise refuses an occasion outside rdi/adm/dsp or one that does not resolve, before anything is minted
    evidence: internal/core/capture/surprise.go:61 — "if !issueschema.ValidSurpriseOccasion(occasion) {"
    evidence: internal/core/capture/surprise_test.go:78 — "func TestSurpriseRefusesAnUnresolvedOccasion"
- diverged: (none)
- missing: (none)

Scope-condition dispositions:
- cond-2609020626047113 — survived: the gate probes ComparativeRunFor over the committed run records the comparative channel writes, an empty item set counts as not exercised, and the channel intent itd-2609020625407419 is in shipped/
  evidence: internal/core/capture/itemfate.go:145 — "func ComparativeRunFor(repoRoot, run string) (string, error) {"
  evidence: internal/core/capture/reading.go:501 — "comp, err := ComparativeRunFor(repoRoot, head.run)"
- cond-2609020626042968 — survived: admission grounds pass through grounds.ValidateText, the same floor every grounds primitive applies; no admission-specific floor exists
  evidence: internal/core/capture/admit.go:289 — "func requireFreeGrounds(repoRoot, verb, raw string)"
  evidence: internal/core/capture/admit.go:292 — "if err := grounds.ValidateText(folded); err != nil {"
- cond-2609020626040151 — survived: requireWidening refuses every other position by name and the ordering gate is keyed on the widening position alone; TestAdmitRefusesANonWideningItem covers it
  evidence: internal/core/capture/admit.go:174 — "func requireWidening(head itemHead) error {"
  evidence: internal/core/capture/admit_test.go:247 — "func TestAdmitRefusesANonWideningItem"

## Grounds

- pursued: we expect a verb that refuses a blank ground and refuses to admit before the comparative outcome exists to make the admitted-against-declined count a query and the ruled ordering a fact rather than a protocol; a hand-written admission the gate accepts with no ground, or an admission recorded before characterisation with every gate green, would show it wrong
