---
schema_version: 1
id: "iss-84"
slug: "managed-pre-commit-gates"
severity: "minor"
category: "future-work-seed"
source: "agent-finding"
found_during: "2026-07-10 prepare-this-repo skill grilling"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Plan core-owned managed pre-commit gates, with draft itd-62?"
---

Managed pre-commit gates: the interim prepare-this-repo skill offers the secrets and absolute-path pre-commit config from a private template directory outside the repo. The gate belongs to the configuration layer itself: a core-owned surface should install and maintain commit gates in managed repos, replacing the template copy. Seeded from the scaffold-repo retirement analysis; relates to the pluggable safety gate draft (itd-62).