---
id: itd-2609212137128014
slug: abcd-lab-mechanises-the-lab-conventions-three-hand-run
spec_id: spc-2609212141418943
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-22]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-75, itd-59]
related_adrs: [adr-2609212115255771]
---

# abcd lab mechanises the lab conventions three hand-run experiments proved

## Press Release

> **`abcd lab` mints, preflights, records, sweeps and harvests a lab, and the procedure the labs converged on becomes a discipline record.**
>
> "Three labs, and by the third the procedure returned SHIP with zero findings, because every amendment had been written down and the scaffolding had been rebuilt by hand each time," said a technical facilitator reading the capstone. "Now the verb builds the scaffolding, the procedure is a record, and no rule can be forgotten under pressure."

## Why This Matters

Between 31 August and 1 September eight labs ran under `~/.abcd/lab/`, three of them the core series; the capstone's series review shows review churn falling from four rounds with three application failures to one round to SHIP as the procedure accumulated amendments, and a draft intent for the verb family was written as evidence and never filed. The recording model (evidence at operator level, knowledge through ceremony, pointers in the local tier) held across the runs. Ruled 2026-09-21: file it from the capstone's text; the four product findings the series left unfiled are captured on the same branch.

## Mechanism

We expect a mechanised lab to reproduce the third lab's SHIP on its first run, because the variable that moved across the series was the procedure and the procedure is what the verb encodes; shown wrong if a lab run under the verb needs more review rounds than lab 3 did.

## Scope Conditions

- Holds on a machine whose lab store is `~/.abcd/lab/` keyed as the other machine-scoped stores are; the repository never holds lab evidence. <!-- cond: cond-2609212141418357 -->
- Holds while a real-session smoke stage is available: offline suites passed while the plugin was unloadable by the host. <!-- cond: cond-2609212141414079 -->

## What's In Scope

- **The verb family** `abcd lab mint | preflight | record | sweep | harvest`: the home minted with its registry entry and snapshot pin; the preflight artefact (harness isolation, dual-binary vintage); probe-record scaffolding; the retraction sweep (grep the pattern, not the instance); harvest assembly against the lifeboat's section shape.
- **The procedure as a discipline record**: the amendments across the three chains, present tense, host-agnostic.
- **The recording model** as the recorded convention: evidence at `~/.abcd/lab/`, knowledge through ceremony, pointers in the local tier; the never-in-repo rules.
- **Halt-and-record on a gate refusal** as a lab rule the verb enforces.
- **No cost claim**: the series could not measure token cost; the verb records what the runner reports and claims nothing more.

## What's Out of Scope

- Auto-merge for lab filings.
- Labs as a grouping of the record.
- A next-labs runner; the menu stays a note.

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that filed and planned this intent (adr-2609212115255771 records the vocabulary rulings it rests on):

1. Filed from the capstone's draft with its ten evidence items (ruled 2026-09-21).
2. Auto-merge for lab-derived work stays out; a lab's findings are captured and drained like any other.

## Open Questions

_None open._

## Acceptance Criteria

- **Given** `abcd lab mint <question>`, **when** it runs, **then** a lab home exists under the machine-scoped lab store with a registry entry, the snapshot pin and the lifecycle's sections scaffolded, and nothing is written into the repository.
- **Given** `abcd lab preflight`, **when** it runs, **then** the harness-isolation and dual-binary checks are written as an artefact, and a failed check halts the lab naming it.
- **Given** a lab's corrections, **when** `abcd lab sweep` runs, **then** every instance of a retracted pattern is listed and an unapplied correction fails the sweep.
- **Given** a finished lab, **when** `abcd lab harvest` runs, **then** the harvest is assembled in the lifeboat's section shape with the probe records cited, and its product findings are listed as capture candidates.
- **Given** a gate refusal during a lab, **when** it occurs, **then** the lab halts and records it as a finding rather than adapting around it.
- **Given** the discipline record, **when** it is read, **then** it carries the procedure's amendments in present tense, host-agnostic.

## Audit Notes

<!-- abcd-review: OWED receipt=rcp-2e848c471a09 -->
Fidelity review OWED (receipt rcp-2e848c471a09).

## Grounds

- pursued: three hand-run labs proved the procedure and paid for its scaffolding three times; we expect the verb to reproduce lab 3's SHIP on the first mechanised run; shown wrong if it needs more review rounds than lab 3 did
