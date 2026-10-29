---
schema_version: 1
id: "iss-2610262042115843"
slug: "itd-48-says-role-3-kind-classification"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1052 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-48-intent-fidelity-reviewer-roles-2-3.md:24"
remedy: "none (filed automatically)"
related_intents: [itd-48]
finding_ref: cfk-76502cb6e28014c9
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262032131693. Same A line and root cause; the verification matrix marks the shape role staged, as the code does."
duplicates: [iss-2610262032131693]
---

itd-48 says Role 3 kind classification ships and `/abcd:intent shape` becomes a working surface; the verification matrix lists the shape-classification role as staged.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-48-intent-fidelity-reviewer-roles-2-3.md:24` — "With all three roles live, `/abcd:intent consistency` and `/abcd:intent shape` move from documented command surface to working command surface."
- End B: `.abcd/development/brief/06-delivery/02-verification-matrix.md:80` — "the shape-classification role that would propose a reclassification"

itd-48's press release says the intent ships Role 3, and that with all three roles live `/abcd:intent shape` becomes a working command surface. The matrix's Three intent kinds row marks as staged the shape-classification role that would propose a reclassification. A shipped intent cannot have delivered a role the matrix says has no shipped surface. itd-48's Decisions narrow it to the consistency pass, but its press release and scope still claim Role 3.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1052.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262032131693. Same A line and root cause; the verification matrix marks the shape role staged, as the code does.
