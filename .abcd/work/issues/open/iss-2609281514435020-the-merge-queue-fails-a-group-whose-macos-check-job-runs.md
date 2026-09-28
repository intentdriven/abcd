---
schema_version: 1
id: "iss-2609281514435020"
slug: "the-merge-queue-fails-a-group-whose-macos-check-job-runs"
severity: "minor"
category: "process"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: ruling Z"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/ci.yml"
---

The merge queue fails a group whose macOS check job runs past 30 minutes, cancelling pull requests whose code is fine: #728 was cancelled at 30.3 minutes, #730 twice at the cap, and #733 once. The check job's timeout-minutes and the main ruleset's merge-queue check_response_timeout_minutes are both 30, while the macOS leg of a source change measures 23 to 30 minutes, so ordinary runner variance cancels a passing run, and nothing warns before the cancellation arrives. Ruling Z (2026-09-28, the technical facilitator) raises both limits to 45 minutes and asks for a warning once the macOS check passes 35 minutes, so a speed lane opens before any cancellation.
