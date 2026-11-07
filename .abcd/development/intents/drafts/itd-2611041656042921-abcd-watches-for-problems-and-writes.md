---
id: itd-2611041656042921
slug: abcd-watches-for-problems-and-writes
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610141646469491]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# abcd notes what it would have taken and the person approves

## Press Release

> abcd watches for problems and writes down what it would have taken on, and does nothing until the person approves. Frank, an SRE, finds a test failed on the main branch at three in the morning and a note saying abcd would have taken it, with a link to the failure. One word from Frank turns the note into an ordinary issue that a drain can pick up. "Before I let it near anything I could see how often its notes were right," said Frank. "And a note can't start work on its own, so nobody outside can steer it through one."

## Why This Matters

A problem waits until someone files it, and handing abcd the power to act on
problems by itself would be a large step taken with no evidence about how good
its judgement is. Writing down what it would have taken gives that evidence at
no risk: nothing happens until the person approves a note, and the list shows
over time how often the notes were right.

The product thinker ruled on 2026-11-04 that each note needs their approval, and
that an approved note becomes an ordinary issue. Everything after that is the
existing path: a drain the person started can pick the issue up.

## Mechanism

We expect the share of notes the person approves to show whether abcd's
judgement about what to take on can be trusted, because each note is judged by
the person before anything happens. We expect a note to give an outsider no way
to steer abcd, because it carries no text the outsider wrote and starts nothing.

## Scope Conditions

- Holds for signals abcd reads from the forge and CI first, where the need is on the record; a chat channel is out of scope until a need for one shows.
- Holds where a note carries only the kind of event, its ids and a link, so a person or an agent reads the content under the existing rules for untrusted text.
- Holds on the computer that wrote the note, until the list is shared some other way.

## Acceptance Criteria

- **Given** a test fails on the main branch, **when** abcd sees it, **then** a note records the kind of event, its ids and a link, and no model runs and no work starts.
- **Given** an event with text written by someone outside the project, **when** a note is written, **then** none of that text is in the note.
- **Given** a note, **when** the person approves it, **then** it becomes an ordinary issue on the ledger.
- **Given** an unapproved note, **when** a drain runs, **then** the drain does not take it.
- **Given** a note the person declines, **when** the same event is seen again, **then** no second note is written for it.
- **Given** notes that have been judged, **when** the person lists them, **then** they see how many were approved and how many declined.

## Open Questions

- Capture refuses an issue with no remedy, and an approved note has none yet. Who writes it, and when (technical facilitator).
- What watches for events: the same timer as the dead-run check, or the watcher of itd-2610141646469491 (technical facilitator).
- Where the list lives if a person works across computers, and how long notes are kept (technical facilitator).
- What evidence would ever justify letting a run take notes without approval. This intent does not grant that (product thinker, later).

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
