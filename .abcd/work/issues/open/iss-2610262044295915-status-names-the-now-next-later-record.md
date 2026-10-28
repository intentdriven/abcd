---
schema_version: 1
id: "iss-2610262044295915"
slug: "status-names-the-now-next-later-record"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1210 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2609151838312703-sessions-announce-what-they-are-doing.md:120"
remedy: "none (filed automatically)"
related_intents: [itd-2609151838312703, itd-2609211913453478]
finding_ref: cfk-736aec2f98b40512
---

`status` names the Now/Next/Later record family in the glossary and the people's view of announcements and reservations in the planned verb

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2609151838312703-sessions-announce-what-they-are-doing.md:120` — "`abcd status` is the people's view of what everyone else is doing: announcements, who holds which reservation and any release being cut (C8, C16)."
- End B: `.abcd/development/intents/shipped/itd-2609211913453478-one-page-in-the-glossary-maps-abcd-s.md:39` — "intent, spec, step, bundle, issue, release, status (Now / Next / Later)"

itd-2609211913453478 puts status (Now / Next / Later) on the one-term-per-concept record-families page. itd-2609151838312703 names a new verb `abcd status` for announcements, reservation holders and releases being cut. Renaming the status board to the dashboard does not rename the status block or the glossary family, so one word ends up with two meanings.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1210.
