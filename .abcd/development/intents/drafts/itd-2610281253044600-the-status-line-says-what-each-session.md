---
id: itd-2610281253044600
slug: the-status-line-says-what-each-session
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610111707085098]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# The status line says what each session is working on

## Press Release

> The status line says what each session is working on. Beside a session's name, abcd's status line shows the piece of work that session holds: the title of the issue or intent it has reserved or is building, for example "Henry · Sweep stale reservations". A person running several sessions sees at a glance which one is on what, without asking each of them. The session's name itself does not change, so other sessions keep messaging it by the same short name however often its work moves on. A session that holds nothing shows nothing there, and the new element switches off like every other element of the status line.

## Why This Matters

Session names (itd-2610111707085098, shipped) gave each session a short name that a person and its peer sessions address it by. A person running several sessions still cannot tell from the name what each one is doing: the name says which session, not what it is on.

The obvious place for the task, the session's title, is the wrong one. The host uses the title as the name other sessions message, so a title such as "Henry working on …" would rename the session every time its work changed, and a peer's message to the old name would miss. abcd also tells the name it gave from one the person typed by matching the title exactly, so a longer title would read as the person's own on resume, and the name would stop being held for that session.

The status line is not the address, so it can carry the task. abcd already knows what a session holds without asking the model: a reservation records the working copy it was made from, and a build run records its session. The element can be filled from local state, with no network read and no summary written by the model.

## Decisions

- 2026-10-28, the product thinker: the task goes on the status line, after the session's name, and the title stays the bare name. The accepted cost: the status line shows only in the session's own terminal, so the host's session list still shows names alone, and telling sessions apart there is outside this intent.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

- Title only, or the record's id as well? The id is what a peer can act on, the title is what a person reads, and the row's width bounds both.
- How is a session matched to what it holds? Its working directory misses the usual case here, a session whose own directory is the main checkout while its work, and so its reservation, is in a store worktree. A build run's recorded session matches exactly but covers only builds.
- A session holding several records: the newest, the first, or a count?
- Work with no record, such as a session answering a question: nothing, the branch already on the row, or a line the session sets for itself?

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
