---
schema_version: 1
id: "iss-282"
slug: "the-retention-prune-has-no-executor-computeretention-renders"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "manual-capture"
found_at: "internal/core/launch/ship.go"
deferred_after: "v0.11.1"
deferral_reason: "planning owed to the product thinker (rulings F and J): the prune still has no executor. ComputeRetention is previewed by launch --dry-run only, ship.go names the prune a later phase, and release.yml carries no prune step. The 2026-08-07 disposition recorded in iss-194 keeps v0.4.0 and v0.4.1 published as the fixture for the shipped mechanism, which rules out a manual first rung; what remains owed is planning itd-70, still an unfleshed draft. (re-checked at e792a2314 by lane drainDQ3, run A, 2026-09-29)"
---

The retention prune has no executor: ComputeRetention renders the newest-per-line plan on every cut (brief section 3) and ship.go names GitHub Release + retention prune as a later phase, so superseded patches (v0.5.0, v0.4.0 today) stay published until someone deletes them by hand from the plan. First rung: a documented manual prune after each publish (gh release delete, tags kept for record); the executor belongs in the release workflow after that proves out