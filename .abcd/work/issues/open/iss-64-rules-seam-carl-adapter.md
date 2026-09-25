---
schema_version: 1
id: "iss-64"
slug: "rules-seam-carl-adapter"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "manual-capture"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Plan a rules-backend seam with an opt-in CARL adapter?"
---

Make rule-injection a seam: keep the native Go loader (itd-3) as the floor, add an opt-in 'carl' backend so a user with CARL installed can route to it. Per adr-22 native-default + easy-onboard-a-better-external. Config rules.backend: native | carl.