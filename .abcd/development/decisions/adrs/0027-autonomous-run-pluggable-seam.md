---
id: adr-27
slug: autonomous-run-pluggable-seam
status: accepted
date: 2026-07-06
supersedes: [adr-16]
superseded_by: null
related_intents: []
related_rfcs: []
related_adrs: [adr-22, adr-25]
---

# ADR-27: The autonomous run is a pluggable seam, not a Ralph port

## Context

abcd's autonomous execution was Ralph: a specific iteration loop with its own
post-iteration edge, where abcd hung behaviour like the autodrain gate
(ADR-16). That tied abcd's
"run work unattended" capability to one bundled loop and its internals. The
rebuild drops Ralph as a hard dependency
([ADR-22](0022-bundled-deps-as-pluggable-adapters.md)), so the autonomous run
needs a shape that does not assume any one loop is present.

## Decision

The autonomous run is a **pluggable `run` seam**, not a Ralph port. abcd defines
what an autonomous run *is* — iterate over ready work, gate each step on a
**receipt**, apply a **safety guard** — and lets the loop itself be supplied by
an adapter:

- **Claude Workflows** — delegate the loop to the host's workflow engine.
- **The companion harness's agent loop** — drive the companion harness's loop as a peer over conventions/MCP
  ([ADR-24](0024-companion-harness-peer-via-conventions.md)).
- **Thin native Go loop** — a minimal built-in fallback that iterates,
  gates on receipts, and enforces the safety guard, so abcd can run
  autonomously with no external loop present.

Because the loop is now an adapter and Ralph's post-iteration edge no longer
exists as a fixed thing, this ADR **supersedes**
ADR-16: the autodrain
behaviour ADR-16 anchored to Ralph's edge is re-expressed as **receipt gating**
inside the seam — a report-don't-block gate at each iteration boundary,
whichever adapter provides the loop.

## Alternatives Considered

- **Port Ralph natively.** Preserves the shipped loop behaviour. Rejected: it
  re-hard-codes one loop shape and its edge semantics, exactly the bundling
  ADR-22 removes, and forecloses delegating to a host workflow engine that may
  do it better.
- **No autonomous run in the MVP.** Simplest. Rejected: unattended execution is
  core to abcd's value, and the thin native loop is cheap enough to be the
  always-available fallback.
- **Chosen: a `run` seam with receipt gating and a safety guard, three adapter
  loops behind it, thin native Go loop as fallback.** Autonomy without binding
  to any one engine.

## Consequences

- abcd runs autonomously out of the box via the thin native loop; richer loops
  are opt-in adapters, and their absence degrades to the fallback rather than
  disabling the run.
- The autodrain/gate behaviour becomes a receipt-gated, report-not-block step
  at the iteration boundary defined by the seam, no longer coupled to Ralph's
  post-iteration hook.
- The safety guard is part of the seam contract, so every adapter loop inherits
  the same stop conditions rather than each re-implementing them.
- Ralph-specific overlay and edge machinery is removed with the rest of the
  bundled-dep subsystem ([ADR-22](0022-bundled-deps-as-pluggable-adapters.md)).
