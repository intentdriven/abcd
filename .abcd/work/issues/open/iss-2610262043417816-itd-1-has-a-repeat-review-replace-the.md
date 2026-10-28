---
schema_version: 1
id: "iss-2610262043417816"
slug: "itd-1-has-a-repeat-review-replace-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1155 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-1-acceptance-gates.md:31"
remedy: "none (filed automatically)"
related_intents: [itd-1, itd-80]
finding_ref: cfk-ac075eab622bb8c8
---

itd-1 has a repeat review replace the one Audit Notes block in place; itd-80 appends each ingested verdict, keyed on its receipt

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-1-acceptance-gates.md:31` — "the section never accumulates stale blocks; git history is the prior-review trail"
- End B: `.abcd/development/intents/shipped/itd-80-intent-lifecycle-automation.md:36` — "appends idempotently (keyed on the receipt id) to `## Audit Notes`"

itd-1 says the writer keeps a single machine-fenced review block that a repeat review replaces, so the section never collects stale blocks and git history holds earlier reviews. itd-80 says the ingest appends to Audit Notes and is idempotent only per receipt id, so a second review with a new receipt adds another block. The section cannot both hold one replaced block and gain one block per receipt.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1155.
