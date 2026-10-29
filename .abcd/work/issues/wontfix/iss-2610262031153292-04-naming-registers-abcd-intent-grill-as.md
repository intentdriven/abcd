---
schema_version: 1
id: "iss-2610262031153292"
slug: "04-naming-registers-abcd-intent-grill-as"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 105 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:117"
remedy: "none (filed automatically)"
finding_ref: cfk-5b24327cf53233d7
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262030468053. Same claim in the same register (grill staged under itd-27) against the same fact (itd-27 superseded, grilling delivered by the planning interview under itd-94)."
duplicates: [iss-2610262030468053]
---

04-naming registers /abcd:intent grill as a staged sub-verb under itd-27; the build sequence says itd-27's grilling is the planning interview under itd-94, which supersedes it.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:117` — "`/abcd:intent grill <itd-N>` (itd-27)"
- End B: `.abcd/development/brief/06-delivery/01-build-sequence.md:100` — "the grilling itd-27 asked for is the planning interview"

04-naming lists '`/abcd:intent grill <itd-N>` (itd-27)' among the sub-verbs that are staged, named to hold the shape, which under the truth rule is a decided design target. The build sequence says there is no grill sub-verb because 'the grilling itd-27 asked for is the planning interview', abcd intent interview, under itd-94, which supersedes it. A sub-verb cannot remain the staged target of itd-27 when itd-27 is superseded and its grilling is delivered by another verb.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 105.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262030468053. Same claim in the same register (grill staged under itd-27) against the same fact (itd-27 superseded, grilling delivered by the planning interview under itd-94).
