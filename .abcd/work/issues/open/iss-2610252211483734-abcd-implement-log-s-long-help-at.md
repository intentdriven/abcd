---
schema_version: 1
id: "iss-2610252211483734"
slug: "abcd-implement-log-s-long-help-at"
severity: "nitpick"
category: "documentation"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D6"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/implement.go"
remedy: "Say the verb-owned events are the claim, window, session, load and outage events, and scope the time and number rule to the events that require or check those fields, then regenerate the CLI reference."
---

abcd implement log's Long help at internal/surface/cli/implement.go:503-508 (page lines 2468-2474) overstates and understates its checks. It says the claim, window and session events are refused, but the load event and the four outage events are refused too (internal/core/implement/log.go:84-89,405-407). It says 'an at or last_productive is an RFC 3339 time, and a *_min or minutes field a number', but those fields are checked only on the events whose rules name them (intervention, decision, stop, agent_end, ceiling_overrun and backoff: log.go:126-141,168-200,457-466); lane_close --field minutes=abc or gate_run --field at=yesterday is written as given.
