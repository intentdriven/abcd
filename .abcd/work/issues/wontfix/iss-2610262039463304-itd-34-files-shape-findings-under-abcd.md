---
schema_version: 1
id: "iss-2610262039463304"
slug: "itd-34-files-shape-findings-under-abcd"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 885 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/04-universal-patterns.md:94"
remedy: "none (filed automatically)"
related_intents: [itd-34]
finding_ref: cfk-c9127c078b228674
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. No shape verb ships and the brief carries the retirement ruling; the shipped intent's path was the plan of its day."
---

itd-34 files shape findings under `.abcd/logbook/`, a name 04-universal-patterns retired and says must not be re-minted.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/04-universal-patterns.md:94` — "Its root was once `.abcd/logbook/`; a 2026-07-12 adjudication moved run output to the local ephemeral tier and retired that name, which must not be re-minted"
- End B: `.abcd/development/intents/shipped/itd-34-three-intent-kinds.md:90` — "Findings live at `.abcd/logbook/audit/shape-<ts>/report.{json,md}`."

itd-34 says shape findings live at `.abcd/logbook/audit/shape-<ts>/report.{json,md}`. 04-universal-patterns says the log root was once `.abcd/logbook/`, that a 2026-07-12 adjudication moved run output to the local tier and retired the name, and it draws the shape reports at `audit/shape-<ts>/` under `.abcd/.work.local/logs/`. itd-34's decisions date from 2026-09-21, after the retirement, so it uses a retired name for a location the brief puts elsewhere.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 885.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. No shape verb ships and the brief carries the retirement ruling; the shipped intent's path was the plan of its day.
