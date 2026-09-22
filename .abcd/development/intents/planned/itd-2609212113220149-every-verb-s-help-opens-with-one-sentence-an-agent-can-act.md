---
id: itd-2609212113220149
slug: every-verb-s-help-opens-with-one-sentence-an-agent-can-act
spec_id: spc-2609212139583822
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-146]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-172]
related_adrs: [adr-2609212115255771]
---

# Every verb's help opens with one sentence an agent can act on

## Press Release

> **Every verb's help opens with one sentence, does / writes / refuses, identical on the list, the verb's `--help` and its page.**
>
> "I read one line per verb before I decide whether to call it," said a technical facilitator watching an agent choose. "When that line says what the verb does, what it writes and when it refuses, the agent calls the right one. When it says 'manage things', it grep's the binary."

## Why This Matters

Fifty-three verbs and sub-verbs carry a short line today, of uneven shape, and nothing holds them to a form or keeps the three places they appear in agreement. The field's guidance for both people and agents is the same: one concise, actionable line per command, in one voice. Ruled 2026-09-21: does, writes, refuses, in that order, under the repository's writing-style guide, generated from one source.

## Mechanism

We expect an agent choosing a verb from its sentence to call the right one more often and to stop reading the binary for verbs, because the sentence answers the three questions an agent asks before a call; shown wrong if agent transcripts after it ships still show a verb called for what it does not do.

## Scope Conditions

None stated.

## What's In Scope

- **The form**: one sentence per verb and sub-verb naming what it does, what it writes (or "writes nothing"), and when it refuses, in that order, under `docs/reference/writing-style.md`.
- **One source**: the sentence lives in the surface manifest and is rendered onto the command list, the verb's own `--help` and its plugin page, byte-identical.
- **The test**: walks every visible verb; fails on a missing clause, a length over the declared cap, or a difference between the three places.
- **The agent block** (itd-146) shows the same sentences.
- **The lint**: the docs-lint writing-style checks run over the sentences.

## What's Out of Scope

- The long help body per verb.
- Examples per verb (the page's).

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that filed and planned this intent (adr-2609212115255771 records the vocabulary rulings it rests on):

1. Does / writes / refuses, in that order, identical in three places, under the writing-style guide.

## Open Questions

_None open._

## Acceptance Criteria

- **Given** any visible verb or sub-verb, **when** its sentence is read, **then** it names what the verb does, what it writes or that it writes nothing, and when it refuses, in that order, and obeys the writing-style guide.
- **Given** the command list, the verb's `--help` and its plugin page, **when** the sentence is compared across them, **then** it is byte-identical, rendered from the surface manifest.
- **Given** a verb whose sentence lacks a clause, exceeds the cap or differs between places, **when** the test runs, **then** it fails naming the verb and the defect.
- **Given** `--help --agent`, **when** the agent block renders, **then** each line is the verb's sentence.
- **Given** the docs lint, **when** it runs, **then** the sentences are checked as any page is.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: the run adds five verbs and the per-verb lines today are fifty-three sentences of uneven shape; we expect an agent to choose from the sentence and stop grepping the binary; shown wrong if transcripts still show a verb called for what it does not do
