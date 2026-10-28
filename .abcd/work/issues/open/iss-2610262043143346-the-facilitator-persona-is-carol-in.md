---
schema_version: 1
id: "iss-2610262043143346"
slug: "the-facilitator-persona-is-carol-in"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1124 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-115-abcd-managed-repos-merge-prs-without-the.md:29"
remedy: "none (filed automatically)"
related_intents: [itd-115, itd-120]
finding_ref: cfk-e87b36002a8c58fd
refines: [iss-371]
---

The facilitator persona is Carol in itd-115 and Nia in itd-120.

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-115-abcd-managed-repos-merge-prs-without-the.md:29` — "said Carol, a facilitator"
- End B: `.abcd/development/intents/shipped/itd-120-a-resolved-issue-points-at-what-fixed-it.md:23` — "says Nia, facilitator"

Under the persona registry rule the role's registered name is used, so the facilitator has one name. itd-115 attributes its quote to 'Carol, a facilitator'; itd-120 (and itd-121) attribute theirs to 'Nia, facilitator'. Two names are in use for one role.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1124.
