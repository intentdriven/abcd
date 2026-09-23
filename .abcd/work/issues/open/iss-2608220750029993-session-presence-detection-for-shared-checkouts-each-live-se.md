---
schema_version: 1
id: "iss-2608220750029993"
slug: "session-presence-detection-for-shared-checkouts-each-live-se"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
related_intents: [itd-2609091034175565, itd-2609091416295622]
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (planning owed: session-presence lease and pre-commit warning; the lease home needs design (itd-2609221656373558 covers run claims, not presence)). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

Session-presence detection for shared checkouts: each live session leases a marker (session id, intended branch, started-at) in the checkout, and the pre-commit gate warns when another live session's lease is present — the armed-detector rung of the AGENTS.md concurrent-session conventions, which are vigilance-only until this ships. Lease home needs thought: the local tier is per-worktree by design, and the hazard is precisely two sessions in ONE worktree, so the lease belongs to the checkout, not the branch.