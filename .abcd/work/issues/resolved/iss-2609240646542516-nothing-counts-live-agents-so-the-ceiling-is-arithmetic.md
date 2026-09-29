---
schema_version: 1
id: "iss-2609240646542516"
slug: "nothing-counts-live-agents-so-the-ceiling-is-arithmetic"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/bounds.go"
resolution: "implement log counts the agents a session's own agent_start/agent_end lines declare alive since it joined, refuses an agent_start past the session's ceiling (logged as refusal condition agent_ceiling), and check reports agents_alive. Live processes are not guessed: a fork or an agent started outside the log stays invisible, so the lane brief's no-fork rule stays the discipline for that half, as the record says."
impact: additive
resolved_by:
  commit: "b3f77f3d0"
---

Nothing in `abcd implement` counts the agents alive, so a run's ceiling rests on the orchestrator's arithmetic: `implement join --ceiling N` records the ceiling and the report repeats it, and the v0.10.0 changelog says as much (recorded and reported, not enforced). Autonomous run A went over its ceiling three ways. At the start two lane agents each forked themselves into three parallel workers that no count saw, until the lane brief forbade forks. At 10:28Z on 2026-09-23 the orchestrator had six agents alive against a ruled five. During the v0.10.0 cut, at 05:06Z on 2026-09-24, it launched two cross-check agents into one free slot, five alive against four, and stopped one within a minute. Each was found by the orchestrator recounting, never by a tool. Wanted, for the implement verb (itd-2609201916151817): agent_start and agent_end kept by the verb itself, a live count derived from them, and a refusal to start an agent over the ceiling. A fork cannot be seen from outside the host, so the lane brief's no-fork rule stays the discipline for that half.

## Grounds

- pursued: an orchestrator that logs its agents is refused the fifth start under a ceiling of four; an overrun found only by the orchestrator recounting logged agents would show it wrong.
