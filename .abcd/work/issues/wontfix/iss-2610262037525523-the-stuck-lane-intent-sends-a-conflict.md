---
schema_version: 1
id: "iss-2610262037525523"
slug: "the-stuck-lane-intent-sends-a-conflict"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 755 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/34-build.md:436"
remedy: "none (filed automatically)"
related_intents: [itd-2610161016391855]
finding_ref: cfk-2ded93aa86eecb39
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The intent's 'as a fix round' is the planning-time wording; the later decision and the code (loop/sync.go) say a sync is not a failed attempt."
---

The stuck-lane intent sends a conflict on generated files back to the implementer as a fix round; the build chapter says a sync counts no fix round.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/34-build.md:436` — "goes to a fresh implementer with a sync brief; its receipt must carry the merged sha as an ancestor of its head. A sync counts no fix round."
- End B: `.abcd/development/intents/shipped/itd-2610161016391855-a-drain-lane-that-cannot-finish-no.md:91` — "a conflict on declared generated files goes back to the implementer as a fix round"

itd-2610161016391855 records that a conflict on declared generated files during a lane's sync goes back to the implementer as a fix round. 34-build.md sends a conflicting sync merge to a fresh implementer with a sync brief and states that a sync counts no fix round. Whether that regeneration spends one of the lane's fix rounds decides when the fix-round cap hands the lane back as unachievable, and the two documents answer it in opposite ways.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 755.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The intent's 'as a fix round' is the planning-time wording; the later decision and the code (loop/sync.go) say a sync is not a failed attempt.
