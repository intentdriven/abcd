---
schema_version: 1
id: "iss-2610110915232822"
slug: "promoterequest-grounds-is-documented-as"
severity: "nitpick"
category: "documentation"
source: "impl-review"
found_during: "review of PR #902 (itd-82 step 1, run-2610101724174688)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/promote.go"
remedy: "rewrite the PromoteRequest.Grounds comment and the requireGrounds mention to say grounds are optional, as optionalGrounds reads them"
---

PromoteRequest.Grounds is documented as REQUIRED and a later comment names requireGrounds, but Promote reads it through optionalGrounds, and the drain's hand-back calls Promote with none. The code is right and the comments are stale, so a reader is told a call the drain makes would be refused.
