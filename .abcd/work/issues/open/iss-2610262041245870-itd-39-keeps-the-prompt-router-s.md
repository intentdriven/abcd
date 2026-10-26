---
schema_version: 1
id: "iss-2610262041245870"
slug: "itd-39-keeps-the-prompt-router-s"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 998 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-39-scope-aware-memory-retrieval.md:33"
remedy: "none (filed automatically)"
related_intents: [itd-39]
finding_ref: cfk-512c962647190769
---

itd-39 keeps the prompt router's force-refresh-every-N; the verification matrix says the refresh is event-driven and nothing counts prompts

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-39-scope-aware-memory-retrieval.md:33` — "Same recall scan, same signature dedup, same force-refresh-every-N."
- End B: `.abcd/development/brief/06-delivery/02-verification-matrix.md:73` — "The refresh is event-driven rather than turn-counted"

itd-39 says memory recall keeps the router's "same force-refresh-every-N", a refresh triggered by counting prompts. The verification matrix's rules-loader dedup row says the refresh is event-driven rather than turn-counted: SessionStart and PreCompact clear the dedup ledger, and nothing counts prompts. A refresh cannot be both prompt-counted and not prompt-counted, so the planned intent depends on a mechanism the brief says is absent.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 998.
