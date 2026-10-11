---
id: adr-13
slug: fn38-memory-single-writer-and-write-lint
status: accepted
date: 2026-06-09
supersedes: null
superseded_by: null
related_intents: [itd-36]
related_rfcs: []
related_adrs: [adr-25]
---

# ADR-13: Durable memory writes — single-writer, atomic-rename crash model

## Context

abcd's memory store is built on **markdown files** (`.abcd/memory/index.md`,
`log.md`, per-page files, `.sources_index.json`). A near-ACID design — a
transaction journal, `pending` markers, crash-recovery reconciliation, lock
ordering — is only warranted if concurrent writers can race and torn writes can
occur. Neither premise holds for this store, so its crash-consistency model is a
deliberate choice rather than a default to the heaviest option.

## Decision

The memory store uses a **single-writer, atomic-rename crash model** (a Go
implementation):

1. **Single-writer to the memory store.** The memory writer touches only
   `.abcd/memory/**`. Concurrent activities — a facilitator refining
   `.abcd/development/brief/**`, a product thinker amending `.abcd/intents/**`,
   issue capture into the `.abcd/work/issues/` ledger — write **disjoint file
   trees** and
   never mutate the memory store. There is no real concurrent-writer scenario
   against `.abcd/memory/**`.

2. **Atomic-rename only.** Each file write is write-temp-then-`rename()`. A crash
   leaves either the complete old or complete new file, never a torn file. No
   transaction journal, no `pending` markers, no lock-ordering protocol. (No
   mid-write corruption has ever been observed in the memory files — the
   empirical failure rate of what a transaction model would defend against is
   ~zero.)

3. **No explicit crash-recovery.** With atomic rename there is no torn write to
   heal; the only residue is cross-file staleness, healed by the **idempotent
   sibling-reconciliation that already runs before every mutating write**. Bare /
   read-only commands stay strictly non-mutating and merely *report* drift
   ("index stale; run an ingest"). This dissolves the recovery-vs-non-mutation
   tension at the root rather than patching it.

4. **Write and lint are separate concerns.** The memory **write core** —
   registry/index (atomic-rename), ingest/atomic-promotion, the
   `principle-distiller` prompt, launch consumer and docs — is the part that
   *stores*. The memory **quality/lint** — the MS/ML/MQ/SD constraint
   validation, citations, provenance budgets, the `lint` command — is the part
   that *audits*. They are built and reasoned about independently, so the
   actually-blocking write core can land ahead of the audit surface.

Review of the memory design is **host-delegated**
([adr-25](0025-host-delegated-llm-default.md)): abcd hands the design and its
prompts to the host's subagent dispatch and consumes the structured verdict,
rather than owning a fixed multi-backend review pipeline.

## Consequences

- The entire class of concurrency objection (lost-update races, transaction
  ownership of `pending` markers, recovery-mutation colliding with the
  non-mutating-read contract) is removed **by construction**, not argued down.
- The write core is independent of the lint surface and can ship first.
- The atomic-rename model keeps the store's durability guarantee cheap: one
  `rename()` per file, no journal to replay, no recovery pass to schedule.

## Related Documentation

- [adr-25](0025-host-delegated-llm-default.md) — the LLM is host-delegated by
  default; review runs through the host's subagent dispatch.
- Intent: `.abcd/development/intents/drafts/itd-36-memory-unification.md`
