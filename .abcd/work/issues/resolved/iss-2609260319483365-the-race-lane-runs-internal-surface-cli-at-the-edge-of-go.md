---
schema_version: 1
id: "iss-2609260319483365"
slug: "the-race-lane-runs-internal-surface-cli-at-the-edge-of-go"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/ci.yml"
resolution: "The race lane runs under a declared go test -race -timeout 20m in ci.yml's check job, release.yml's verify job (and its scaffold template) and make preflight, which lifts go test's 10m per-package default: a slow package no longer fails at ten minutes, the failure that ejected PR #717. On merge-group runs the merge queue's 30-minute check response cap (the main ruleset, mirrored in .abcd/work/rulesets/main-protection.json) is the outer ceiling, so ci.yml's check job stays at 30 minutes, equal to that cap; on the macOS leg the slowest package starts about 18 minutes in, so a genuine hang there is still cancelled at 30 without a goroutine dump. Raising the queue's cap is an admin act on the live ruleset, left to the technical facilitator. release.yml's verify job is not a merge-queue job; its ceiling rises from 15 to 35 minutes so the package timeout fires there before the job is cancelled. TestRaceLaneBudgetIsDeclaredAndFitsItsJob holds the three commands to one explicit timeout, the check job's timeout-minutes at or below the mirror's merge-queue cap, and verify's timeout-minutes to the package timeout plus its measured headroom (the check-job half corrected in 2f7b903b)."
impact: internal
resolved_by:
  commit: "f0994f70"
---

The race lane runs internal/surface/cli at the edge of go test's default 10-minute per-package timeout on the macOS runner, so a slow runner ejects an unrelated pull request from the merge queue. PR #717 was ejected by merge_group run 36212607621 (job check (macos-latest), step Test (race, internal)): panic: test timed out after 10m0s and FAIL github.com/intentdriven/abcd/internal/surface/cli 600.120s, with no test hung; the package's total wall time crossed the default. The same package under -race on macOS took 417.9s (run 36211489533), 472.0s (36211390371) and 567.9s (36209973364), and 324-355s on ubuntu. ci.yml's race step, release.yml's verify race step and the Makefile preflight all run go test -race ./internal/... with no -timeout, so the ceiling is the implicit default rather than a budget sized from measurement. The job ceilings around the step are as tight: the macOS check job took 19.0, 21.4, 25.9 and (failing) 28.1 minutes against timeout-minutes: 30, so lifting the package timeout alone moves the failure to the job cap, where it arrives with no goroutine dump; and release.yml's verify job, ubuntu only with an uncached toolchain, took 13.2 minutes of its 15 on 2026-09-24 (run 35963282477, cli race 302s), since when the ubuntu race step has grown from 9.8 to about 12 minutes, so the next release's verify job is at risk of the same cancellation.

## Grounds

- pursued: the macOS race step stops failing when internal/surface/cli's total -race wall time crosses 600s, while a genuinely hung test still fails with a goroutine dump at 20m in release.yml's verify job; shown wrong by a merge-queue ejection from a package-timeout panic under 20m on a passing package, by a verify job cancelled at its ceiling before the race step's own timeout could fire, or by a check job set above the merge queue's response cap. A merge-group check cancelled at the 30-minute cap does not show it wrong: that ceiling is the queue's, and the record says so
