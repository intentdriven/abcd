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
---

The race lane runs internal/surface/cli at the edge of go test's default 10-minute per-package timeout on the macOS runner, so a slow runner ejects an unrelated pull request from the merge queue. PR #717 was ejected by merge_group run 36212607621 (job check (macos-latest), step Test (race, internal)): panic: test timed out after 10m0s and FAIL github.com/intentdriven/abcd/internal/surface/cli 600.120s, with no test hung; the package's total wall time crossed the default. The same package under -race on macOS took 417.9s (run 36211489533), 472.0s (36211390371) and 567.9s (36209973364), and 324-355s on ubuntu. ci.yml's race step, release.yml's verify race step and the Makefile preflight all run go test -race ./internal/... with no -timeout, so the ceiling is the implicit default rather than a budget sized from measurement. The job ceilings around the step are as tight: the macOS check job took 19.0, 21.4, 25.9 and (failing) 28.1 minutes against timeout-minutes: 30, so lifting the package timeout alone moves the failure to the job cap, where it arrives with no goroutine dump; and release.yml's verify job, ubuntu only with an uncached toolchain, took 13.2 minutes of its 15 on 2026-09-24 (run 35963282477, cli race 302s), since when the ubuntu race step has grown from 9.8 to about 12 minutes, so the next release's verify job is at risk of the same cancellation.
