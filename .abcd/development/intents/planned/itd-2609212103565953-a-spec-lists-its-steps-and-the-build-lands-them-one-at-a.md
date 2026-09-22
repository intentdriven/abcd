---
id: itd-2609212103565953
slug: a-spec-lists-its-steps-and-the-build-lands-them-one-at-a
spec_id: spc-2609212138246060
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609201916151817, itd-2609211116005482]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-2609211913453478]
related_adrs: [adr-2609212115255771]
---

# A spec lists its steps, and the build lands them one at a time

## Press Release

> **A spec lists its steps, and `abcd build` lands them one at a time, each its own lane and pull request.**
>
> "The implement verb was three lanes in the spec's order, and nothing had a word for the three," said a technical facilitator reading the run file. "Now the spec says its steps, the loop builds them in that order, and a step that does not fit the cut leaves the rest as the remainder. No new record family, no task tracker: a list in the design record."

## Why This Matters

The record's only unit below a spec was another spec, minted as a remainder, and the build machinery's scope condition says a lane too large for one context is split by the spec, not by the loop. The run file already split the implement verb into three lanes with no name for the pieces. The field's answers are Shape Up's scopes (pieces inside the pitch) and Kubernetes' in-record graduation; a child record family (Linear's sub-issues) adds folders and verbs the vocabulary is shedding. Ruled on 2026-09-21 (adr-2609212115255771, decision 4): a section in the spec, called steps, sharing the loop's own word.

## Mechanism

We expect specs split into named steps to land with fewer fix rounds and smaller lane contexts than the pilot's 349k-token lane, because a step is what one implementer can hold and land; shown wrong if stepped specs show no smaller lane contexts than unstepped ones of the same footprint.

## Scope Conditions

None stated.

## What's In Scope

- **The section**: `## Steps` in the spec, an ordered list, each step with a title and its footprint (packages, tests); the spec template seeds it empty at `intent plan`; a spec without steps is one step.
- **The loop**: `abcd build` reads the list and runs one lane per step in order, each landing as its own pull request; the next step's lane starts after the previous has merged.
- **The remainder**: a step that does not fit the cut leaves the unfinished steps in the remainder spec `spec close --remainder` mints.
- **The brief**: a lane's brief names the step it builds and the steps before it; the run record lists steps as it lists lanes.
- **One word**: `implement step` (the loop's step interface) and this section share the word, and the command page says so.

## What's Out of Scope

- The build proposing a split (the author writes the steps).
- A task record family.
- Reordering steps mid-run.

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that filed and planned this intent (adr-2609212115255771 records the vocabulary rulings it rests on):

1. The unit below a spec is a section called steps, not a record family (adr-2609212115255771, decision 4).
2. The spec's author writes the steps; a spec without any is one step.
3. A step that does not fit the cut leaves the rest as the remainder.

## Open Questions

_None open._

## Acceptance Criteria

- **Given** `abcd intent plan` mints a spec, **when** the stub is read, **then** it carries an empty `## Steps` section beside `## Footprint`, and a spec with no steps listed is built as one step.
- **Given** a spec with three steps, **when** `abcd build` runs it, **then** three lanes run in order, each landing as its own pull request, and the second starts only after the first has merged.
- **Given** a step that does not fit the cut, **when** the spec is closed with `--remainder`, **then** the remainder spec carries the unfinished steps.
- **Given** a lane for a step, **when** its brief is rendered, **then** it names the step and the steps before it, and the run record lists the steps.
- **Given** the command page, **when** it is read, **then** `implement step` and the spec's steps are described as one word for one thing.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: the run file already builds the implement verb as three lanes in the spec's order with no word for the pieces; we expect named steps to land with smaller contexts and fewer fix rounds; shown wrong if stepped specs show no smaller lane contexts
