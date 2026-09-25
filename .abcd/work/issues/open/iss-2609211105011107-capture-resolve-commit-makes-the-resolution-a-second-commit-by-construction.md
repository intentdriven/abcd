---
schema_version: 1
id: "iss-2609211105011107"
slug: "capture-resolve-commit-makes-the-resolution-a-second-commit-by-construction"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "pilot run 2026-09-20"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli (capture resolve --commit)"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Resolve placeholder filled post-commit, or the documented two-commit protocol?"
---

capture resolve --commit wants the sha of the commit that fixed the record, so the resolution is a second commit by construction: the lane commits the fix, runs capture resolve --commit <that sha>, and commits the ledger move with the Resolves: trailer on the second commit. Every one of the pilot run's four lanes produced this pair, and AGENTS.md's rule that a fix resolves its record in the same change is met only by the pair, never by one commit. Wanted: either resolve accepts a placeholder the post-commit stamp fills (the reachability gates RS002/RS003 already verify the sha after the fact), or the surface page states the two-commit shape as the documented protocol so an implementer stops trying to make it one.
