---
schema_version: 1
id: "iss-2610230015137603"
slug: "a-review-the-rp-route-hands-back-to-the"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "confirmation review pair of feat/tier-escalation-round, 2026-10-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/escalate.go"
remedy: "When the reviewer route falls back to the host, record the round as the host's and resolve the role's row tier for it, so a fallen-back review is asked for a tier and escalates on a later failed round like any other"
---

When the rp reviewer route is configured and reachable as a step starts but its call then fails, the review falls back to the host, and the round's record still says the rp reviewer route took it with no tier, so the host runs that review without a tier and a later failed round never escalates it: tier escalation (itd-2609170822093401 criterion 10) does not apply to a review that fell back. The behaviour predates the escalation work; the escalation fix round covered only a route with no URL.
