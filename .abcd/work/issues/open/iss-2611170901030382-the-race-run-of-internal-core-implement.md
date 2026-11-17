---
schema_version: 1
id: "iss-2611170901030382"
slug: "the-race-run-of-internal-core-implement"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "v0.17.0 release cut, macOS gate"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop"
remedy: "Run the package with go test -json on a macOS runner at 73fd41ad and at 04b5bd00, compare each test's time, and speed up or split the tests whose time grew, then bring the shard's ceiling back down."
---

The race run of internal/core/implement/loop on the macOS runner grew from 50 and 68 minutes on the nightly runs of 15 and 16 Nov 2026 to past 80 on both of the v0.17.0 cut's runs of 17 Nov, while the same package took 1976 seconds on the ubuntu runner, against 2235 before, and about 1830 on a local machine, so something in it is slow on the macOS runner alone; the ceiling was raised for the cut (dcn-2611170900489974), which hides the growth rather than explaining it.
