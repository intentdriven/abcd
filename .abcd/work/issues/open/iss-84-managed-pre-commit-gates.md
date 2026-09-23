---
schema_version: 1
id: "iss-84"
slug: "managed-pre-commit-gates"
severity: "minor"
category: "future-work-seed"
source: "agent-finding"
found_during: "2026-07-10 prepare-this-repo skill grilling"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (planning owed: core-owned managed pre-commit gates (relates draft itd-62) has no planned intent). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

Managed pre-commit gates: the interim prepare-this-repo skill offers the secrets and absolute-path pre-commit config from a private template directory outside the repo. The gate belongs to the configuration layer itself: a core-owned surface should install and maintain commit gates in managed repos, replacing the template copy. Seeded from the scaffold-repo retirement analysis; relates to the pluggable safety gate draft (itd-62).