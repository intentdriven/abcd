# Phase 7 — Provenance ledger and cold reading

## Expectation

By the end of this phase, the record shows who originated a choice and on
what grounds, at the granularity of the conjecture rather than the release:
every commitment made through an abcd surface — an intent planned, a claim
typed, a conjecture pursued, an issue promoted or resolved — carries its
origin and its grounds, written by the command at the point of commitment
and never reconstructed afterwards. And an instrument exists that can read
the record cold, blind to the reasoning it is meant to test, and report where
the record diverges from the frame it reasons inside.

The frame is stated where a reader can find it: the brief's framing section
opens with the construal as it presently stands — the ledger's construal,
this cycle — and a widening reading reads against that statement and nothing
behind it; the construal's history stays on the local ledger side, per
[adr-55](../../decisions/adrs/0055-the-construal-stands-in-the-record-its-history-does-not.md).

The phase ends with the instrument built and **unrun**: running a reading is a
different cycle and a different day.

## Milestone

- Claim typing and scope-condition identity are stamped by `intent plan`,
  never hand-typed, and `intent ready` refuses an intent whose claims are
  untyped; the nullity-token grammar lives in the claim-recording gradient
  discipline and the gate stages forward-only.
- The origin and production-mode keys are command-written on every intent and
  issue write path, excluded from every reading by projection, and refused
  when hand-edited.
- Grounds are recorded at conjecture granularity on the selection surfaces
  (`intent ready`, `capture promote`, `capture resolve`) with the vocabulary
  pursued / deferred / declined, and refused on absence.
- The reading record and its disposition record exist in the issue tier
  (`issues/readings/<run-id>/`, `issues/dispositions/`), with the five-state
  disposition slate and availability by position; `capture promote` refuses an
  undispositioned detection; the outstanding report is report-only.
- The cold-reading input assembler produces a content-hashed manifest by
  positive inclusion at field granularity, context-isolated; the output
  contract's ingest validates a strict schema and the supply regime; the four
  reading definitions share one byte-identical core.
- The read-block eval and the amnesia eval pass in CI.
- The scribe context exists as a definition and a written protocol; the step-2
  admission record schemas exist, hand-enforced.
- The `lapse` category holds the cycle's pre-tooling entries and whatever the
  build generated; the record carries a worked example of every new field,
  populated by use.

## Phase Acceptance

> _Roll-up acceptance per [adr-9 amendment](../../decisions/adrs/0009-phase-as-product-layer.md). Each bullet asserts an emergent, cross-intent truth._

- **Given** an intent shipped through this phase's surfaces, **when** a
  reader with no prior context inspects its record, **then** they can tell who
  originated each claim, on what grounds it was pursued, and which scope
  conditions were met — from the repository alone.
- **Given** the assembled input of a reading, **when** it is compared with the
  local ledger side, the transcript store, and the instrument's own prior
  outputs, **then** none of them is present — asserted on fields by the
  read-block eval, not by inspection.
- **Given** the same object set assembled twice at the same commit, **when**
  the two inputs are compared with the manifest excluded, **then** they are
  byte-identical — the amnesia property the closing run depends on.
- **Given** the cycle's record, **when** its lapse log is read, **then** every
  skipped recording step is disclosed as a `lapse` entry made at the point of
  the lapse, never reconstructed.
- **Given** the phase's exit, **when** the readings directory is inspected,
  **then** no reading has run.

## Scope

