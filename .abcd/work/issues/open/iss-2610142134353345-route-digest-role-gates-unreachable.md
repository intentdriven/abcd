---
schema_version: 1
id: "iss-2610142134353345"
slug: "route-digest-role-gates-unreachable"
severity: "nitpick"
category: "tech-debt"
source: "review-followup"
found_during: "PR-976-review-round-5"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/loop.go"
remedy: "Remove the reviewerVerdicts role checks around the route's digest compare in receipt (loop.go) and verifyValidation (validate.go), or move the one real gate to where a non-reviewer could reach it, and keep the outcome tests."
---

The role gates on the reviewer route's digest compare, in receipt (loop.go) and in verifyValidation (validate.go), are redundant and unreachable: no validator but a ruthless or security reviewer reaches the compare, so removing both leaves every test green.
