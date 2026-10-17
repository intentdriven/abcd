---
schema_version: 1
id: "iss-2610142134342766"
slug: "place-route-return-keeps-updated-at"
severity: "nitpick"
category: "bug"
source: "review-followup"
found_during: "PR-976-review-round-5"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/validate.go"
remedy: "Pass the step's clock into placeReturn and set st.UpdatedAt when it records the answer on the await, as every other state mutation does."
---

placeRouteReturn (placeReturn) writes the run state when it records the route's answer on the await, but does not bump the state's UpdatedAt, so the run reads as unchanged since its last stamped mutation although its state moved.
