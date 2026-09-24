---
schema_version: 1
id: "iss-2609240646549900"
slug: "implement-log-has-no-ceiling-overrun-event"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/log.go"
---

`abcd implement log` has no event for going over the ceiling. Its loggable events are backoff, lane_open, lane_close, agent_start, agent_end, ceiling_wait, gate_run, review, fallback, stop, refusal, pr, capture and context. When the orchestrator of autonomous run A launched a fifth agent under a ceiling of four during the v0.10.0 cut, the only fit was `refusal` with a hand-chosen `kind=ceiling_overrun` field, and the earlier breach of 2026-09-23 (six alive against five) went in as a `refusal` with the condition in prose. `abcd implement report` counts ceiling waits and cannot count breaches, so the run's report is silent on the one ceiling figure that went wrong. Wanted: a `ceiling_overrun` event (agents alive, the ceiling, the lane, the minutes over) and a report column for it.
