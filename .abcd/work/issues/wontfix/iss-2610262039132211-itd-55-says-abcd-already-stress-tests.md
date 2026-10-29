---
schema_version: 1
id: "iss-2610262039132211"
slug: "itd-55-says-abcd-already-stress-tests"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 846 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-55-first-principles-foundations-auditor.md:17"
remedy: "none (filed automatically)"
related_intents: [itd-55]
finding_ref: cfk-bd59f584517e738b
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262034293679. Same draft edit as iss-2610262034293679, set against a different statement of the same current state."
duplicates: [iss-2610262034293679]
---

itd-55 says abcd already stress-tests acceptance criteria with the grill, but the surfaces register says no grill ships

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-55-first-principles-foundations-auditor.md:17` — "abcd already stress-tests acceptance criteria (the grill)"
- End B: `.abcd/development/brief/04-surfaces/README.md:395` — "per-session output are command-shaped, and the `intent` parent ships no `grill`"

itd-55 positions the foundations auditor against an existing grill and scopes itself out of duplicating `/abcd:intent grill`. The surfaces register says the `intent` parent ships no `grill` sub-verb, and that the grill is a design target. The line itd-55 draws is against a surface that does not exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 846.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262034293679. Same draft edit as iss-2610262034293679, set against a different statement of the same current state.