**Intents:** [itd-177](../../intents/shipped/itd-177-an-intent-s-claims-are-typed-and-its-scope-conditions-keep-t.md), [itd-178](../../intents/shipped/itd-178-every-record-written-through-a-command-carries-its-origin-an.md), [itd-179](../../intents/shipped/itd-179-the-reasoning-behind-what-was-pursued-no-longer-evaporates-a.md), [itd-180](../../intents/shipped/itd-180-a-cold-reading-s-findings-land-as-reading-records-and-the-re.md), [itd-181](../../intents/shipped/itd-181-a-shipped-intent-s-scope-conditions-are-dispositioned-by-the.md), [itd-182](../../intents/shipped/itd-182-the-record-s-discipline-failures-are-themselves-recorded-a-l.md) (record side, track A); [itd-183](../../intents/shipped/itd-183-the-cold-reading-sees-exactly-what-the-assembler-passes-posi.md), [itd-185](../../intents/shipped/itd-185-one-ingest-verb-validates-every-cold-reading-output-includin.md), [itd-184](../../intents/shipped/itd-184-four-cold-reading-definitions-one-blindness-core-each-positi.md) (the
instrument bundle, track B, one shared spec); [itd-186](../../intents/shipped/itd-186-the-read-block-eval-falsifies-the-firewall-planted-warm-cont.md), [itd-187](../../intents/shipped/itd-187-amnesia-is-a-repository-property-proven-by-an-eval-the-same.md) (evals, track C);
[itd-188](../../intents/shipped/itd-188-machine-assistance-in-maintaining-the-ledger-without-any-con.md) (scribe context, track D); [itd-189](../../intents/shipped/itd-189-what-the-widening-reading-proposes-is-admitted-or-declined-o.md) (step-2 admission records,
schema only). **Disciplines:** [itd-190](../../intents/disciplines/itd-190-the-claim-recording-gradient-an-intent-s-three-claim-kinds-c.md) (claim-recording gradient),
[itd-191](../../intents/disciplines/itd-191-the-selection-criteria-are-a-declared-recorded-discipline-a.md) (selection criteria). Receives the framing chapter
[itd-143](../../intents/drafts/itd-143-the-brief-gains-a-framing-chapter-under-01-product-the-macro.md)
and the cold-reading surface
[itd-86](../../intents/drafts/itd-86-cold-reading-surface.md), which this
phase realises.

**Mechanism and content are distinct.** The mechanism (schemas, commands,
gates, evals) is engineering: reviewable and reworkable. The content — the
recorded reasoning — is populated only by actual use during the build and never
reconstructed; a defect in the mechanism is fixed, a defect in the content is
disclosed (decision log, 2026-08-28).

**Grounds extend the ADR family rather than duplicating it.** The ADR family
holds decision-granularity grounds; the ledger adds conjecture granularity as a
grounds argument on the selection surfaces — one canonical primitive, no
parallel store.

## Maps against

- **Brief:** `01-product/06-framing.md` (the construal home);
  `02-constraints/03-invariants.md` invariants 14 and 15;
  `04-surfaces/05-intent.md` and `04-surfaces/06-capture.md` (the surfaces the
  keys and grounds land on).
- **ADRs:** adr-50 as refined by adr-55 (record admissibility and reader
  access); adr-51 (mechanism claims and scope conditions, which claim typing
  enforces); adr-45 (ids minted by the verb, never by hand).
- **Disciplines:** the claim-recording gradient and the selection criteria
  (this phase); itd-84 (decomposition) and itd-81 (judge calibration) as the
  existing protocols the filing follows.

## Dependency rationale

- **Runs after Phase 3's record surfaces and Phase 2's ledger.** Claim
  typing, grounds, and the origin keys extend `intent plan` / `intent ready`
  and `capture`; the reading record joins the issue tier. Nothing here is a
  new front door.
- **The construal files first.** The record must show frame first, then
  commitments — the sequence the ledger exists to protect — so adr-55 and the
  framing section land before any intent of this phase is planned.
- **The instrument is sequential internally; the record side is not.**
  Assembler → output contract → definitions is a hard chain; the evals need
  the assembler; the record-side intents parallelise and merge in dependency
  order onto the integration branch.
- **No reading runs in this phase.** The opening run and the closing run are a
  later cycle; this phase ends with the instrument built and unrun, and a
  clean opening run is later recorded as a run with an empty item set.

## Open questions

- Whether `held` is available at the widening position or a deferred
  configuration belongs to the selection vocabulary's `deferred` — deferred by
  the facilitator to the first widening run's dispositions.
- Where the 150-word outward-facing form of the construal files — held on the
  local ledger side until the product thinker assigns a home.
- Whether "ledger" needs a glossary term to separate the provenance ledger from
  the issue ledger — flagged by the adversarial reviews, not ruled.
- The [HAND] owner for the step-2 admission records — to be named when the
  workstream's planning cycle phase runs. Nothing is owed before then: a
  widening reading has to run before a proposal exists to admit, and no reading
  runs this cycle.
