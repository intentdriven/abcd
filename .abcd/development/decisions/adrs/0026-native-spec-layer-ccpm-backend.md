---
id: adr-26
slug: native-spec-layer-ccpm-backend
status: accepted
date: 2026-07-06
supersedes: [itd-49]
superseded_by: null
related_intents: []
related_rfcs: []
related_adrs: [adr-3, adr-22, adr-24]
---

# ADR-26: A native minimal spec layer with the companion harness `ccpm` as the primary deeper backend

## Context

abcd's spec and task tracking rode on flow-next: its receipts, its plan/work
skills, its on-disk state. The rebuild drops flow-next as a hard dependency
([ADR-22](0022-bundled-deps-as-pluggable-adapters.md)), which leaves abcd
needing a spec/task store of its own — but abcd already owns a lightweight,
proven pattern for lifecycle state (directory location is the source of truth,
[ADR-3](0003-directory-as-truth-for-lifecycle.md)), and the companion harness's `ccpm` is a
capable deeper store abcd can reach at the convention level without linking to
it ([ADR-24](0024-companion-harness-peer-via-conventions.md)).

## Decision

The spec/task layer is a **native minimal store as the MVP**, with **the companion
harness `ccpm` as the primary deeper backend**, and **flow-next is not built**.

- **Native minimal (MVP).** Specs and tasks are directories whose location
  encodes status ([ADR-3](0003-directory-as-truth-for-lifecycle.md)), plus a
  **dependency graph** over them. This is enough to plan, sequence, and track
  work with no external tool, and it ships in the first milestone.
- **The companion harness `ccpm` (primary deeper backend).** When an operator wants a richer
  store, abcd reads and writes the `ccpm` markdown layout at the **convention
  level** — a shared on-disk shape, no binary or library dependency on the companion harness.
- **flow-next: not built.** Its role is covered by the native store below and
  ccpm above; abcd carries no flow-next code or contract.

## Alternatives Considered

- **Port flow-next's model natively.** Reuses a known design. Rejected:
  flow-next's shape assumes its own orchestration and receipts; porting it
  rebuilds machinery ADR-22 exists to shed, when directory-as-truth already
  gives abcd a lighter native store.
- **Native store only, no deeper backend.** Simplest. Rejected: it caps abcd at
  minimal tracking and forgoes ccpm's depth, which is available almost for free
  at convention level.
- **Depend on ccpm as the only store.** Deepest integration. Rejected: it makes
  a peer tool mandatory, breaking the no-hard-deps rule and leaving abcd unable
  to track work out of the box.
- **Chosen: native minimal MVP + ccpm as convention-level deeper backend,
  flow-next dropped.** Works alone, scales up when a peer store is present.

## Consequences

- The MVP spec layer is buildable with only abcd's own code — directories plus
  a dependency graph — so first-milestone planning does not wait on any adapter.
- ccpm interop is a convention contract, documented as an on-disk shape, wired
  as an optional backend and absent by default.
- The spec store is one more capability behind the core
  ([ADR-23](0023-transport-agnostic-core.md)): every front door sees the same
  specs and tasks regardless of which backend holds them.
- flow-next references in the design record are reconciled away as the native
  store and ccpm backend land; nothing re-introduces a flow-next contract.
- itd-49 (the flow-state drift detector) is **superseded** by this decision: it
  compared checkpoint blocks against flow-next's runtime state store, and
  flow-next is not built, so there is nothing left to drift against.
