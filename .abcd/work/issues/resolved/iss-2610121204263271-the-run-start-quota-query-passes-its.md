---
schema_version: 1
id: "iss-2610121204263271"
slug: "the-run-start-quota-query-passes-its"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "security review of fix/loop-quota-and-ratelimit (PR #979), 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/runner/runner.go"
remedy: "have quotaOf hold the caller's deadline itself: run the reporter in a goroutine and return ctx's error when ctx ends first, with a test using a reporter that ignores its context"
resolution: "quotaOf runs the reporter in a goroutine and returns the caller's deadline error when the context ends first, so a reporter that ignores its context can no longer hold the start of a build or drain"
impact: fix
---

The run-start quota query passes its deadline to the runner's QuotaReporter and then waits on the reporter: runner.quotaOf (internal/core/runner/runner.go) calls qr.Quota(ctx) and returns whatever it returns, so a reporter that ignores its context holds the start of abcd build and abcd drain for as long as it chooses, the deadline notwithstanding. No shipped runner implements QuotaReporter today, so nothing reaches it yet; the next runner that reports quota over I/O inherits it. The route is abcd's: the wait is the loop's own.

## Grounds

- pursued: a reporter that never returns leaves quotaOf returning context.DeadlineExceeded at the caller's deadline; a quotaOf call that outlives its deadline shows it wrong

## Route

abcd adds this route: the wait on the reporter is the loop's own, at the start of `abcd build` and `abcd drain`. Nothing outside abcd waits on it, so the fix is abcd's and changes nothing about how a repository works.
