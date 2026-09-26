---
schema_version: 1
id: "iss-2609261631134401"
slug: "closed-match-spec-names-a-move-that-did-not-happen"
severity: "minor"
category: "drift"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-match"
origin: researcher-authored
production_mode: hand-written
---

The closed spec spc-2609212141417782 says the filing-time overlap function is 'the one the embark ranking uses, moved to the record package' at internal/core/record/match.go. Neither happened: no embark overlap function existed, and the primitive shipped new, at internal/core/record/match (a sub-package, because record imports capture and intent). The closed record states a delivery that did not occur.
