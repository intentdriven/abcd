---
schema_version: 1
id: "iss-2610100913086397"
slug: "a-validator-that-meets-a-rate-limit-and"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "review of PR #886 (itd-2609201925079472, run-2610100550297632)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/ratelimit.go"
remedy: "let --restart re-tell a kept validator await, or name a remedy the loop accepts, with a test that drives a refused save-aside to a resumed lane"
resolution: "A rate-limit checkpoint whose save aside is refused now marks the kept await in the run state, and implement step --restart <lane> re-tells exactly the marked awaits, a validator's included, from a fresh Since: a validator's partial return is saved aside and its worktree left untouched, and an unmarked validator still out on the round is left alone."
impact: fix
---

A validator that meets a rate limit and whose partial return cannot be saved aside leaves its lane stuck: the record tells the user to run abcd implement step --restart <lane> after the pause, but Restart refuses any lane without an implementer out, so every later step waits on validators that are gone.
