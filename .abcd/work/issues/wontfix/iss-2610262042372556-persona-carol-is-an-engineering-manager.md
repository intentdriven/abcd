---
schema_version: 1
id: "iss-2610262042372556"
slug: "persona-carol-is-an-engineering-manager"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1082 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-12-work-adapter-weighting.md:17"
remedy: "none (filed automatically)"
related_intents: [itd-12, itd-115]
finding_ref: cfk-6a9ac1c73bd16649
refines: [iss-300, iss-371]
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262043143346. Same wrong attribution in itd-115 line 29 seen from the other side; the registry makes Carol an engineering manager, so itd-12 is the correct end."
duplicates: [iss-2610262043143346]
---

Persona Carol is an engineering manager in itd-12 and a facilitator in itd-115

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-12-work-adapter-weighting.md:17` — "said Carol, engineering manager"
- End B: `.abcd/development/intents/drafts/itd-115-abcd-managed-repos-merge-prs-without-the.md:29` — "said Carol, a facilitator"

Under role-first persona selection a registered name maps to one role. itd-12 quotes Carol as an engineering manager and itd-115 quotes Carol as a facilitator, two different roles under one name.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1082.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262043143346. Same wrong attribution in itd-115 line 29 seen from the other side; the registry makes Carol an engineering manager, so itd-12 is the correct end.
