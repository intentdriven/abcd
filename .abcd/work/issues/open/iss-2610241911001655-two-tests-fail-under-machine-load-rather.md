---
schema_version: 1
id: "iss-2610241911001655"
slug: "two-tests-fail-under-machine-load-rather"
severity: "major"
category: "observation"
source: "user-observation"
found_during: "autonomous run 2026-10-24, gated preflight of chore/capture-holdups-3fee"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/autonomy"
remedy: "Drive both tests from an injected clock or a deadline the test controls rather than wall-clock time, so a slow machine cannot change their outcome."
---

Two tests fail under machine load rather than on a defect: TestReadOwedHonoursDeadline in internal/core/autonomy found the read marked complete when the forge timed out, and TestHookSessionEndRetriesATransientRootFailure in internal/surface/cli ran out of retry time and staged nothing, both during a gated preflight at load 17 on ten cores
