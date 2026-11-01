---
schema_version: 1
id: "iss-2611011332417169"
slug: "under-away-mode-the-stop-hook-can-name-a"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "away mode, 2026-10-31 to 2026-11-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/autonomy/wake.go"
remedy: "Clamp the deadline wakeDeadline returns to no earlier than now plus a minimum lead (the host's cron lateness, a few minutes), and drop a WakeBy or NextRenewal already in the past from the minimum, so the named deadline is always one a wake scheduled now can meet; proven by a test where a renewal instant before now yields a deadline after now."
---

Under away mode the stop hook can name a wake deadline that is already past, and then no scheduled wake can release the stop: wakeDeadline (internal/core/autonomy/wake.go) takes the earlier of now plus the owed ceiling and the away record's WakeBy or a reservation's NextRenewal less the lease skew, so a renewal instant fixed before now yields a fixed past deadline. On 2026-10-31 it named 18:28 GMT from 18:2x onward and kept naming it after 18:28, and on 2026-11-01 it did the same at 00:23 and 05:21; a cron set for the next minute was not counted, and the stop then fell to the unchanged-set row.
