---
schema_version: 1
id: "iss-2611060809210526"
slug: "abcd-development-brief-04-surfaces"
severity: "nitpick"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/23-reading.md"
remedy: "Count two unrecorded refusals, adding the rerun of a run that already has an outcome and why it writes no record."
resolution: "The reading brief now counts two unrecorded refusals past the identity point: the rerun of a run that already has an outcome, which writes nothing so it cannot overwrite that outcome, and the prose citation. The matching code comment in ingest.go is corrected too, and TestTheBriefCountsEveryUnrecordedRefusal holds the count."
impact: internal
resolved_by:
  commit: "36b46f3ea0d20804d8e8f2a3aa7b9a007eae504d"
---

`.abcd/development/brief/04-surfaces/23-reading.md:282` says one refusal after the identity point is deliberately unrecorded, the prose citation. `refuseARerun`, called at `internal/core/reading/ingest.go:614` after the identity is proven, also refuses and writes no refusal record, because a run that already has an outcome must not have it overwritten.

## Grounds

- pursued: the brief's refusal section matches Ingest, where refuseARerun runs before refuse() and writes no refusal record; it would be shown wrong if a rerun of a run with an outcome wrote a refusal record (TestARerunOfACommittedRunIsRefused holds that it does not)
