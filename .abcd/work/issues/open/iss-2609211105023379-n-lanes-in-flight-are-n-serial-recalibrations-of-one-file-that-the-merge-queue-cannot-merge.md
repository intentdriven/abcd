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
deferred_after: v0.11.1
deferral_reason: "The product thinker's ruling M33 of 2026-09-23: plan it next cycle with its sibling iss-2609210748032488 (a reading calibrate verb), moving calibration from each lane to one step per merge. Runs recalibrate by hand at the integration tip meanwhile (1e0e6f5b3). Owed: that planning interview, which opens on one question: does the once-per-merge calibration run as a merge-queue job or as a post-merge commit on main?"
remedy: "Waits on the M33 planning interview's first question: if a merge-queue job, lanes stop recalibrating and the merge_group run measures the queue's merged tree, failing only when it breaches the declared window by more than the margin iss-2609210748032488 asks about; if a post-merge step, a workflow on main runs that record's calibrate verb and opens one recalibration pull request authored by the person, as the H2 dependency-update arrangement does, with the window eval on lane branches demoted to a warning. Prove either with two corpus-growing lanes queued together in a rehearsal, both merging."
---

N lanes in flight that grow the reading corpus are N serial recalibrations of one file, .abcd/config/reading-presets.json, because every lane's recalibration rewrites the same four fields per position and conflicts with every other lane's, and the merge queue's update-branch cannot merge them. So two such PRs cannot be in the queue together: the second goes BEHIND when the first merges and is stuck on a conflict the forge cannot resolve. The pilot run sequenced its lanes by hand around this: lane C recalibrated three times (at its own tip, after PR 648, after PR 649) and lane D twice, at fifteen minutes and sixty to eighty thousand tokens each, and the last lane closed a full session later than its code was ready. Sibling of iss-2609210748032488 (the recalibration is a hand-run recipe): that record wants the verb; this one records that even with the verb the recalibration must happen once, on the integration tip, not once per lane — the calibration belongs to the merge, as a queue-side step or a single post-merge commit, not to the branch. For the big run's file: lanes that touch internal/core/intent, internal/core/lint, internal/core/capture or their surface pages cannot be queued together.

## Deferral 2026-09-29

Deferred past v0.11.1: The product thinker's ruling M33 of 2026-09-23: plan it next cycle with its sibling iss-2609210748032488 (a reading calibrate verb), moving calibration from each lane to one step per merge. Runs recalibrate by hand at the integration tip meanwhile (1e0e6f5b3). Owed: that planning interview, which opens on one question: does the once-per-merge calibration run as a merge-queue job or as a post-merge commit on main?

## Remedy grounds (2026-09-29)

- Why: the record places calibration on the merge, not the branch; the two answers differ in who writes the new window, so each is stated with its writer, and neither is picked.
- Sources (consulted 2026-09-29): a merge queue runs required checks on its temporary gh-readonly-queue branch through the merge_group event (https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/configuring-pull-request-merges/managing-a-merge-queue), so a queue job judges a merged tree rather than adding a commit to it.
- Rejected: a bot-authored post-merge commit, which the attribution gate refuses by design.
