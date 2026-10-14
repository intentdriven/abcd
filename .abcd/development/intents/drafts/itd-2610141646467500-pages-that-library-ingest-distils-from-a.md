---
id: itd-2610141646467500
slug: pages-that-library-ingest-distils-from-a
spec_id: null
kind: bundle-member
bundle: ideate-librarian
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: breaking
origin: researcher-authored
production_mode: hand-written
---

# Ingested pages stay private until a person promotes them

## Press Release

> **A person can ingest a confidential document without its pages reaching the
> team.** Pages that `abcd library ingest` distils from a source are kept on the
> person's machine by default, and enter the shared project only when a person
> promotes them.
>
> "I ingested a client's design notes to ask questions of them," said Grace, a
> consultant. "Nothing went into the repository until I promoted the two pages
> that were safe to share."

## Why This Matters

`library ingest` writes its cited pages into the project's `.abcd/memory/`, which
is committed, and checks nothing about whether the source is confidential. The
banned-names guard catches a confidential source's title at commit, but not its
ideas, quotes or address. Everything decided since itd-36 shipped points the
other way: the sources corpus keeps confidential sources apart (adr-41), memory
notes live on the machine until a person promotes them
(adr-2610091918443997), and ideate asks private or public. The product thinker
ruled on 2026-10-14: private by default, a person promotes to share.

This reverses itd-36's shared default, so it is breaking (adr-2610141646523373).

Decided in the product thinker's planning interview of 2026-10-14 (session abcd-66), which followed lab lab-261014142726-9c4d759, a hand-run of /abcd:ideate on one external write-up.

## Mechanism

We expect confidential material to stay private because the default write goes
to a machine store the repository never sees, and only a person's explicit
promotion moves a page into the committed tree.

## Scope Conditions

- Pages already committed under `.abcd/memory/` stay where they are; nothing is
  moved retroactively.
- Teammates see an ingested page only once it is promoted.

## Acceptance Criteria

- **Given** a source, **when** it is ingested, **then** its pages are written to
  the machine store and the working tree is unchanged.
- **Given** a page in the machine store, **when** a person promotes it, **then**
  it is written under `.abcd/memory/` and the promotion is recorded as the
  person's act.
- **Given** an agent, **when** it tries to promote a page, **then** it is refused.

## Open Questions

- The machine store's path and how promotion tells a person's act from an
  agent's (technical facilitator; the memory notes' promotion is the precedent).

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
