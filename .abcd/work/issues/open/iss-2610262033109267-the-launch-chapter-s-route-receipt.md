---
schema_version: 1
id: "iss-2610262033109267"
slug: "the-launch-chapter-s-route-receipt"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 370 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/04-launch.md:937"
remedy: "none (filed automatically)"
related_intents: [itd-166]
finding_ref: cfk-aeaae6b3acc0ffa5
---

The launch chapter's route receipt leaves an unreported model empty and has no served side; itd-166 says the receipt records both sides and gives a reason for every missing field

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/04-launch.md:937` — "`model` field (a reading's `instrument.model`) and empty when the payload names"
- End B: `.abcd/development/intents/shipped/itd-166-each-run-should-record-the-model.md:18` — "Whatever the route cannot show is written down as unpinned or self-reported, never left out and never guessed."

The launch chapter lists the route receipt's fields (tier_asked, connection_tried, connection_used, fallback_reason, override, settings_sent, model_reported) and says model_reported is empty when the payload names no model. itd-166 says every model call now records both what was asked and what answered (model, provider, token counts, cost), and that these two sides extend the override's receipt. It also says whatever a route cannot show is written down as unpinned or self-reported, never left out. A field left empty with no reason, and a receipt with no served side, contradict that. The disembark chapter carries the same receipt sentence.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 370.
