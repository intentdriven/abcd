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
remedy: "Make a git call that misses its WaitDelay under load distinguishable from a real answer in both lookups the site build makes: the root lookup, which reports the timeout as 'git cannot name its root', and gitutil.RootCommit (internal/gitutil/repo.go:281), which folds every rev-list error into an empty root commit; retry once or return the timeout as its own error, and reproduce with the test under a CPU-bound load before and after."
---

TestRebuildingInPlaceIsByteIdentical (internal/core/site) failed in make preflight on 2026-10-10 with 'sits inside a checkout but git cannot name its root (exec: WaitDelay expired before I/O complete (stderr: ""))' while the machine's five-minute load average was about 48 on 16 cores, from concurrent sessions' preflights; the same test passed three times in a row once the load fell. The site build's root lookup turns a git subprocess that misses its WaitDelay into a refusal, so a loaded machine fails a records-only branch's preflight on code it did not touch.

A second symptom, reported by a peer session at about 08:40Z on 2026-10-10: the same test failed in the race lane of another branch's preflight with 'build: site: <tmp>/003 is not empty, and this repository has no root commit for a .abcd-site-build to name ... refusing to remove it'. The test's first build succeeded and its second got an empty root commit. gitutil.RootCommit (internal/gitutil/repo.go:281) folds every rev-list error into "", so the same WaitDelay miss under load reads as a repository with no root commit. Five isolated race-enabled reruns passed.
