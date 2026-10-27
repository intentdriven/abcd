---
schema_version: 1
id: "iss-2610262044512344"
slug: "shipped-itd-6-leaves-cost-tracking-per"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1234 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:52"
remedy: "none (filed automatically)"
related_intents: [itd-6, itd-2609221009495079]
finding_ref: cfk-c62c12611ba1ad82
---

Shipped itd-6 leaves cost tracking per model to itd-17; itd-2609221009495079 records itd-17 as superseded

A sequencing impossibility (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-6-rp-mcp-only-integration.md:52` — "Cost tracking per model (covered by itd-17 model effectiveness tracking — independent concern)"
- End B: `.abcd/development/intents/planned/itd-2609221009495079-abcd-s-typed-judgements-can-run-through.md:52` — "A learned per-request router for lane models (itd-17, superseded)."

itd-6 puts cost tracking per model out of its scope because itd-17's model effectiveness tracking covers it. itd-2609221009495079 lists itd-17 as superseded. Cost tracking per model now rests on a superseded record, and neither document names a successor that takes it over.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1234.
