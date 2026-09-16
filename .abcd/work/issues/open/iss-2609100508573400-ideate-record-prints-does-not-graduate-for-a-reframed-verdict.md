---
schema_version: 1
id: "iss-2609100508573400"
slug: "ideate-record-prints-does-not-graduate-for-a-reframed-verdict"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-09/10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal (ideate record)"
---

`abcd ideate record` prints "the idea does not graduate" for a `reframed` verdict, which reads as "killed" to the operator receiving it.

Observed recording an ideate verdict during an autonomous run in a managed repository. A `reframed` verdict is not a rejection: the idea survives in a different shape, and the whole point of recording it is that the reframing is the output. The message conflates it with the outcome where nothing survives, so an operator (or a downstream session reading the record) takes a reframed idea for a dead one.

Wanted: distinct wording per verdict — say what the reframing was, or at minimum "the idea does not graduate in its original shape; it was reframed" — so the terminal message matches the verdict the record carries. This is the only defect the run found in a set of record verbs that otherwise all worked, which is filed separately as a positive observation.
