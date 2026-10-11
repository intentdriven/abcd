---
id: adr-24
slug: companion-harness-peer-via-conventions
status: accepted
date: 2026-07-06
supersedes: null
superseded_by: null
related_intents: []
related_rfcs: []
related_adrs: [adr-22, adr-26]
---

# ADR-24: the companion harness is a peer integrated via conventions and MCP, not a code dependency

## Context

abcd and the companion harness overlap in territory — spec/task tracking, agent loops, review
flows — and it is tempting to make one import or vendor the other. abcd's prior
posture toward its integrations was exactly that: bundle the tool, couple to its
internals, and carry machinery to keep the coupling safe. The rebuild rejects
mandatory bundling wholesale ([ADR-22](0022-bundled-deps-as-pluggable-adapters.md)),
and the companion harness is the case that most needs a stated boundary, because a peer with
overlapping scope is where accidental coupling is most likely to creep back in.

## Decision

abcd integrates with the companion harness as a **peer, through shared conventions and MCP,
with no code dependency in either direction**. Neither imports, vendors, or
builds against the other's internals.

- **Conventions** are the ground-level contract: shared on-disk markdown shapes
  (the `ccpm` spec/task layout — [ADR-26](0026-native-spec-layer-ccpm-backend.md))
  that either tool can read and write without linking to the other.
- **MCP** is the runtime contract: when both are present, they interoperate
  over MCP as two independent servers, each a front door over its own core
  ([ADR-23](0023-transport-agnostic-core.md)).

Interoperation is a capability, never a prerequisite: abcd runs fully with no
companion harness present, and the companion harness owes abcd no build-time symbol.

## Alternatives Considered

- **Depend on the companion harness as a library (or vice versa).** Deepest integration,
  least glue. Rejected: it recreates the hard-dependency trap ADR-22 removes —
  version lockstep, vendoring, and a boundary subsystem to keep the coupling
  safe — between two tools that should each stand alone.
- **Ignore the companion harness; no interop.** Simplest. Rejected: the two share real
  territory, and convention-level compatibility is nearly free while unlocking
  ccpm as abcd's deeper spec backend and letting operators run both.
- **Chosen: peer via conventions + MCP, no code dependency.** Interop where it
  pays, independence everywhere else.

## Consequences

- abcd's spec/task on-disk format is convention-compatible with `ccpm` so the
  two tools read the same tree; the format is documented as a contract, not
  discovered from the companion harness's source.
- MCP interop is an optional adapter surface, wired only when the companion harness is
  present; its absence is normal operation.
- Neither project can break the other by refactoring internals — only a
  convention or MCP-contract change is observable across the boundary, and such
  changes are negotiated as contracts.
