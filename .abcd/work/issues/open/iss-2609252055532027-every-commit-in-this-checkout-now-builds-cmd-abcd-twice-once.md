---
schema_version: 1
id: "iss-2609252055532027"
slug: "every-commit-in-this-checkout-now-builds-cmd-abcd-twice-once"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: ".githooks/pre-commit"
deferred_after: "v0.11.1"
deferral_reason: "measured at BASE, the second build is a cache hit (the pre-commit and commit-msg builds use the same flags; a warm rebuild of ./cmd/abcd took 0.55 s here), so sharing one binary across the two hooks would save about half a second at the cost of a trust hand-off between them. What remains is the linked-worktree skip line printed on every commit, where the choice is between keeping it, silencing it when the primary checkout's store is inherited, or refreshing that store from the worktree: a ruling on hook output owed to the product thinker (drain lane drainRest, run A, 2026-09-29)."
---

Every commit in this checkout now builds ./cmd/abcd twice, once in .githooks/pre-commit for the sources refresh and once in commit-msg for the outbound lint (about 13 s on a cold build cache), where one shared build per commit would do; and in a linked worktree the pre-commit prints a sources skip line on every commit beside the existing linked-worktree notice (review2-sources 4 and 7).
