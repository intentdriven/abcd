---
schema_version: 1
id: "iss-2610272041031418"
slug: "a-context-bound-git-read-can-fail-on-a"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gitutil/repo.go"
remedy: "Read a context-bound git's output through cmd.StdoutPipe and cmd.StderrPipe to EOF before calling Wait, so exec's own copy goroutines, which WaitDelay times, are not in the path; keep WaitDelay as the guard against a killed git's child holding a pipe after the context ends. Add a test that runs a context-bound git under an artificially slow reader and asserts a successful git is never reported as ErrWaitDelay."
---

A context-bound git read can fail on a loaded machine after git has already answered: isolatedGitContext (internal/gitutil/repo.go) sets WaitDelay to 50 ms whenever the context can end, and exec starts that timer when git exits on its own, so the copy of git's output can miss it and Wait reports exec.ErrWaitDelay over a successful git. The status reader runs under a deadline, so on 2026-10-27 TestADoubtfulAnnouncementIsFlaggedAndEveryOtherBranchStillRead (internal/core/claim/status_test.go:300) failed a records-only preflight with 'reading the status branches: exec: WaitDelay expired before I/O complete', and in use the prompt hook would print 'status unavailable' for the same reason. The fix for iss-2610100846469473 removed the delay only for callers with no deadline.
