---
schema_version: 1
id: "iss-2609021857343626"
slug: "ruling-owed-on-the-reading-of-at-the-target-in-adr-260902101"
severity: "major"
category: "process"
source: "impl-review"
found_during: "cold-reading Phase A rehearsal"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/candidates.go"
deferred_after: "v0.7.1"
deferral_reason: "The ruling belongs with the workstream, and the workstream has moved. The cold-reading instrument was lifted to a separate fork during this cycle to be restarted there, so the reading of 'at the target' in adr-2609021016272867 is a question for the design that continues in that fork rather than for a release cut of the tool. The code the ruling governs still ships here, which is why this is deferred rather than closed: the question is live, its owner is elsewhere, and it returns at the next anchor."
wontfix_reason: "Out of scope for this repository (product thinker, 2026-09-23 run A interview, M21): the cold-reading research workstream lives in a separate fork, and the reading of 'at the target' in adr-2609021016272867's comparative derivation is ruled there, with that workstream. Recorded in DECISIONS.md 2026-09-23."
---

ruling owed on the reading of 'at the target' in adr-2609021016272867 for the comparative derivation; the implementation accepts a widening run whose target is an ancestor of the comparative's target when only the readings store changed between them, so the run's own records can be committed between ingest and the next reading; the alternative is an assembly whose target may differ from HEAD at the comparative position, which reads no tree; the maintainer chooses and the register gains the entry

## Grounds

- declined: the ruling's owner is the fork's workstream; this would be wrong if the comparative derivation's reading changed code that ships from this repository, which would bring the question back here
