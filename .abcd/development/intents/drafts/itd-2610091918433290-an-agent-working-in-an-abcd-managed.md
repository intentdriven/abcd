---
id: itd-2610091918433290
slug: an-agent-working-in-an-abcd-managed
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-3]
severity: minor
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-39, itd-36, itd-2610090831227812]
related_adrs: [adr-2610091918443997]
---

# Agents keep memory notes with /abcd:memory, recalled when they matter

## Press Release

> An agent working in an abcd-managed project keeps what it learns as short memory notes that come back exactly when they matter: it adds, lists and removes them with /abcd:memory, each with the words that should recall it, and the same loader that brings in the project's rules brings a note back when a prompt matches. Notes stay on the person's own machine until one is promoted into the project's record for every agent to share, and abcd never edits the host's own memory files.

## Why This Matters

> _Why this matters to the user — replace before planning._

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

- **Given** a person reading abcd's user-facing docs after this intent ships, **when** they look up memory, **then** the page explains memory and the library side by side in plain words (what each holds, who adds to it, how it comes back, where it can go), the same comparison the brief's surfaces overview carries, so the two are never mistaken for one store. (Required by the product thinker, 2026-10-09: "explain both in that accessible way in the docs and in the brief".)

## Open Questions

- Where notes are kept: a `memory.json` beside each `rules.json` layer (the machine's `~/.abcd.noindex/memory.json`, the project's `.abcd/memory.json`), read by the same loader, or entries inside `rules.json` itself. The facilitator's lean is `memory.json`: agents write notes, people curate rules, and the one-writer-per-file principle keeps an agent's note from editing the file that holds reviewed conventions; the product thinker confirms at planning.
- What may enter a note without the person's yes, and how a note is promoted into the project's record: decided in adr-2610091918443997.
- How this relates to itd-39 (memory pages recalled at the right moment): this intent writes and manages the notes; itd-39 retrieves pages. Keep both, bundle them, or let one refine the other.
- What happens to the memory notes an agent already keeps in the host's own memory files, which abcd never touches: does abcd offer a way for the person to move a note across by hand?

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Decisions

- 2026-10-09: the product thinker chose to manage memory notes through abcd's own rules loader rather than a separate recall path or the host's memory files (answer: "yes, file it as A"), after the verb-split interview freed the name `memory` by merging the sources and memory commands into `library`; `/abcd:memory` is an agent's command.
