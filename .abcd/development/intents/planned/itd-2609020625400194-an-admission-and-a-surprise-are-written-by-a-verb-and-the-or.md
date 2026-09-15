---
id: itd-2609020625400194
slug: an-admission-and-a-surprise-are-written-by-a-verb-and-the-or
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

Typed links: `builds_on` [itd-189](../shipped/itd-189-what-the-widening-reading-proposes-is-admitted-or-declined-o.md) (the admission and surprise schemas), [itd-180](../shipped/itd-180-a-cold-reading-s-findings-land-as-reading-records-and-the-re.md) (dispositions), [itd-185](../shipped/itd-185-one-ingest-verb-validates-every-cold-reading-output-includin.md) (the ingest verb); `refines` [itd-180](../shipped/itd-180-a-cold-reading-s-findings-land-as-reading-records-and-the-re.md) (admission is the `accepted` disposition plus the admission record, written as one act, flagged for the maintainer).

## Press Release

> **What the widening reading proposes is admitted or declined through a command.** `abcd capture admit <rdi-N> --grounds "<why>"` records an admission as one act: the item's `accepted` disposition, carrying the grounds, and the admission record that joins it to the run's candidate set. `abcd capture surprise --occasioned-by <id> "<what>"` writes one surprise entry as its own record. Declining stays what it already is, a disposition in the `declined` state. An admission is refused until the comparative reading over that run has been ingested or recorded as not exercised, because the design characterises first and admits second. `abcd <id>` dispatches on `adm-N` and `srp-N`, which no record dispatch reaches today, and the outstanding report says at any moment which widening items carry neither an admission nor a `declined` or `held` disposition.

> "Every proposal I took into the candidate set carries the reason I took it, written at the moment I took it, and every one I passed over says I passed over it," said a facilitator who runs the loop for a researcher-developer. "The verb refuses to let me admit before the characterisation is in, so the record shows the order happened the way the design says it must."

## Why This Matters

[itd-189](../shipped/itd-189-what-the-widening-reading-proposes-is-admitted-or-declined-o.md) places the recording burden on admission: rejecting a proposal costs nothing epistemically, and admitting one into the candidate set is where the frame is engaged. Its scope text says the enforcement is hand-run in Iteration 1, no reading running to produce proposals, and enforced at the command in Iteration 2. Iteration 1 delivered the schemas: the admission family, the surprise family, the store layout and the gate that refuses a blank ground on a committed record. It delivered no verb, and its fidelity verdict named filing the enforcement intent as "the concrete next step this verdict asks for".

The gate that reads committed records has two open defects that a verb closes for the records it writes: its blank refusal decides on literal spellings rather than on the YAML null and empty class ([iss-2608301808198621](../../../work/issues/resolved/iss-2608301808198621-isabsentvalue-decides-on-literal-strings-rather-than-the-yam.md)), and a trailing comment on a key defeats every spelling it refuses ([iss-2608301744268001](../../../work/issues/resolved/iss-2608301744268001-a-trailing-comment-on-a-frontmatter-key-defeats-every-blank.md)). Both belong where the scanner lives and are fixed there, in their own change; a verb that cannot write a blank is the other half.

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

- [itd-189](../shipped/itd-189-what-the-widening-reading-proposes-is-admitted-or-declined-o.md) and its spec (the schemas), [itd-180](../shipped/itd-180-a-cold-reading-s-findings-land-as-reading-records-and-the-re.md) (dispositions), adr-56 (absence as a class, ruled for the exclusion floor).
- The cold-reading rulings of 2026-08-28 in the decision log.

## Open Questions

None beyond the flagged decision above.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: we expect a verb that refuses a blank ground and refuses to admit before the comparative outcome exists to make the admitted-against-declined count a query and the ruled ordering a fact rather than a protocol; a hand-written admission the gate accepts with no ground, or an admission recorded before characterisation with every gate green, would show it wrong
