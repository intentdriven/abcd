---
id: itd-2610141646467500
slug: pages-that-library-ingest-distils-from-a
spec_id: null
kind: standalone
bundle: null
suggested_kind: null
reclassification_history:
  - { date: 2026-10-14, from: bundle-member, to: standalone, reason: "The product thinker ruled in the planning interview that private ingest ships as soon as it is ready, outside bundle ideate-librarian: it needs nothing from the librarian." }
builds_on: []
reverses: [itd-36]
related_adrs: [adr-2610141646523373]
severity: minor
impact: breaking
origin: researcher-authored
production_mode: hand-written
---

# Ingested pages stay private until a person promotes them

## Press Release

> **A person can ingest a confidential document without its pages reaching the
> team.** Pages that `abcd library ingest` distils from a source are kept on the
> person's machine by default. The person can ask questions of them and check
> them as before, and a page enters the shared project only when the person
> promotes it, after it passes the same redaction and banned-names checks as
> every committed page.
>
> "I ingested a client's design notes to ask questions of them," said Jack, a
> consultant. "Nothing went into the repository until I promoted the two pages
> that were safe to share."

## Why This Matters

`library ingest` writes its cited pages into the project's `.abcd/memory/`,
which is committed, and checks nothing about whether the source is
confidential (internal/core/memory/ingest.go). The banned-names guard catches a
confidential source's title at commit, but not its ideas, quotes or address.
The product thinker ruled on 2026-10-14: private by default, a person promotes
to share. This reverses itd-36's shared default (adr-2610141646523373), so it is
breaking: a teammate sees an ingested page only once it is promoted, and the
first ingest after the change says so.

## What's In Scope

- A machine store for ingested pages, `~/.abcd.noindex/library/<root-sha>/pages/`,
  owner-only.
- `library ask` and `library lint` over both stores; `ask --file-back` writing to
  the machine store whenever a cited page is held only there.
- `library promote <page>`: the person's act that moves a page into
  `.abcd/memory/`.

## What's Out of Scope

- Pages already committed under `.abcd/memory/`: nothing is moved.
- Reading packs and research notes (bundle ideate-librarian).

## Mechanism

We expect confidential material to stay private because the default write goes
to a store the repository never sees, and the only path into the committed tree
is a promotion that re-runs the redactor and the banned-names check. Shown wrong
if a page reaches `.abcd/memory/` without a promotion, or a promoted page holds a
banned phrase.

## Scope Conditions

- One person's machine and checkout; a page is promoted from the machine that
  ingested it.
- Telling a person from an agent is done the way the sources ledger's flip does
  it: the verb asks at a terminal, and a guard entry refuses it when an agent's
  shell runs it. Neither can prove a person typed the answer; the command page
  says so.

## Acceptance Criteria

- **Given** a source, **when** it is ingested, **then** its pages are written to
  the machine store with owner-only permissions and the working tree is
  unchanged.
- **Given** the first ingest after this change in a checkout, **when** it runs,
  **then** it says that pages now stay private until promoted.
- **Given** pages in both stores, **when** `library ask` answers, **then** it
  cites pages from both; **when** `--file-back` cites any page held only on the
  machine, **then** the synthesis page is written to the machine store.
- **Given** pages in both stores, **when** `library lint` runs, **then** it checks
  both.
- **Given** a page in the machine store, **when** a person runs `library promote`
  on it at a terminal, **then** it is written under `.abcd/memory/` after the
  redactor and the banned-names check pass, and the promotion is recorded.
- **Given** a page whose source is in the corpus's confidential class, **when**
  it is promoted, **then** the promotion is refused.
- **Given** an agent's shell, **when** it runs `library promote`, **then** the
  guard refuses it.

## Prior Art

- itd-36 (shipped): the shared memory store this reverses.
- adr-2610091918443997: memory notes stay on the machine until a person promotes
  them, the stance this follows.
- The sources ledger's flip (`library ledger --flip`): the person-only act and its
  guard entry, the mechanism this copies.

## SOTA

- Personal knowledge tools with local-first storage and explicit sharing
  (Obsidian vaults, Logseq graphs): mature; sharing is a separate act.
- Retrieval stores with access tiers (enterprise RAG with per-document ACLs):
  mature, server-based.

**Path:** 2. A native floor: a second file store and a promotion verb; an
external store would add a dependency for one person's pages.

## Open Questions

_None._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
