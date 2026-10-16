---
schema_version: 1
id: "iss-2610161639025702"
slug: "preflight-throughput-sets-the-pace-of"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "product thinker, 2026-10-16 evening plan"
origin: researcher-authored
production_mode: hand-written
found_at: "Makefile"
remedy: "First research the state of the art, with primary sources: merge-queue batching (GitHub merge queue groups, bors, Zuul), test impact analysis and selective testing (go list -deps per changed package, Bazel and Pants affected targets), build and test result caching, sharding, and splitting local preflight from CI. Then adopt what fits: (1) a bundled rule that a feature's reviewed step branches are merged into one integration branch and preflighted once, and that records-only branches are stacked on one branch; (2) a preflight whose test stages run only the packages a change can affect, plus the race lane's affected subset, with a full run kept for the release cut and CI; (3) records-only changes skip the Go test stages, under a check that proves no code changed. Pin each with a test and measure queue time before and after."
---

Preflight throughput sets the pace of all work on this machine, and nothing keeps it in bounds. On 2026-10-16 the load gate admitted one make preflight at a time, each took about 40 minutes, and about 20 waiters were queued across seven sessions. Every reviewed branch had its own preflight, so a seven-step feature needed seven full runs, and records-only branches paid the same full cost as code. Sessions waited hours, and the queue's ordering after the reboot starved late waiters (fixed separately on fix/load-queue-start-order). No rule tells sessions to batch work, and no part of preflight scales its cost to what a change touches.
