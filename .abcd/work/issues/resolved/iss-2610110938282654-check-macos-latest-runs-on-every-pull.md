---
schema_version: 1
id: "iss-2610110938282654"
slug: "check-macos-latest-runs-on-every-pull"
severity: "major"
category: "tech-debt"
source: "agent-finding"
found_during: "CI-cycles research, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/ci.yml"
remedy: "run check (macos-latest) only on merge_group and a nightly schedule; on pull_request keep the check name reporting success on an ubuntu runner with its steps gated off"
resolution: "check (macos-latest) runs its suite on every event but pull_request; on a pull request runs-on sends the leg to an ubuntu runner and every step but a notice is gated off, so the required context reports success without a macOS runner or a suite run. The merge-queue entry runs the macOS suite on the tree that lands. TestMacOSLegRunsItsSuiteOnlyWhereItLands evaluates every step's condition and the runner expression per leg, event and classifier verdict."
impact: fix
---

check (macos-latest) runs on every pull request as well as on every merge-queue entry, and macOS is the account's scarcest runner, 5 jobs at once at a median 40 minutes each (8 queue runs, 2026-10-11). Between 03:36 and 08:40 on 2026-10-11 nothing merged: entries waiting for a macOS slot passed the queue's timeout and were removed. Every push already passed make preflight on a Mac, so the pull-request macOS run repeats a local run, and only the queue run tests the merged tree.
