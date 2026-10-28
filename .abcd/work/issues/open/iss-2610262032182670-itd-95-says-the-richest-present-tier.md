---
schema_version: 1
id: "iss-2610262032182670"
slug: "itd-95-says-the-richest-present-tier"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 250 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-95-disembark-grounds-a-lifeboat-s-open.md:42"
remedy: "none (filed automatically)"
related_intents: [itd-95]
finding_ref: cfk-df0fcfb732add493
---

itd-95 says the richest present tier wins a section; the meta page says tiers are cumulative, so a richer tier can never ground a section worse than a poorer one

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-95-disembark-grounds-a-lifeboat-s-open.md:42` — "it does not change the rule that the richest present tier wins a section"
- End B: `.abcd/development/brief/00-meta.md:52` — "A richer tier can therefore never ground a section worse than a poorer one."

itd-95 describes the reduction rule it leaves unchanged as 'the richest present tier wins a section'. Under that rule, an abcd-native tier that finds nothing can override open questions that the conventions tier grounds from TODO markers. 00-meta says a richer tier can never ground a section worse than a poorer one. The first rule lets a section come back worse at a richer tier; the second forbids it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 250.
