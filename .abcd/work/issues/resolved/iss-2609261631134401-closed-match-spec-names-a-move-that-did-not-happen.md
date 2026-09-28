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
resolution: "The closed spec carries a dated amendment naming what shipped: a new primitive at internal/core/record/match, since no embark overlap function existed and internal/core/record would have closed an import cycle."
impact: internal
resolved_by:
  commit: "bee225d9"
---

The closed spec spc-2609212141417782 says the filing-time overlap function is 'the one the embark ranking uses, moved to the record package' at internal/core/record/match.go. Neither happened: no embark overlap function existed, and the primitive shipped new, at internal/core/record/match (a sub-package, because record imports capture and intent). The closed record states a delivery that did not occur.

## Grounds

- pursued: the amendment states the package the code holds and that the embark ranking is unbuilt scope 7 of spc-2609211751376504; a later embark overlap function predating the match, or the primitive found elsewhere than internal/core/record/match, would show it wrong
