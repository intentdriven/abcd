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
resolution: "A git with no deadline (the site build's root lookup and gitutil.RootCommit, which the lane worktree store also keys on) no longer runs under the 50ms WaitDelay that a loaded machine's slow output copy missed, so it waits for git's answer. Under a deadline, a WaitDelay miss is retried once and a second miss is returned as gitutil.ErrGitTimedOut, which RootCommitContext passes on rather than reading as no root commit. Proven by fault injection, not by load: a fake git that answers and leaves a process holding its output pipe reproduced both refusals word for word before the fix, and the build and the lookups answer correctly after it."
impact: fix
resolved_by:
  commit: "f1703a29f"
---

TestRebuildingInPlaceIsByteIdentical (internal/core/site) failed in make preflight on 2026-10-10 with 'sits inside a checkout but git cannot name its root (exec: WaitDelay expired before I/O complete (stderr: ""))' while the machine's five-minute load average was about 48 on 16 cores, from concurrent sessions' preflights; the same test passed three times in a row once the load fell. The site build's root lookup turns a git subprocess that misses its WaitDelay into a refusal, so a loaded machine fails a records-only branch's preflight on code it did not touch.

A second symptom, reported by a peer session at about 08:40Z on 2026-10-10: the same test failed in the race lane of another branch's preflight with 'build: site: <tmp>/003 is not empty, and this repository has no root commit for a .abcd-site-build to name ... refusing to remove it'. The test's first build succeeded and its second got an empty root commit. gitutil.RootCommit (internal/gitutil/repo.go:281) folds every rev-list error into "", so the same WaitDelay miss under load reads as a repository with no root commit. Five isolated race-enabled reruns passed.

A third sighting, reported by a peer session on 2026-10-10: TestASiteWithNoDocsBuildLinksNoDocsTree (internal/core/site) failed on another branch's preflight with the same 'sits inside a checkout but git cannot name its root (exec: WaitDelay expired before I/O complete (stderr: ""))' refusal. It is the root lookup again, reached through a different test: any site build on a loaded machine can hit it.

A fourth sighting, reported by a peer session on 2026-10-10 under a load average of about 169 on 16 cores: TestALaneThatExhaustsItsFixRoundsIsHandedBack (internal/core/implement/loop) failed with 'refused at validate for lane-1: the repository has no root commit to key the worktree store on'. The lane worktree store keys on gitutil.RootCommit (internal/core/implement/loop/lane.go:112), so the same transient git failure reads there as a repository with no commits.

A possible fifth sighting, reported by a peer session on 2026-10-10 at a load average of about 125: TestALaterRefusalInTheCallKeepsTheTimeTheCheckWaitBegan (internal/core/implement/loop) failed with 'lane 1 waits for its full check: <nil>', and three isolated race-enabled reruns passed. It is consistent with this cause but not proven to be it. The test steps the run up to ten times and stops at the first refusal of any kind. The land and validate stages derive the lane's worktree through gitutil.RootCommit (internal/core/implement/loop/lane.go:112), so an empty root commit there refuses before lane 1's landing is written, and that leaves exactly the nil landing the message shows. Any other refusal in those calls would leave the same trace.
