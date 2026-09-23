---
schema_version: 1
id: "iss-2608261550580260"
slug: "prompt-router-has-no-removal-signal-for-snapshotting-clients"
severity: "minor"
category: "future-work-seed"
source: "impl-review"
found_during: "second-harness adaptor lab review (2026-08-24/26)"
found_at: "internal/core/rules/inject.go"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (ruling owed: removal-signal shape for snapshotting clients (full active-domain set vs tombstones) in the prompt-router protocol). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

The prompt-router emits only domains whose content changed this turn, so a client that snapshots injected rules — a host-side adaptor staging them into a system prompt, or a future MCP consumer — keeps a domain that was deleted or renamed mid-session until session end: there is no removal signal in the protocol. Emit the full active-domain-name set, or tombstones for removals, so a snapshotting client can prune. A local second-harness adaptor lab hit this and had to accept the lingering snapshot as a recorded gap.