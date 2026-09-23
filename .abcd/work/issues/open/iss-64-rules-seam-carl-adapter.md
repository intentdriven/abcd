---
schema_version: 1
id: "iss-64"
slug: "rules-seam-carl-adapter"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "manual-capture"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (planning owed: rules-backend seam with an opt-in CARL adapter (adr-22 pattern) has no planned intent). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

Make rule-injection a seam: keep the native Go loader (itd-3) as the floor, add an opt-in 'carl' backend so a user with CARL installed can route to it. Per adr-22 native-default + easy-onboard-a-better-external. Config rules.backend: native | carl.