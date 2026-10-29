---
schema_version: 1
id: "iss-2610262043466977"
slug: "itd-25-s-dredge-promote-calls-abcd"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1161 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-25-dredge-cross-corpus-synthesist.md:36"
remedy: "none (filed automatically)"
related_intents: [itd-25, itd-46]
finding_ref: cfk-bbbbed176917234d
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262031315138. Same draft edit as iss-2610262031315138, set against a different statement of the same current state."
duplicates: [iss-2610262031315138]
---

itd-25's dredge promote calls /abcd:intent new; itd-46 replaced the new sub-verb with quoted-text create

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-25-dredge-cross-corpus-synthesist.md:36` — "calls `/abcd:intent new` with the cluster summary"
- End B: `.abcd/development/intents/shipped/itd-46-abcd-intent-quoted-text-create-symmetric.md:18` — "files the new intent directly — no `new` sub-verb required"

itd-25 has dredge promote create its intent draft by calling /abcd:intent new. itd-46 shipped quoted-text create, which files the intent with no new sub-verb. On the surface itd-46 shipped, the verb itd-25 calls is gone.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1161.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262031315138. Same draft edit as iss-2610262031315138, set against a different statement of the same current state.
