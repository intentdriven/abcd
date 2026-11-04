---
schema_version: 1
id: "iss-2611020016159754"
slug: "testanautoplansteprefusesasharedmainchec"
severity: "major"
category: "bug"
source: "review-followup"
found_during: "merge-queue run 41358066383 for PR 1337, 2026-11-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/build_autoplan_surface_test.go"
remedy: "Find why the auto-plan step did not back off in the test's shared main checkout on macOS under -race (the live-session count the refusal reads, or the timing of the fixture's sessions), make the test's fixture deterministic about the sessions it stands up, and run it repeatedly under -race on macOS before calling it fixed."
deferred_after: "v0.16.0"
deferral_reason: "Deferred by the product thinker on 2026-11-03 for this cut, v0.16.1 as the records derive it: the flake's cause is not yet known, and chasing it would hold the cut on a fix, its review and a queue run."
---

TestAnAutoPlanStepRefusesASharedMainCheckoutBeforeItPlans failed in the merge queue's check (macos-latest) job (run 41358066383, the race stage over the whole module) on a merge group whose change touched records only: build_autoplan_surface_test.go:117 'the plan step backs off in a shared main checkout: exit 0'. The step it tests is meant to refuse in a main checkout that two or more live sessions started in, and it exited 0 instead. The same test passed elsewhere, so it is flaky on macOS under the race detector or depends on timing, and each failure ejects the queue's entries: PR 1337 was removed from the queue at 23:52 and waited to be queued again.
