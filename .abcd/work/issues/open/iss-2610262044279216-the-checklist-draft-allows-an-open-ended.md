---
schema_version: 1
id: "iss-2610262044279216"
slug: "the-checklist-draft-allows-an-open-ended"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1207 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2610170638184486-every-abcd-session-keeps-a-checklist-on.md:296"
remedy: "none (filed automatically)"
related_intents: [itd-2610170638184486, itd-2610180912189125]
finding_ref: cfk-dcdc451f90066d8e
---

The checklist draft allows an open-ended away state, but the shipped away mode is capped at seven days

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2610170638184486-every-abcd-session-keeps-a-checklist-on.md:296` — "in the app, either until a set time or open-ended; open-ended, they say when they are back"
- End B: `.abcd/development/intents/shipped/itd-2610180912189125-a-session-told-to-work-on-while-its.md:78` — "when it ends, at most seven days on; `abcd mode back` ends it"

itd-2610170638184486 says the product thinker can mark themselves away until a set time or open-ended, ending it by saying they are back. itd-2610180912189125's `abcd mode autonomous` records an end at most seven days on. An open-ended away cannot be recorded under the shipped mode.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1207.
