---
schema_version: 1
id: "iss-2610230015129841"
slug: "implement-step-s-reviewer-route-hand-off"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "confirmation review pair of feat/tier-escalation-round, 2026-10-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/build.go"
remedy: "Add a CLI test of implement step with oracle.review: rp pointing at a reachable fake, asserting the ruthless and security reviews record the reviewer route with no tier, so passing nil there fails it"
---

No test at the command level covers implement step handing its reviewer route into the routing load: changing loop.LoadTiering(roots, review) to loop.LoadTiering(roots, nil) in internal/surface/cli/build.go leaves every CLI test green, though with oracle.review: rp and a reachable URL every ruthless and security review would then record a tier and its brief ask for one while the rp route takes the review. The logic is covered at step level through Drive; only this one-line hand-off at the front door is not.
