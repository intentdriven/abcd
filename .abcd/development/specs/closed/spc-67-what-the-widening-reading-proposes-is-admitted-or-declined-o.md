---
id: spc-67
slug: what-the-widening-reading-proposes-is-admitted-or-declined-o
intent: itd-189
---
# Step-2 admission records: schemas now, command enforcement next cycle

## Summary

spc-67 delivers [itd-189](../../intents/shipped/itd-189-what-the-widening-reading-proposes-is-admitted-or-declined-o.md)'s
record shapes for the widening reading's step 2: every admission into the
candidate set carries recorded grounds, every declined proposal carries a
disposition, and a surprise is its own entry rather than a field on either.

This cycle delivers **schemas, not commands**. No reading has run, so there is
nothing to write yet, and the intent scopes command enforcement to Iteration 2.
That leaves an obvious trap: a schema no code reads is dead scaffolding. It is
avoided by wiring the schemas to the gate that reads committed records rather
than to the command that will one day write them. A hand-written admission
record is validated by `record_schema` the moment it is committed, so
`make preflight` exercises every shape this spec declares from the day it lands.
The writing verb is the next cycle's; the refusal is this cycle's.

## Scope

In: the admission-grounds record shape and its store; the statement that a
declined proposal is spc-58's disposition record in its `declined` state and not
a second record type; the surprise entry's store and keying; the record-lint
wiring that makes all three refusable; the outstanding report's admission leg.

Out: everything under `## Out of scope`.

## Approach

### Three shapes, two of which already exist

The reuse rule does most of the work here. Of the three records itd-189 names,
only one is new.

**The admission-grounds record is new.** Family `adm`, minted through
`recordid.Minter.Mint` like every other record family in this workstream
([adr-45](../../decisions/adrs/0045-record-ids-are-timestamp-numeric-and-capture-stable.md)).
It carries `schema_version`, `id`, `run`, `proposal` (the widening item's
`rdi-N`), and `grounds` (free text, non-empty). It is filed at
`.abcd/work/issues/admissions/<run-id>/adm-N.md`, bucketed by run id exactly as
the reading store is, because an admission is meaningful only against the run
whose proposals it admits.

**The declined-proposal disposition is not new.** Ruling (19) (2026-08-28)
already reserves `declined` for the widening position, and
[spc-58](../closed/spc-58-a-cold-reading-s-findings-land-as-reading-records-and-the-re.md)
already delivers the disposition record, its `disposition_grounds` field and its
state-availability check. A second declined-proposal record type would be a
parallel store for a state the disposition vocabulary already holds. This spec
therefore declares no disposition schema at all; it states that the declined
proposal's record sits in `dispositions/rdi-N/` with `state: declined`, and it adds
the one thing that is genuinely missing: the report that notices a widening
proposal which is neither admitted nor declined.

**The surprise entry is declared once, in spc-58's family.** itd-180's own scope
reserves the surprise entry's schema "in this family now", populated in
Iteration 2, and itd-189 also lists it. The two intents describe one record.
The declaration lives with the reservation, in spc-58; spc-67 states its keying
and its separateness and declares nothing a second time. A surprise entry is
`srp-N`, filed at `.abcd/work/issues/surprises/`, carrying `occasioned_by`
naming whatever occasioned it (an `rdi-N` detection, an `adm-N` admission, a
consequence). It is never a field on a disposition and never shares a key with
one: the reading's output, the researcher's response and the surprise that
occasions abduction are three acts and three records.

### Wiring the schemas to the gate, not to a verb

`internal/core/issueschema` gains `AdmissionRequired`, `AdmissionKnown`,
`SurpriseRequired` and `SurpriseKnown`, in the same schema-as-data idiom the
issue record already uses: one definition, read by every gate that asks the
question. `internal/core/lint/schema.go` gains two store entries, `adm` at
`.abcd/work/issues/admissions` (bucketed by the run-id grammar spc-58
introduces) and `srp` at `.abcd/work/issues/surprises` (flat), each declaring
its `requiredFields` from `issueschema` rather than from a hand-copied list.

