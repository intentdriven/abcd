---
schema_version: 1
id: "iss-2610262044537770"
slug: "shipped-itd-69-says-the-brief-change"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1237 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-69-plugin-metadata-lockstep-update.md:43"
remedy: "none (filed automatically)"
related_intents: [itd-69, itd-61]
finding_ref: cfk-23f455c567048a6c
---

Shipped itd-69 says the brief-change derivation gate (itd-61) produced it; itd-61 is still a draft that has not yet given abcd that gate

A sequencing impossibility (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-69-plugin-metadata-lockstep-update.md:43` — "Drawn out from a human brief edit by the brief-change derivation gate"
- End B: `.abcd/development/intents/drafts/itd-61-brief-change-derivation.md:24` — "abcd gains a brief-change derivation: when a human edits the brief directly"

itd-69's provenance line says the brief-change derivation gate of itd-61 turned a human brief edit into this intent. itd-61 is in drafts, and its press release describes the derivation as something abcd will gain. A shipped record cannot have come from a gate whose own record is not yet delivered. Either the gate exists in a form itd-61 does not record, or itd-69's provenance is wrong.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1237.
