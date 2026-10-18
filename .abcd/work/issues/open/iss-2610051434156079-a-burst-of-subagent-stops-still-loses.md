---
schema_version: 1
id: "iss-2610051434156079"
slug: "a-burst-of-subagent-stops-still-loses"
severity: "major"
category: "bug"
source: "drift-detection"
found_during: "v0.13.1 release PR #834 merge-queue CI, macOS race lane, 2026-10-05"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/history/staging.go"
remedy: "Make a stage that cannot take the shared lock within its bound wait longer or queue the transcript for the next drain instead of dropping it, and have the hook say on stderr when a transcript was not staged; prove it with the 12-way burst test under -race on a deliberately slowed lock."
deferred_after: v0.13.1
deferral_reason: "Second deferral, made by the 2026-10-07 autonomous run cutting v0.13.2: that patch ships the hook-message fix and the change-guard hardening and touches no staging code. The loss has appeared only on a slow, race-instrumented macOS runner, so its cause is not yet established, and the remedy needs the staging lock reworked and proved under a deliberately slowed lock. That is an investigation of its own, owed before the next minor release; a third deferral should go to the product thinker."
---

On the v0.13.1 release PR's merge-queue run, the macOS race lane failed TestHookSubagentStopConcurrentCompletionsAllStage: 12 simultaneous subagent-stop hooks all exited 0 but only 10 transcripts were staged (internal/surface/cli/hook_subagent_stop_test.go:336). The same suite passed on the PR's own run and on every queue run earlier that day, so the loss appears only under a slow, race-instrumented host, but it is a silent loss: the hook reports success and the transcript never reaches the store. It is the defect iss-2609090828371674 resolved (the shared lock's backoff ceiling capping staging throughput), shown not to hold under load.

## Deferral 2026-10-05

Deferred past v0.13.0: Found during the v0.13.1 cut in code that release does not touch (the staging path shipped before v0.13.0); v0.13.1 is a patch of unrelated fixes, and a correct fix needs the staging lock redesigned and proved under load, which is a change of its own for the next cycle.

## Deferral 2026-10-07

Deferred past v0.13.1: Second deferral, made by the 2026-10-07 autonomous run cutting v0.13.2: that patch ships the hook-message fix and the change-guard hardening and touches no staging code. The loss has appeared only on a slow, race-instrumented macOS runner, so its cause is not yet established, and the remedy needs the staging lock reworked and proved under a deliberately slowed lock. That is an investigation of its own, owed before the next minor release; a third deferral should go to the product thinker.

## Evidence 2026-10-08

Recurred in the merge queue: PR #869 (records only) was dropped at 06:59Z when its merge group's macOS check failed TestHookSubagentStopConcurrentCompletionsAllStage, staging 11 of 12 transcripts from 12 simultaneous completions (run 37736585477). Re-enqueued; it merged. The deferral past v0.13.1 lapsed at v0.13.2, and the v0.13.3 cut's findings gate did not trip on it because it predates that anchor.

## Cause established 2026-10-18

Reproduced deterministically, so the cause no longer rests on the slow macOS runner alone. The staging lock is not it: since iss-2609090828371674 each agent stages behind its own lock, so the twelve completions of the burst never contend for one, and 25 rounds of the burst under -race here staged every transcript. Slowing all seventeen git calls the hook makes (gittest.SlowPipeGit) still stages, only slowly.

What drops a transcript is a git call that fails once rather than slowly. `gitutil.RootCommit` reads any git failure other than a timeout as "no root commit", so `resolveSubagentStore` takes the payload's cwd for a directory that is not a repository and falls back to the session note. A burst's first completions run before any sibling has written that note, so the hook prints "cannot resolve the repository for this sub-agent … staging nothing" on stderr, exits 0, and the transcript is gone. A git shim that fails the root-commit lookup once reproduces it: the hook exits 0 and nothing is staged. A loaded runner produces the same failure when a fork or a git process fails, and the burst test never printed the hooks' stderr, which is why the cause went unseen. The other path to the same silent loss is a stage that still fails, such as a lock held past its timeout, with the store known.
