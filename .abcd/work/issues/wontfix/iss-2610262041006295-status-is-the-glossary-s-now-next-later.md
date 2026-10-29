---
schema_version: 1
id: "iss-2610262041006295"
slug: "status-is-the-glossary-s-now-next-later"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 970 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/record-families.md:28"
remedy: "none (filed automatically)"
related_intents: [itd-2609151838312703]
finding_ref: cfk-0606e545e6ec84a2
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262038254260. Same End A and the same decision 21; the rename is already captured as iss-2610150726321422."
duplicates: [iss-2610262038254260]
---

'status' is the glossary's Now/Next/Later record family, and the announcements intent gives the same name to `abcd status`, a view of announcements, reservations and release cuts.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/record-families.md:28` — "Now / Next / Later, rendered from the shelves, the gate and the build's state"
- End B: `.abcd/development/intents/planned/itd-2609151838312703-sessions-announce-what-they-are-doing.md:120` — "`abcd status` is the people's view of what everyone else is doing: announcements, who holds which reservation and any release being cut"

record-families.md defines the status family as the Now / Next / Later view computed from the shelves, the gate and the build's state. itd-2609151838312703 decision 21 names `abcd status` as the view of everyone's announcements, reservation holders and releases being cut, saying a board rename frees the word. The glossary still ties 'status' to the Now/Next/Later view, so one name now covers two different views.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 970.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262038254260. Same End A and the same decision 21; the rename is already captured as iss-2610150726321422.
