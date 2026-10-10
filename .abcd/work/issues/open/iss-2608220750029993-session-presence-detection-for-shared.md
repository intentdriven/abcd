---
schema_version: 1
id: "iss-2608220750029993"
slug: "session-presence-detection-for-shared"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
related_intents: [itd-2609091034175565, itd-2609091416295622]
remedy: "Waits on ruling A (plan the presence lease inside draft itd-2609150819440345, or as its own intent): either way, build it on the implement run store's lease primitive (exclusive create, expiry, lapse logged) rather than a new format, so each live session holds a lease keyed on its worktree path with session id, branch and start time, and have `.githooks/pre-commit` warn, never refuse, when another unexpired lease names the same worktree; prove it with a hook test in which a second session's live lease prints the warning and an expired one prints nothing."
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25; rulings-owed A): Plan the session-presence lease and its pre-commit warning inside the session-register draft itd-2609150819440345 (M30), or as its own intent?"
---

Session-presence detection for shared checkouts: each live session leases a marker (session id, intended branch, started-at) in the checkout, and the pre-commit gate warns when another live session's lease is present — the armed-detector rung of the AGENTS.md concurrent-session conventions, which are vigilance-only until this ships. Lease home needs thought: the local tier is per-worktree by design, and the hazard is precisely two sessions in ONE worktree, so the lease belongs to the checkout, not the branch.

## Remedy grounds (2026-09-29)

- SOTA check: git-worktree (https://git-scm.com/docs/git-worktree, read 2026-09-29) offers only `git worktree lock`, an administrative marker with a free-text reason and no notion of which process uses a worktree, so git has no presence to reuse; Gray and Cheriton's leases (SOSP 1989, https://doi.org/10.1145/74850.74870) show a time-bounded grant makes a crashed holder cost a delay rather than correctness, the property the implement claim leases already have.
- Rejected: a lock that refuses the commit (a stale marker would block a live session, and the AGENTS.md convention asks for announcement) and a marker in the per-worktree local tier, which the record itself excludes.
