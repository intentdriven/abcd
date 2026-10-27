---
schema_version: 1
id: "iss-2610262045552105"
slug: "itd-46-lists-ship-as-a-live-intent"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1307 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-46-abcd-intent-quoted-text-create-symmetric.md:76"
remedy: "none (filed automatically)"
related_intents: [itd-46, itd-2609111003026787]
finding_ref: cfk-7109fcd7556b314a
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262042088869. Same itd-46 sentence and root cause; itd-2609111003026787 holds intent ship and relies on spec close with a trailer."
duplicates: [iss-2610262042088869]
---

itd-46 lists ship as a live intent sub-verb; itd-2609111003026787 holds intent ship for a cycle

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-46-abcd-intent-quoted-text-create-symmetric.md:76` — "`grill`, `plan`, `ship`, `review`, `reclassify`, `link`, `resolve`,"
- End B: `.abcd/development/intents/shipped/itd-2609111003026787-a-gate-catches-the-intent-whose-work-is.md:98` — "**`intent ship` and the intent supersede verb.** Held for a cycle by ruling."

itd-46 names ship among the mutating /abcd:intent sub-verbs that continue to require their target's ID, an already-correct shape it leaves alone. itd-2609111003026787 puts intent ship out of scope, held for a cycle by ruling, and relies on spec close and a trailer instead. The surface either has the sub-verb or it does not.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1307.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262042088869. Same itd-46 sentence and root cause; itd-2609111003026787 holds intent ship and relies on spec close with a trailer.
