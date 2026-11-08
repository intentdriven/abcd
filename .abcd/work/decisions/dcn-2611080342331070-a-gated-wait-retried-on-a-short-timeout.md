---
id: dcn-2611080342331070
slug: a-gated-wait-retried-on-a-short-timeout
date: 2026-11-08
---
2026-11-08 — A gated wait retried on a short timeout keeps its place: the load gate carries a waiter's stand-aside and queue budget across retries, keyed on the command, the lane and the run id so no other lane can take it over, and the carried budget expires with the waiter's own timeout. Grounds: a drive's harness ends each command after about ten minutes, so a lane's gated test must wait in short runs; today each retry starts at the back of the queue and never reaches the narrow gate's five-minute stand-aside cap, so a lane can starve behind whole runs. Carrying the budget is the smallest trust change: the alternative of letting the guard's approver pass a background flag through touches a trust boundary, and having the drive gate a lane's scoped tests itself is too large for now. With it, AGENTS.md's two-minute drive-lane form stands. Reverse by having the drive gate a lane's scoped tests itself, at which point the carried budget and the short-retry form can go.
