---
id: adr-29
slug: native-transcript-corpus
status: superseded
date: 2026-07-06
supersedes: null
superseded_by: adr-2609090717039680
related_intents: []
related_rfcs: []
related_adrs: [adr-22]
---

# ADR-29: A native local redacted transcript corpus

## Context

abcd captured session transcripts through specstory, redirected to an external
history store keyed on the repo's root-commit SHA. The rebuild drops specstory
as a hard dependency ([ADR-22](0022-bundled-deps-as-pluggable-adapters.md)), so
abcd needs its own transcript capture — and transcripts are worth keeping for
their own sake: they are the research and benchmark corpus that lets abcd study
how its own flows behave. abcd already owns a proven redaction design — the
two-stage write-time-sanitise-then-verify model of
ADR-6 — built for exactly this
class of content.

## Decision

abcd captures transcripts into a **native local redacted store** as the
specstory replacement and as a **research/benchmark corpus**:

- **Local and native** — no external tool required; capture is abcd's own code.
- **Root-SHA-keyed** — transcripts are keyed on the repo's root-commit SHA, so a
  repo's sessions group stably across clones and renames.
- **Gitignored** — the store is never committed; it is local working data.
- **Redacted on capture** — redaction runs at write time, before anything lands
  on disk.

Redaction **reuses ADR-6's
two-stage model** — the write-time sanitiser rewrites high-confidence secrets
and home-directory paths, and the pre-commit-class verifier catches the rest —
so this store and the review store share one redaction discipline. This ADR
**links to ADR-6, it does not supersede it**: ADR-6's decision stands; this ADR
applies its redaction model to a new corpus.

A **private companion / cloud store is optional** — an operator may sync the
local corpus to a private companion for cross-machine research, but the local
redacted store is the default and works standalone.

## Alternatives Considered

- **Keep specstory + the external history store.** Preserves the shipped
  capture path. Rejected: it hard-requires an external tool ADR-22 makes
  optional and puts the corpus outside abcd's control.
- **Capture raw, redact later.** Simplest capture. Rejected: raw transcripts on
  disk are an exposure window; ADR-6 already settled that redaction is
  write-time, and a research corpus is exactly where un-redacted secrets must
  never persist.
- **A new redaction model for transcripts.** Rejected: it would fork a
  security-critical discipline into two implementations that drift; ADR-6's
  two-stage model already covers this content class.
- **Chosen: native, root-SHA-keyed, gitignored, redacted-on-capture, reusing
  ADR-6's two-stage model, optional private companion.** One redaction
  discipline, no external dependency, a corpus abcd owns.

## Consequences

- Transcript capture works with no external tool; the corpus is abcd's own
  local data, redacted before it touches disk.
- The redaction implementation is shared with the review store, so a fix to the
  sanitiser patterns hardens both surfaces at once — the reuse ADR-6's model
  makes possible.
- The corpus is available for research and benchmarking (studying abcd's own
  flows) without any content leaving the machine by default.
- An optional private companion/cloud sync is a separate, opt-in path over the
  same redacted store — never a prerequisite, never a place raw transcripts go.
