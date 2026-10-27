---
schema_version: 1
id: "iss-2610262038033829"
slug: "itd-6-selects-the-reviewer-route-with"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 768 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:43"
remedy: "none (filed automatically)"
related_intents: [itd-6]
finding_ref: cfk-f28b9427cdae4f31
---

itd-6 selects the reviewer route with `oracle.backend` (rp, codex, in-session); 34-build selects it with `oracle.review` (host, rp)

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:43` — "If reachable: lock `oracle.backend = "rp"`. If not: lock `oracle.backend = "codex"` (if Codex CLI present) or `"in-session"` (final fallback)"
- End B: `.abcd/development/brief/04-surfaces/34-build.md:544` — "`oracle.review` in the layered configuration names `host`, the bundled default, or `rp`"

itd-6 has the route locked as `oracle.backend` with the values `rp`, `codex` or `in-session`, while 34-build reads the same choice of reviewer route from `oracle.review`, whose only values are `host` (the bundled default) and `rp`. One setting carries two key names and two value sets, so a configuration written to one document's key is not the one the other reads.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 768.