That is the whole of the enforcement this cycle, and it is real: an admission
record with a blank `grounds` is a blocker finding today, because
`checkRecordRequiredFields` treats an absent value and an empty one alike. The
`[HAND]` part is confined to who writes the file, not to whether anything checks
it.

### The outstanding report gains a leg, not a twin

spc-58 delivers `reading_outstanding`, a report-only rule pinned at `info` that
names every reading item with no keyed disposition. The widening position's
question is the same question with a wider answer set: a widening proposal is
answered by an `adm-N` record **or** by a `declined` disposition, and one with
neither is outstanding. That is one more branch inside the existing rule, not a
second rule. Adding a parallel `admission_outstanding` would put the same
judgement in two places, and the first divergence between them would be silent.

Its severity stays `info` and stays pinned in code for spc-58's reason: an
unanswered proposal must never block an unrelated push.

## Acceptance criteria mapping

| itd-189 criterion | How spc-67 satisfies it | Test |
|---|---|---|
| An admitted widening proposal carries grounds, and a blank is refused once command enforcement lands | The `adm` store declares `grounds` required, so `record_schema` refuses a blank at blocker severity from the day this lands. The command-side refusal is Iteration 2 and is named as such rather than claimed | `TestAdmissionRecordRequiresGrounds`, `TestAdmissionRecordRequiresProposal` |
| A declined widening proposal has its disposition on the ledger side by session end | The disposition is spc-58's record in its `declined` state, filed under the proposal's own id; the outstanding report names any proposal with neither an admission nor a decline | `TestWideningProposalWithoutAdmissionOrDeclineIsOutstanding`, `TestDeclinedDispositionSatisfiesTheAdmissionLeg` |
| A surprise entry is a distinct record from any disposition it relates to | Separate store, separate family prefix, and `occasioned_by` rather than a shared key, so a surprise can never be read as a disposition or overwrite one | `TestSurpriseRecordIsNotADisposition`, `TestSurpriseOccasionedByResolves` |

## Tests

Every case below is watched to fail before its change lands.

- `internal/core/issueschema/admission_test.go` :
  `TestAdmissionRequiredIsTheOneList` and `TestSurpriseRequiredIsTheOneList`
  assert the lint's required-field sets are the shared values, not restatements.
- `internal/core/lint/schema_test.go` :
  `TestAdmissionRecordRequiresGrounds` (blank and absent `grounds` both flagged),
  `TestAdmissionRecordRequiresProposal`,
  `TestAdmissionStoreBucketsByRun` (a run-id directory is a declared bucket; a
  directory that is not run-shaped is still reported),
  `TestSurpriseRecordIsNotADisposition` (a surprise id in the disposition store,
  and a disposition id in the surprise store, are each refused),
  `TestSurpriseOccasionedByResolves` (an `occasioned_by` naming no record in the
  corpus is a finding).
- `internal/core/lint/reading_outstanding_test.go` :
  `TestWideningProposalWithoutAdmissionOrDeclineIsOutstanding`,
  `TestDeclinedDispositionSatisfiesTheAdmissionLeg`,
  `TestAdmissionRecordSatisfiesTheAdmissionLeg`,
  `TestAdmissionLegSeverityIsInfoNotBlocker`.

The corpus fixtures are one well-formed admission record, one well-formed
surprise entry and one declined disposition, each of which the gate must pass:
a rule watched only failing is a rule that might refuse everything.

## Out of scope

- The commands that write these records. Iteration 2, by itd-189's own words;
  this cycle's records are hand-written and gate-checked.
- The disposition record, its vocabulary and its state-availability rule, all of
  which are spc-58's.
- The surprise entry's schema declaration, which is spc-58's reservation; this
  spec states its keying and separateness only.
- The widening reading's definition, its candidate set and the criteria by which
  a proposal is characterised. Those belong to the definitions bundle and to the
  selection-criteria discipline, and the criteria never arrive at invocation.
- Populating any of these records. No reading has run in this cycle.
- Dispatching `abcd <id>` on `adm-N` or `srp-N`, which shares spc-58's residual:
  the cited-id grammar covers the four id-bearing families only.
