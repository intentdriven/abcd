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

Agents relearn the same project facts every session: which flags the tests need, what you prefer. The host's own memory files hold some of it, outside the project and unreviewed. Notes captured automatically and reviewed now and then let lessons carry over in plain sight.

## Mechanism

We expect fewer repeated mistakes, because a note comes back exactly when a prompt matches its words, through the loader that already brings rules in. Shown wrong if agents keep repeating a mistake a note covers, or if reviews find most notes wrong or stale.

## Scope Conditions

- Holds for a host that runs abcd's prompt hook, through which the rules loader recalls.
- Holds while notes are written by agents in sessions on the person's own machine, and promotion is a person's act.

## Acceptance Criteria

- **Given** an agent that learns, during a session, that the project's tests need the race flag, **when** it writes a memory note saying so, without asking the person, **then** in the next session a prompt about tests brings the note back.
- **Given** notes written since the last review, **when** the person opens the review, **then** they see every one of those notes with who wrote it, when and in which session, and they keep, discard or promote each.
- **Given** a note the person promotes, **when** the promotion lands, **then** the note joins the project's record, so every agent and teammate gets it; only a person promotes, and in a repository with more than two maintainers at least one other maintainer must agree before it lands.
- **Given** a memory note and a longer memory page that both match a prompt about tests, **when** that prompt is sent, **then** both come back together, through one recall.
- **Given** a lesson in the host's own memory files, **when** the person points at it (or pastes it), **then** it is filed as a memory note, and nothing reads or changes the host's memory files on its own.
- **Given** a person reading abcd's user-facing docs after this intent ships, **when** they look up memory, **then** the page explains memory and the library side by side in plain words (what each holds, who adds to it, how it comes back, where it can go), the same comparison the brief's surfaces overview carries, so the two are never mistaken for one store. (Required by the product thinker, 2026-10-09: "explain both in that accessible way in the docs and in the brief".)
- **Given** a personal note and a promoted note, **when** each is stored, **then** the personal note lives in the machine's file and the promoted note in the project's.
- **Given** a note whose text holds a secret, a private name or a network address, **when** an agent writes it, **then** the write is refused, so a note can never hold one.
- **Given** any note, **when** it is written, **then** it records who wrote it, when, and in which session.
- **Given** a session filling up, **when** a prompt matches notes, **then** recall shrinks as the session fills: the notes in full, then one line each, then nothing.

## Open Questions

- None for the product thinker. The four questions first left here are settled under Decisions (2026-10-11); the technical questions the plan leaves open are in the bundle's spec.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Decisions

- 2026-10-09: the product thinker chose to manage memory notes through abcd's own rules loader rather than a separate recall path or the host's memory files (answer: "yes, file it as A"), after the verb-split interview freed the name `memory` by merging the sources and memory commands into `library`; `/abcd:memory` is an agent's command.
- 2026-10-11, product thinker, what agents may write: agents capture automatically. An agent writes any note without asking. Every note records who wrote it, when, and in which session, and the product thinker or the technical facilitator reviews the notes from time to time and keeps, discards or promotes each one. Asked why this was a question at all: a note steers later sessions the way a rule does, so how much gets written unseen was the trust question; the answer is automatic capture with visible review. Recorded in adr-2610091918443997.
- 2026-10-11, product thinker, promotion: only a person promotes a note into the project's record, and only in a review. In a repository with more than two maintainers, at least one other maintainer must agree before a promotion lands. Recorded in adr-2610091918443997.
- 2026-10-11, product thinker, notes and pages: "Build both together". Memory notes (this intent) and memory pages (itd-39) share one recall and are planned as one bundle, so the third question first left here is closed.
- 2026-10-11, product thinker, the host's memory files: "Offer a manual move". The person points at a lesson in the host's own memory files, or pastes it, and it is filed as a memory note. abcd never reads or writes the host's memory files on its own; the product thinker's ruling of 2026-10-05 that abcd never touches `~/.claude` stands. Recorded in adr-2610091918443997.
- 2026-10-11, technical facilitator, where notes are kept: a `memory.json` beside each `rules.json` layer. The machine's `~/.abcd.noindex/memory.json` holds personal notes and the project's `.abcd/memory.json` holds promoted notes, and the same loader reads both. Agents write `memory.json`; people curate `rules.json`. This closes the first question first left here, as the facilitator leaned.
- 2026-10-11, technical facilitator, how recall shrinks: "Share the brackets". Note recall uses the same context brackets as the rules loader (itd-3) and has no thresholds of its own.
- 2026-10-11, technical facilitator, which notes survive when only some fit: "A relevance ranking". When the context is short and only some matching titles fit, a relevance ranking chooses the ones that stay, not a raw count of matching words. The spec designs a simple ranking whose choice can be explained, cites its prior art, and adds no dependency unless the person signs off.
- 2026-10-11, settled on the record without a question: a note never holds a secret, a private name or a live network identifier. abcd's existing redactor and private-names banlist refuse one. Recorded in adr-2610091918443997.
- 2026-10-11: the product thinker accepted Why This Matters and the Mechanism with its falsifier exactly as written above, and walked and accepted all ten acceptance criteria.
- 2026-10-11, technical facilitator: the two scope conditions (a host that runs abcd's prompt hook; notes written by agents on the person's own machine, with promotion a person's act) were proposed by the facilitator, not asked of the product thinker.
- 2026-10-11: the impact is additive. Nothing that exists today is renamed or removed: `/abcd:memory` was freed by the verb split (itd-2610090831227812), and notes are a new store.
