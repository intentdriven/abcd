---
id: itd-201
slug: every-question-abcd-s-agents-put-to-a-human-is-asked-one-at
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-84]
severity: major
impact: additive
origin: researcher-authored
production_mode: dictated-and-formatted
---

# Every question abcd's agents put to a human is asked one at a time, in plain language, with options that widen

## Press Release

> **A question from an abcd agent is now something a person can answer
> without reading the record.** Whenever an agent needs a human decision, at
> a planning interview, a decomposition routing, a grill, or any stop for a
> verdict, it asks one question at a time through the host's interactive
> question tool. Each question carries one sentence of context, one concrete
> example of what each answer would mean in practice, and options that widen
> rather than recommend: no starred default, no recommended label, and the
> null answer always offered. Deferral is recorded as an answer; silence is
> never consent. The rule ships as a rule domain that abcd installs into
> every repository it manages, so a session that never read the protocol
> still follows it from its first prompt.

> "I was handed five design questions in one message, numbered, each three
> lines of record vocabulary," said a product thinker who answers for what
> their team ships. "I could not tell what any of them was asking me to
> choose between. Now I get one at a time, in words I use, with an example,
> and I answer in a click."

## Why This Matters

The record already holds the two halves that matter for options: the
widen-options rule and the three drafts that fix how options are offered.
What it does not hold is how a question reaches a person at all. On
2026-09-01 the maintainer received the outstanding design decisions as a
numbered list inside a long message and could not answer them; the same
session then asked the status-line interview one question at a time and
every answer landed. The difference was not the content but the delivery.

A rule that lives only in an interview page is followed by sessions that
read the page. A rule domain is injected by the loader whenever a prompt
matches its recall words, and re-injected after compaction, so it reaches
every session in this repository. Propagation is what makes it abcd's rule
rather than one repository's habit: `ahoy install` writes the domain into a
managed repository's rules file the way it writes the other conventions.

## Decisions (maintainer, 2026-09-01)

- **The register follows the addressee.** A product thinker is asked in
  product terms: outcomes and choices, no record ids, no code, no internals.
  A technical facilitator is asked in the mechanism's terms, with the ids and
  the trade-offs. When the agent does not know which hat the human wears, the
  first question asks that, and the itd-200 mode records the answer so the
  next question does not ask again.

## What's In Scope

- The GRILL rule domain in this repository's `.abcd/rules.json`, and the
  same text in the `/abcd:intent` interview page, so the protocol and the
  injected rule match.
- The domain in the binary's bundled defaults, so a managed repository
  receives it at install and update, with the usual per-repo override.
- A check that the bundled default and this repository's copy do not drift.

## What's Out of Scope

- The options rule's content (itd-167, itd-168, itd-169 on the design
  branch): this intent delivers the asking, not the widening.
- Any harness that offers no interactive question tool: there the rule
  degrades to one question per message, stated as such.

## Mechanism

We expect a person to answer more of the questions put to them, because a
single plain question with an example is answerable where a numbered list
of record vocabulary is not.

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** a prompt in this repository that matches the GRILL recall words,
  **when** the rules loader runs, **then** the four GRILL rules are injected.
- **Given** an agent at a planning interview, **when** it needs a decision,
  **then** it asks one question through the interactive question tool, waits
  for the answer, and only then asks the next.
- **Given** a question is asked, **when** the human reads it, **then** it
  carries one sentence of context, one example per option of what that
  answer means in practice, and a null answer.
- **Given** `ahoy install` on a managed repository, **when** the conventions
  are written, **then** the GRILL domain is present in that repository's
  rules file with the bundled text.
- **Given** the mode says the addressee is the product thinker, **when** a
  question is asked, **then** it carries no record id, code or internal name,
  and the same decision put to the facilitator carries the mechanism and ids.
- **Given** the addressee is unknown, **when** the first question is asked,
  **then** it asks which hat the human wears before anything else.
- **Given** the bundled GRILL text and this repository's copy differ,
  **when** the record gates run, **then** they refuse.

## Open Questions

1. Whether the domain's recall words are wide enough to fire on every stop
   for a verdict, or whether the mode verb from itd-200 should inject it
   directly when the agent parks a stop.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
