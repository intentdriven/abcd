---
schema_version: 1
id: "iss-2610100846469473"
slug: "testrebuildinginplaceisbyteidentical-internal-core-site"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "the records branch preflight after v0.13.4, 2026-10-10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/site"
remedy: "Find the WaitDelay the root lookup's git call sets and make a timeout under load distinguishable from a real failure to name the root: retry the lookup once, or report the timeout as its own error rather than as 'git cannot name its root'; reproduce with the test under a CPU-bound load before and after."
---

TestRebuildingInPlaceIsByteIdentical (internal/core/site) failed in make preflight on 2026-10-10 with 'sits inside a checkout but git cannot name its root (exec: WaitDelay expired before I/O complete (stderr: ""))' while the machine's five-minute load average was about 48 on 16 cores, from concurrent sessions' preflights; the same test passed three times in a row once the load fell. The site build's root lookup turns a git subprocess that misses its WaitDelay into a refusal, so a loaded machine fails a records-only branch's preflight on code it did not touch.
