---
id: adr-2610091918443997
slug: what-may-enter-abcd-s-memory-notes
status: accepted
date: 2026-10-09
supersedes: null
superseded_by: null
related_intents: [itd-2610091918433290, itd-39, itd-3, itd-2610111655362931]
related_rfcs: []
related_adrs: []
---

# ADR-2610091918443997: What may enter abcd's memory notes without the person's yes, and how a note reaches the record

## Context

The product thinker asked on 2026-10-09 (itd-2610091918433290) for agents to keep memory notes through `/abcd:memory`, recalled by the rules loader. Notes are written mostly by agents, often mid-session, so what may be written without the person's yes is a trust question, and so is the step that moves a note from the person's machine into the project's committed record. Constraints already in force: abcd never touches the host's `~/.claude` (the product thinker's ruling of 2026-10-05); the memory-graduates-to-record principle (a lesson every agent should receive belongs in the committed record, never only one person's local memory); the private-names banlist and the scanner's redaction for anything committed; and the loader's ownership and size checks on the machine's rules file.

Open before this record can be decided:

- Which notes an agent may write on its own (for example a fact about the person's machine) and which need the person's yes (for example anything about another person, or anything promoted).
- Whether promotion is the person's act alone, as the source ledger's flip is, or an agent's with disclosure.
- What a note may never hold (secrets, private names, live network identifiers), and which check refuses it.

## Decision

Decided at the planning interview of itd-2610091918433290 on 2026-10-11, by the product thinker except where a line says otherwise.

- **An agent writes any note without asking.** Capture is automatic: there is no class of note that needs the person's yes before it is written.
- **Every note is visible and reviewed.** Each note records who wrote it, when, and in which session. The product thinker or the technical facilitator reviews the notes from time to time and keeps, discards or promotes each one, on the later dashboard the draft itd-2610111655362931 carries (the product thinker's ruling of 2026-10-11); until it ships, the review is `memory list` and `memory remove` (Consequences). A note steers later sessions the way a rule does, which is why this was a trust question; the answer is automatic capture with visible review, not a yes before each write.
- **Only a person promotes, in the dashboard's review.** Promotion, which the dashboard draft itd-2610111655362931 carries, moves a note from the person's machine (`~/.abcd.noindex/memory.json`) into the project's record (`.abcd/memory.json`, the storage the technical facilitator ruled the same day). An agent never promotes. In a repository with more than two maintainers, at least one other maintainer must agree before a promotion lands.
- **The host's memory files are moved by hand only.** The person points at a lesson in the host's own memory files, or pastes it, and it is filed as a memory note. abcd never reads or writes the host's memory files on its own; the product thinker's ruling of 2026-10-05 that abcd never touches `~/.claude` stands.
- **Never in a note** (settled on the record, not asked): a secret, a private name or a live network identifier. The write is judged by the existing redactor (the shared scanner's secret, identity and network spans, which the memory store's writer already applies) and the private-names banlist, and a note that holds one is refused outright, never redacted and written: a lesson with a span cut out of it may say something wrong, so the agent rewrites it in its own words. How a note entering the project's file is judged is the dashboard draft's to plan, since this record's notes intent writes only the machine's file.

## Alternatives Considered

- An agent writes any note without asking; every note is stamped and reviewed, and only a person promotes (chosen): lessons are kept the moment they are learned, and the review is where trust is earned, in plain sight.
- Every note asks the person first: nothing is written unseen, but an agent mid-task stops to ask about each small fact, so most lessons are never kept or the person learns to say yes without reading.
- An agent writes and promotes, disclosing the promotion in the commit: no person is needed for a note to reach every teammate's agents, so a wrong lesson spreads before anyone reads it, and the commit trailer is the only trace.
- An agent writes freely to the machine layer but needs the person's yes for some kinds of note (for example anything about another person): a line between kinds that every agent must judge alike, where the review already shows every note whatever its kind.

## Consequences

Notes accumulate unseen between reviews, so the review is load-bearing: a note recalled wrongly steers sessions until a person reviews it, and the intent's falsifier names reviews that find most notes wrong or stale. Promotion needs a person's act the binary can tell from an agent's, as the source ledger's flip does under adr-41 (a guard entry and a terminal check), and the more-than-two-maintainers rule needs abcd to know how many maintainers a repository has, which it does not record today; both moved, with the review and promotion, to the dashboard draft itd-2610111655362931 (the product thinker's ruling of 2026-10-11): its Decisions carry the guard entry and terminal check as a starting design, and its open questions the maintainer count and how agreement is recorded. The review's home is that dashboard. Notes ship before it, by the product thinker's ruling of 2026-10-11 ("Ship, list and remove"): until the dashboard ships, the review is `memory list` and `memory remove`, which removes any personal note, run by a person or by an agent at a person's request, and a note reaches the project's file only through a person's reviewed change to it. The redactor and the banlist become a write-time gate on the machine's file, which no committed-artefact scan reaches, so they must run on every note write, not only on promotion.
