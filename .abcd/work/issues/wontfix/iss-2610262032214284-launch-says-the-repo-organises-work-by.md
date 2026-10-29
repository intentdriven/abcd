---
schema_version: 1
id: "iss-2610262032214284"
slug: "launch-says-the-repo-organises-work-by"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 256 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/04-launch.md:489"
remedy: "none (filed automatically)"
finding_ref: cfk-f30736abcaf6c814
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262030345408. Same End A (04-launch.md line 489, work organised by phase); only End B differs (the disembark chapter's account of retired phases)."
duplicates: [iss-2610262030345408]
---

Launch says the repo organises work by phase; disembark says sequence is dependencies plus the lifecycle shelves, with phases retired

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/04-launch.md:489` — "the repo organises work by **phase**"
- End B: `.abcd/development/brief/04-surfaces/02-disembark.md:18` — "sequence is dependencies plus the lifecycle shelves ([adr-2609212115255771] (../../decisions/adrs/2609212115255771-phases-and-milestones-are-retired.md))"

Launch's versioning section grounds versions-as-output on the repo organising its work by phase. Disembark records that sequence is dependencies plus the lifecycle shelves, citing the decision that retires phases and milestones, and calls the phase document it names retired. A reader cannot both organise work by phase and treat phases as retired.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 256.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262030345408. Same End A (04-launch.md line 489, work organised by phase); only End B differs (the disembark chapter's account of retired phases).
