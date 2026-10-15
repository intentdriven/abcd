---
schema_version: 1
id: "iss-2610142151428802"
slug: "aside-exhaustion-names-two-places"
severity: "nitpick"
category: "inconsistency"
source: "review-followup"
found_during: "PR-976-final-review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/validate.go"
remedy: "Name the same thing in both exhaustion refusals: the review's return path, passed from keepAside into createAside."
---

createAside's refusal when every numbered aside name is taken names the review's directory, while keepAside's refusal for the same exhaustion names the review's return path, so one condition reads two ways.
