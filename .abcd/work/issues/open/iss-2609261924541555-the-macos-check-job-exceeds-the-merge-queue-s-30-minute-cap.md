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
---

The macOS check job exceeds the merge queue's 30-minute cap once the cli lane's worked-example tests land: on PR #728 (integration head d850f06f) run 36262653588 was cancelled at 30.3 minutes while internal/surface/cli was still running under the race lane, where main's macOS runs take 23 to 28.5 minutes. The job's timeout-minutes equals the queue's check_response_timeout, so the queue fails the group rather than waiting. The cli package's test wall time, above all under -race, is the lever: it must come down by at least the three minutes the leg is over without weakening what the tests prove.
