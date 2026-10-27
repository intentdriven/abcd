---
schema_version: 1
id: "iss-2610262034489695"
slug: "itd-114-s-maintainer-reads-capture-list"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 524 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-114-abcd-mints-collision-proof-record-ids.md:32"
remedy: "none (filed automatically)"
related_intents: [itd-114]
finding_ref: cfk-c51d1c3211a14744
---

itd-114's maintainer reads capture list newest-first; the capture chapter orders the list by derived priority, unblocked first and then by severity.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-114-abcd-mints-collision-proof-record-ids.md:32` — "I still read `capture list` newest-first at a glance"
- End B: `.abcd/development/brief/04-surfaces/06-capture.md:588` — "in derived-priority order: unblocked first, then severity"

itd-114's press release has the maintainer say they still read capture list newest-first at a glance after the id change. 06-capture's acceptance states the listing comes in derived-priority order: unblocked first, then severity, with blocked rows demoted and annotated. A list ordered by priority is not read newest-first, so the intent claims an ordering the surface does not give.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 524.
