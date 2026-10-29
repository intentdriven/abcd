---
schema_version: 1
id: "iss-2610262041455139"
slug: "the-verification-matrix-says-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1023 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/02-verification-matrix.md:73"
remedy: "In the Rules-loader dedup row say the refresh is event-driven (SessionStart and PreCompact clear the ledger) with a counted backstop, every 15th prompt by default and settable as rules.force_refresh_every_n; delete 'Nothing counts prompts'."
related_intents: [itd-3]
finding_ref: cfk-c89b936d0f833a9f
---

The verification matrix says the rules-loader refresh is event-driven and nothing counts prompts; itd-3 refreshes in full every N turns.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/06-delivery/02-verification-matrix.md:73` — "The refresh is event-driven rather than turn-counted"
- End B: `.abcd/development/intents/shipped/itd-3-modular-rules-loader.md:30` — "Signature-based dedup; force full re-inject every N turns (N defaults to 5, configurable)."

The matrix's dedup row says the refresh is event-driven rather than turn-counted. Only SessionStart and PreCompact clear the dedup ledger, and nothing counts prompts. The shipped itd-3 scope promises a full re-injection every N turns, with N defaulting to 5 and configurable. A gate written from the matrix would fail the behaviour itd-3 promises, and the reverse is also true.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1023.
