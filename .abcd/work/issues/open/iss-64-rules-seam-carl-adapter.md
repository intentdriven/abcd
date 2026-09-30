---
schema_version: 1
id: "iss-64"
slug: "rules-seam-carl-adapter"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "manual-capture"
related_intents: [itd-2609292109214516]
deferred_after: "v0.11.1"
deferral_reason: "The product thinker's ruling J22 of 2026-09-29: plan the rules-backend seam with an opt-in CARL adapter; the native loader stays the default. Filed as draft itd-2609292109214516. Owed: that draft's planning interview (the draft stays in drafts/ until a person plans it)."
---

Make rule-injection a seam: keep the native Go loader (itd-3) as the floor, add an opt-in 'carl' backend so a user with CARL installed can route to it. Per adr-22 native-default + easy-onboard-a-better-external. Config rules.backend: native | carl.