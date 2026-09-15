---
schema_version: 1
id: "iss-2609012202237613"
slug: "the-strict-up-to-date-policy-keeps-auto-merge-prs-out-of-the-merge-queue-that-already-gates-them"
severity: "major"
category: "process"
source: "user-observation"
found_during: "pr-queue-observation-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/work/rulesets/main-protection.json"
resolution: "Strict stays; iss-172's first rung ships as scripts/pr-keep-current.sh, which merges the base into every armed pull request that is BEHIND and not queued, once or on a watch loop, and CONTRIBUTING points at it. The queue-only alternative was declined by the maintainer on 2026-09-01 (DECISIONS.md)."
impact: additive
resolved_by:
  commit: "06574914"
---

Every auto-merge pull request opened tonight (#571, #572, #574) sat BEHIND with queue entry null until a hand-run update-branch, while #570 and #573 entered the queue only because they happened to be up to date at the moment they were armed. Cause: main-protection combines the merge queue (grouping ALLGREEN, merge method MERGE, mirrored 2026-08-19) with strict_required_status_checks_policy true. Auto-merge enqueues a pull request only when its merge state is CLEAN, and under the strict policy a pull request behind main is never CLEAN, so it cannot enter the queue whose whole purpose is to test it combined with the current base. Every merge from the queue moves main and knocks the next un-queued pull request BEHIND again, so a batch of parallel PRs needs one manual update per merge. iss-172 records the invariant that strict is never relaxed because it is what gates the merged result against a duplicate record id minted by two checkouts; that reasoning predates the queue, and the queue now gates exactly that merged result (the group commit on top of main runs every required check before anything lands), so strict adds no gate the queue does not already hold and costs the queue its entry. itd-115 promises the queue removes this toil. Directions, for the maintainer to rule: (1) set strict false in the live ruleset and the mirror, keeping the queue as the gate, and record the ruling against iss-172's invariant; (2) keep strict and automate iss-172 rung 1, an update-branch loop after arming auto-merge, which is what tonight's session runs by hand.

## Grounds

- pursued: we expect armed pull requests to enter the queue without hand updates once the script runs after arming and after each merge; a pull request still sitting BEHIND for a full cycle with the watch loop running would show the rung is insufficient
