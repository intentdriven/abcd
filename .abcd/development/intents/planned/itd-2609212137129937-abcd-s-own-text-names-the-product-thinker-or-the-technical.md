---
id: itd-2609212137129937
slug: abcd-s-own-text-names-the-product-thinker-or-the-technical
spec_id: spc-2609212141412864
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609212130146198, itd-201]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-97, itd-174, itd-2609211913453478]
related_adrs: [adr-2609212115255771]
---

# abcd's text names the product thinker or the technical facilitator, never the maintainer

## Press Release

> **Every place abcd writes names which of the two people it means, and the word maintainer leaves the vocabulary.**
>
> "Agents kept asking 'the maintainer' and I never knew if they meant me deciding what to build or me running the gates," said a product thinker. "They learned the word from abcd's own pages. Now the pages say which of us, the badge says which of us, and the word cannot come back."

## Why This Matters

On 2026-09-21 the product thinker noted that agents blur the two roles under one word. The word is abcd's: twelve occurrences in the command pages agents read, twenty-two in the brief, five in the principles, one in the bundled rules and one in this repository's rule overrides ("a planned intent's adoption is a maintainer decision"), and a persona hint. Ruled: everywhere abcd writes, each use rewritten to the role it meant, and a lint from then on. The sweep is the run's, not this session's.

## Mechanism

We expect agents to stop saying maintainer once no abcd text says it, because agents say what they read; shown wrong if transcripts after the sweep still use the word for either role.

## Scope Conditions

None stated.

## What's In Scope

- **The sweep**: every occurrence in the command pages, the bundled and repository rules, the brief, the principles, the docs, the personas registry and abcd's rendered text rewritten to the product thinker (what to build, adoption, rulings) or the technical facilitator (how, gates, mechanics); one change, each rewrite reviewed.
- **The lint**: the docs-lint banned-token list gains the word for every lint root including the command pages and the rules; the acknowledgements and a persona's outside job title are the only escapes, marked.
- **The questions**: every question or stop abcd's agents put to a human names which role it asks (the GRILL rule made mechanical where the page templates carry the question).
- **The glossary**: the two roles defined beside the record-families page, citing itd-97's stance that the facilitator is a mode.
- **The test**: walks the rendered help and the plugin pages for the word.

## What's Out of Scope

- Defining a third role.
- Changing what either role decides (itd-174, itd-97).

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that filed and planned this intent (adr-2609212115255771 records the vocabulary rulings it rests on):

1. Everywhere abcd writes, each use names the role it meant; a lint bans the word after (ruled 2026-09-21).
2. Captured for the run to build; no sweep in the session that ruled it.

## Open Questions

_None open._

## Acceptance Criteria

- **Given** the command pages, the rules, the brief, the principles, the docs and the personas, **when** the sweep lands, **then** no occurrence of the word remains outside the marked escapes, and each rewrite names the product thinker or the technical facilitator.
- **Given** a new page or rule carrying the word, **when** the docs lint runs, **then** it is refused as a banned token on every lint root.
- **Given** a question or stop an agent puts to a human through the page templates, **when** it renders, **then** it names which of the two roles it asks.
- **Given** the glossary, **when** the two roles are looked up, **then** each has an entry beside the record-families page citing itd-97.
- **Given** the rendered help and the plugin pages, **when** the test runs, **then** it fails on the word.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: the badge intent and every interview today turn on which of two people is being asked, and the pages that brief agents blur them; we expect agents to stop saying maintainer once no abcd text says it; shown wrong if transcripts after the sweep still use the word
