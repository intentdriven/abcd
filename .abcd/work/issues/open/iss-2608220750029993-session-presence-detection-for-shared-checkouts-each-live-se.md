---
schema_version: 1
id: "iss-2608220750029993"
slug: "session-presence-detection-for-shared-checkouts-each-live-se"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
related_intents: [itd-2609091034175565, itd-2609091416295622]
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Plan the session-presence lease and its pre-commit warning inside the session-register draft itd-2609150819440345 (M30), or as its own intent?"
---

Session-presence detection for shared checkouts: each live session leases a marker (session id, intended branch, started-at) in the checkout, and the pre-commit gate warns when another live session's lease is present — the armed-detector rung of the AGENTS.md concurrent-session conventions, which are vigilance-only until this ships. Lease home needs thought: the local tier is per-worktree by design, and the hazard is precisely two sessions in ONE worktree, so the lease belongs to the checkout, not the branch.