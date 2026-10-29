---
schema_version: 1
id: "iss-2610262042040605"
slug: "the-out-of-scope-chapter-says-intents"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1044 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/03-out-of-scope.md:172"
remedy: "none (filed automatically)"
related_intents: [itd-2609211913453478]
finding_ref: cfk-f377600bbb7958ff
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262040267270. Identical A end and the same retirement ruling, quoted through the vocabulary intent instead of the build sequence."
duplicates: [iss-2610262040267270]
---

The out-of-scope chapter says intents enter work by being scoped into a phase; the vocabulary intent retired phases in favour of dependencies and the lifecycle shelves.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/06-delivery/03-out-of-scope.md:172` — "A later-phase intent enters work by being scoped into a phase"
- End B: `.abcd/development/intents/shipped/itd-2609211913453478-one-page-in-the-glossary-maps-abcd-s.md:55` — "Phases and milestones are retired; sequencing is dependencies plus the lifecycle shelves"

03-out-of-scope describes its work as 'bundled into the eight planned phases'. It says a later-phase intent enters work by being scoped into a phase and then planned. The shipped vocabulary intent rules that phases and milestones are retired and that sequencing is dependencies plus the lifecycle shelves. The chapter prescribes an entry step through a mechanism the record has retired.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1044.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262040267270. Identical A end and the same retirement ruling, quoted through the vocabulary intent instead of the build sequence.
