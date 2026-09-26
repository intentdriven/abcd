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
---

Every commit in this checkout now builds ./cmd/abcd twice, once in .githooks/pre-commit for the sources refresh and once in commit-msg for the outbound lint (about 13 s on a cold build cache), where one shared build per commit would do; and in a linked worktree the pre-commit prints a sources skip line on every commit beside the existing linked-worktree notice (review2-sources 4 and 7).
