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
resolution: "Removed both reviewerVerdicts role checks around the route's digest compare: in verifyValidation the implementer and auditor roles return before the compare and parseVerdict accepts only a reviewer's verdict words, and in receipt validatorRoute returns a route only for a validator whose Route the verifier recorded, which is a reviewer's alone."
impact: internal
resolved_by:
  commit: "843a445f3"
---

The role gates on the reviewer route's digest compare, in receipt (loop.go) and in verifyValidation (validate.go), are redundant and unreachable: no validator but a ruthless or security reviewer reaches the compare, so removing both leaves every test green.

## Grounds

- pursued: every validator outcome stays as before, so the loop's existing outcome tests stay green; a non-reviewer return that gained a route record would show the change wrong
