---
schema_version: 1
id: "iss-2609261457365969"
slug: "itd-28-ac-1-promises-review-of-commit-written-by-the-tool-no"
severity: "minor"
category: "drift"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: fidelity audit itd-28"
origin: researcher-authored
production_mode: hand-written
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (rulings-owed D3, narrowing a shipped promise; run A 2026-09-28): Should an abcd verb write the dated review folders so the tool itself stamps review_of_commit, or should itd-28 ac-1 be amended to what shipped (the charter template carries the key and gate RD004 refuses a folder without it), ratifying the implementer's Decision 3?"
remedy: "Waits on ruling D3: if a verb: add an abcd verb that creates a dated review folder under .abcd/work/reviews/ and stamps review_of_commit with the full HEAD sha, proven by a test that RD004 passes on its output; if narrowed: record the narrowing in the H10 shape, never rewriting ac-1: this issue resolved by the lane's commit plus an Audit Notes line on itd-28 saying the charter template carries the key and RD004 refuses a folder without it, ratifying the implementer's Decision 3."
---

itd-28 ac-1 promises review_of_commit written by the tool; no abcd path writes a dated review folder, so the pin is a charter-template obligation on the author whose absence RD004 refuses, and the narrowing was settled by the implementer in Decision 3 rather than by the product thinker

## Remedy grounds (2026-09-29)

Confirmed at this base: no code path writes a review folder (internal/core/reviews/reviews.go only reads the pin). Rejected: a hook that stamps the sha after the fact, which pins whatever HEAD is at stamp time rather than the commit reviewed.
