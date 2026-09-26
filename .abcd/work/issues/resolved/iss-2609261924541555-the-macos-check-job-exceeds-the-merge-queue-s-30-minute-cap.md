---
schema_version: 1
id: "iss-2609261924541555"
slug: "the-macos-check-job-exceeds-the-merge-queue-s-30-minute-cap"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: #728 macOS cancel"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/ci.yml"
resolution: "The cli package's race-instrumented time comes down without dropping an assertion: the 9 MiB history capture and the surface-prose check run uninstrumented only (no goroutine of ours in either, asserted on both CI legs by the plain lane), ProseShapeClaims skips a path whose words the prose never spells, and recordid compiles its filename grammar once per family (iss-2609261943168303). Paired on one machine: cli under -race 410.1s to 186.1s, the whole go test -race ./internal/... lane 532.7s to 310.0s; cli plain unchanged within noise (user CPU 63.1s and 62.5s)."
impact: internal
resolved_by:
  commit: "bc435cdb2"
---

The macOS check job exceeds the merge queue's 30-minute cap once the cli lane's worked-example tests land: on PR #728 (integration head d850f06f) run 36262653588 was cancelled at 30.3 minutes while internal/surface/cli was still running under the race lane, where main's macOS runs take 23 to 28.5 minutes. The job's timeout-minutes equals the queue's check_response_timeout, so the queue fails the group rather than waiting. The cli package's test wall time, above all under -race, is the lever: it must come down by at least the three minutes the leg is over without weakening what the tests prove.

## Grounds

- pursued: the macOS check job concludes inside the merge queue's 30-minute cap with cli no longer the race step's tail; a macOS check run on this change reaching 30 minutes, or its race step's cli line within three minutes of the 454s the #728 re-run took, would show it wrong.
