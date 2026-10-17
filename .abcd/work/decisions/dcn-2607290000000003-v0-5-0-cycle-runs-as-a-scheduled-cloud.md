---
id: dcn-2607290000000003
slug: v0-5-0-cycle-runs-as-a-scheduled-cloud
date: 2026-07-29
---
2026-07-29 — v0.5.0 cycle runs as a scheduled cloud loop (one item per 5-hour round, plan order): auto-merge authorised through the strict gate only (CI green + two adversarial MERGE verdicts; security lens mandatory on workstream A/B diffs); itd-74 is in autonomous scope, multi-round via PR resume, bounded by spc-20 and STOP condition 1. Rejected: human-merges-everything (stalls the ordered pipeline at merge cadence) and auto-merge-C/D-only (slows exactly the security half the release is for).
