---
schema_version: 1
id: "iss-220"
slug: "deterministic-ci-failures-lack-compose-time-prevention"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "manual-capture"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (planning owed: compose/verify PR surfaces for managed repos and an allowlisted self-repair class for deterministic check failures). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

simple deterministic CI failures have no compose-time prevention or self-repair path: the attribution gate detects a missing PR-body Assisted-by trailer post-hoc (PR 230), but nothing composes or verifies the body at creation time and no loop is authorised to apply the mechanical fix. PR-body composition is facilitator-owned manual work and therefore automation backlog: abcd should compose/verify the PR surfaces it requires for managed repos, and an allowlisted class of deterministic check failures (trailer append, formatting) should be self-repairing