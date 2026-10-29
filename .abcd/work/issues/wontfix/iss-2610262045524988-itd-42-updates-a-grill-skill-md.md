---
schema_version: 1
id: "iss-2610262045524988"
slug: "itd-42-updates-a-grill-skill-md"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1303 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-42-coherence-aware-grill.md:56"
remedy: "none (filed automatically)"
related_intents: [itd-42, itd-2610090831227812]
finding_ref: cfk-3238ce45865892a8
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The repository has no skills directory and no grill flags; the flag renaming sits in the press release and scope the banner says to read through the re-scope."
---

itd-42 updates a grill SKILL.md; itd-2610090831227812 says abcd ships zero skills

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-42-coherence-aware-grill.md:56` — "the grill `SKILL.md` flag list are updated accordingly"
- End B: `.abcd/development/intents/shipped/itd-2610090831227812-a-person-who-types-abcd-sees-only-the.md:54` — "abcd ships zero skills and the `/abcd:` namespace is commands only"

itd-42 renames the grill flags and updates the grill SKILL.md flag list to match, treating a grill skill as part of the shipped surface. itd-2610090831227812 states that abcd ships zero skills and that the /abcd: namespace is commands only. The grill's flags cannot live in a skill that does not exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1303.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The repository has no skills directory and no grill flags; the flag renaming sits in the press release and scope the banner says to read through the re-scope.
