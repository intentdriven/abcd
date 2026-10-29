---
schema_version: 1
id: "iss-2610262033248041"
slug: "05-intent-names-the-phase-plan-as-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 400 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md:74"
remedy: "In 05-intent.md lines 74 and 179 replace 'the canonical build order is the phase plan at roadmap/phases/' with dependencies (blocked_by, builds_on) plus the lifecycle shelves and build next's pick order; also fix line 12's 'scoped into a roadmap phase'."
finding_ref: cfk-6e0be541d84f19ce
---

05-intent names the phase plan as the canonical build order, but 08-abcd orders what gets built by build next's pick order.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/05-intent.md:74` — "ID number is *not* an execution-order guarantee — the canonical build order is the phase plan at"
- End B: `.abcd/development/brief/04-surfaces/08-abcd.md:338` — "read in build next's pick order (itd-2609211116005482): each READY intent is"

05-intent says the canonical build order is the phase plan at `roadmap/phases/`, although elsewhere it records that phases are retired. 08-abcd says the head of Now and the Next list are read in build next's pick order (readiness score, then age), and that the head is the intent the pick would start. The build order cannot be both the phase plan and the pick order.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 400.
