---
schema_version: 1
id: "iss-2610262043598494"
slug: "itd-98-leaves-model-level-benchmarking"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1176 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-98-solo-vs-duo-is-measured-not-debated.md:68"
remedy: "In itd-98 drop itd-17 from the owners of model-level benchmarking (itd-17 is superseded by itd-2609221009495079, which rules a learned router out) and name itd-64 alone, plus itd-2609221009495079 where typed-judgement shadow runs are meant; fix the context line and References too."
related_intents: [itd-98, itd-2609221009495079]
finding_ref: cfk-1c1e37ff017db428
resolution: "itd-98 now names itd-64 alone as the owner of model-level benchmarking, and itd-2609221009495079 for the shadow runs of typed judgements; the context line and References no longer hand substrate to the superseded itd-17. A test in internal/core/intent fails on any itd-98 line that names a superseded intent without saying so."
impact: internal
resolved_by:
  commit: "4371b0067ea86122415d697229e2ca9a0c3c220d"
---

itd-98 leaves model-level benchmarking to itd-17; itd-2609221009495079 records itd-17 as superseded

A sequencing impossibility (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-98-solo-vs-duo-is-measured-not-debated.md:68` — "itd-17 and itd-64 own model-level"
- End B: `.abcd/development/intents/planned/itd-2609221009495079-abcd-s-typed-judgements-can-run-through.md:52` — "A learned per-request router for lane models (itd-17, superseded)."

itd-98 keeps model benchmarking out of its own scope because itd-17 and itd-64 own it. itd-2609221009495079 marks itd-17 as superseded and excludes its learned-router scope. Half of the dependency itd-98 names points at a record that no longer exists to satisfy it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1176.

## Grounds

- pursued: every line of itd-98 naming an intent names one still able to own its scope (itd-17 only as superseded); shown wrong if TestItd98NamesNoSupersededOwner fails, or a reader of itd-98 finds a scope or substrate handed to a superseded record
