---
schema_version: 1
id: "iss-2609211105023379"
slug: "n-lanes-in-flight-are-n-serial-recalibrations-of-one-file-that-the-merge-queue-cannot-merge"
severity: "major"
category: "process"
source: "agent-finding"
found_during: "pilot run 2026-09-20"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/config/reading-presets.json; evals/coldreading_window_test.go"
---

N lanes in flight that grow the reading corpus are N serial recalibrations of one file, .abcd/config/reading-presets.json, because every lane's recalibration rewrites the same four fields per position and conflicts with every other lane's, and the merge queue's update-branch cannot merge them. So two such PRs cannot be in the queue together: the second goes BEHIND when the first merges and is stuck on a conflict the forge cannot resolve. The pilot run sequenced its lanes by hand around this: lane C recalibrated three times (at its own tip, after PR 648, after PR 649) and lane D twice, at fifteen minutes and sixty to eighty thousand tokens each, and the last lane closed a full session later than its code was ready. Sibling of iss-2609210748032488 (the recalibration is a hand-run recipe): that record wants the verb; this one records that even with the verb the recalibration must happen once, on the integration tip, not once per lane — the calibration belongs to the merge, as a queue-side step or a single post-merge commit, not to the branch. For the big run's file: lanes that touch internal/core/intent, internal/core/lint, internal/core/capture or their surface pages cannot be queued together.
