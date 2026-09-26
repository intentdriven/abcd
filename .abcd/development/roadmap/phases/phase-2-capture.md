> **Retired on 2026-09-21** (adr-2609212115255771): phases and milestones are no longer units of the record. Sequencing is dependencies plus the lifecycle shelves, rendered as the Now / Next / Later status block; the checkpoint is the derived release. This document stays as history and is not maintained.

# Phase 2 — History, capture, and memory

## Expectation

By the end of this phase, an abcd-installed repo has a working native memory
substrate: session history is captured locally, findings are captured to a
structured ledger, and both feed a curated knowledge store. Session transcripts
land in a **native local redacted store** (per
[adr-29](../../decisions/adrs/0029-native-transcript-corpus.md)) — no external
recorder is required. `/abcd:capture "<text>"` lands a finding immediately in a
structured issue ledger — no context switch, no form, no triage decision at
capture time; issues carry stable `iss-N` IDs and a folder-as-status lifecycle
(`open` / `resolved` / `wontfix`). `/abcd:memory` unifies these local sources
into the curated substrate later phases draw on. This is the phase that gives
abcd a memory of its own, and makes "I noticed something" a one-line act rather
than a deferred intention.

## Milestone

- The native transcript store captures session history to the per-repo store
  ahoy scaffolded (`~/.abcd/history/` keyed on root-commit SHA), redacted on
  write, with no external recorder in the path (per adr-29).
- `/abcd:capture "<text>"` writes a structured issue-ledger entry under
  `.abcd/work/issues/open/` with a stable `iss-N` ID, per the
  acceptance in `04-surfaces/06-capture.md`.
- The folder-as-status lifecycle works: `capture resolve <iss-N>` and
  `capture wontfix <iss-N>` move the entry between status folders; no `status:`
  field is stored (directory is truth, per adr-3).
- `/abcd:capture list` returns a filtered view of the ledger (the earned
  sub-verb — bare `/abcd:capture` with no argument renders help per the
  bare-command-as-render discipline).
- `/abcd:memory` ingests the local sources (transcript store, issue ledger) into
  the curated substrate that later phases consume.
- The ledger and store directories are created on first use if absent — the user
  never scaffolds them by hand.

## Phase Acceptance

> _Roll-up acceptance per [adr-9 amendment](../../decisions/adrs/0009-phase-as-product-layer.md). Each bullet asserts an emergent, cross-intent truth or a phase-spanning user journey — never a copy of an intent's own `## Acceptance Criteria`._

- **Given** an abcd-installed repo with no prior captures, **when** a user runs
  `/abcd:capture` three times and then `capture resolve` on one entry, **then**
  the ledger holds two entries under `open/` and one under `resolved/`, with
  stable IDs unchanged across the move — the directory-as-truth lifecycle
  working end to end.
- **Given** a session run in an abcd-installed repo, **when** it ends, **then**
  the transcript lands in the native local store redacted, with no external
  recorder involved — the self-contained history property adr-29 guarantees.
- **Given** the transcript store and the issue ledger both hold local content,
  **when** a user runs `/abcd:memory`, **then** the curated substrate reflects
  both sources — the unified-memory property that no single capture surface
  delivers alone, and that Phase 6's lifeboat later draws its synthesised
  content from.

## Scope

**Intents:** itd-4 (`/abcd:capture` issue ledger — capture-only; cross-corpus
synthesis via `/abcd:dredge` is a separate, later-phase intent and is not in
scope here), itd-36 (`/abcd:memory` unification — the multi-source curated
knowledge substrate).

**Native history store** (per adr-29): the session-transcript capture and read
behaviour on the per-repo store ahoy scaffolds in Phase 1. The store is native
and local; redaction is applied on write. This is the substrate the capture
ledger and the memory command sit alongside.

**Capture is arch-neutral.** The capture logic — the `iss-N` allocator, the
ledger primitives, the command flow, legacy-issue migration, and the
fidelity-reviewer cross-check — is engine-neutral. It is delivered natively in
Go against the native stores, with the capture surface wired at the plugin
command layer rather than through an external hook.

This phase groups the three surfaces that share one substrate — session history,
the issue ledger, and the curated memory store. Capture and intent authoring
(Phase 3) stay separate phases: capture is a fast, low-stakes ledger entry;
intent authoring is a high-stakes, Socratic act. The two "the user records
something" surfaces are kept apart so each phase milestone stays sharp.

## Maps against

- **Brief:** `04-surfaces/06-capture.md` (the capture command);
  `05-internals/07-memory.md` (itd-36's component spec);
  `05-internals/03-configuration.md` (the `.abcd/work/` issue-ledger namespace
  and visibility-driven gitignore).
- **Intents deliver the expectation:** itd-4 delivers the capture command,
  ledger, and folder-as-status lifecycle; itd-36 delivers the curated memory
  substrate.
- **ADRs realised:** adr-3 (directory-as-truth — capture's `open`/`resolved`/
  `wontfix` folders ARE the status, no field stored); adr-29 (native transcript
  corpus).

## Dependency rationale

- **Runs after Phase 1** — capture and history write under the `.abcd/`
  namespace ahoy provisions, and the transcript store is captured into the
  per-repo scaffolding ahoy lays down. itd-3's per-domain rule injection makes
  capture's discipline visible.
- **History and memory before Phase 6** — the lifeboat pipeline draws its
  synthesised content from the curated memory substrate and its retrieval from
  the native transcript store; both must exist and be populated before disembark
  can consume them.
- **Independent of Phases 3–5** — the capture, history, and memory surfaces have
  no hard dependency on intent authoring, the spec engine, or the run seam. The
  one soft link: a captured issue can later be promoted toward an intent draft —
  but `capture promote` is part of itd-4's scope and does not require the intent
  phase to have shipped.

## Open questions

- `capture promote <iss-N>` produces an intent *draft* — confirm whether that
  draft must already satisfy the Phase 0 acceptance-gate discipline (itd-1) at
  promotion time, or whether a promoted draft is allowed to be incomplete until
  it reaches `/abcd:intent` in Phase 3. (Lean: incomplete is fine in `drafts/`;
  the gate bites at `/abcd:intent plan`, not at capture-promote.)
- Confirm the transcript store's redaction pass shares one implementation with
  the launch scrub stack (Phase 1) rather than duplicating a scrubber.
