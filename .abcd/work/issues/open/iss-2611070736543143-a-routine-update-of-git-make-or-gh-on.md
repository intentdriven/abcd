---
schema_version: 1
id: "iss-2611070736543143"
slug: "a-routine-update-of-git-make-or-gh-on"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "follow-up list from a release cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/runner/wall.go"
remedy: "Treat a change to those tools' binaries the way a harness update is treated: verify the new binary and re-pin it, and carry on; tell the person in one line that it happened."
---

A routine update of git, make or gh on the machine ends a running drive with 'binary changed while the drive ran'. The behaviour is documented and sits outside the re-prove, but a drive meant to run for weeks with no stops cannot survive an ordinary package update of a tool it merely calls.

2026-11-07: met at 14:25 in the autonomous run of 7 Nov: a lane working this issue stopped at the check, as it should, and the fix is queued for a ruling.
