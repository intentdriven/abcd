---
schema_version: 1
id: "iss-2610262036101369"
slug: "itd-84-types-a-proposal-s-relation-to"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 629 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-84-intent-decomposition.md:118"
remedy: "none (filed automatically)"
related_intents: [itd-84]
finding_ref: cfk-059e6e9355c7dd63
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. Capture's typed links (duplicates, refines, in internal/core/record/match) state a new record's relation to existing ones, while ideate's record-grill relation states how an existing record bears on an idea (covered, contradicted, superseded). No record or code requires one closed set, so this is a naming parallel, not a conflict."
---

itd-84 types a proposal's relation to existing records as supersedes/reverses/duplicates/refines; ideate's record grill types the same relation as covered/contradicted/superseded.

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-84-intent-decomposition.md:118` — "Typed cross-record links (`supersedes` / `reverses` / `duplicates` / `refines`)"
- End B: `.abcd/development/brief/04-surfaces/18-ideate.md:46` — "carrying a relation (`covered` / `contradicted` / `superseded`)"

itd-84 requires typed cross-record links (supersedes, reverses, duplicates, refines) for the existing records a new proposal touches. 18-ideate's record grill records hits on the existing record carrying a relation of covered, contradicted or superseded. Both judge a new proposal against the existing record, yet name the same relation with two closed vocabularies (duplicates and covered, reverses and contradicted).

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 629.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. Capture's typed links (duplicates, refines, in internal/core/record/match) state a new record's relation to existing ones, while ideate's record-grill relation states how an existing record bears on an idea (covered, contradicted, superseded). No record or code requires one closed set, so this is a naming parallel, not a conflict.
