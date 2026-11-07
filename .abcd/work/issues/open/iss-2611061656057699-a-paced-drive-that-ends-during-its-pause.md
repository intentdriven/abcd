---
schema_version: 1
id: "iss-2611061656057699"
slug: "a-paced-drive-that-ends-during-its-pause"
severity: "nitpick"
category: "bug"
source: "impl-review"
found_during: "review of the drive's profile move"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/terminal.go"
remedy: "Restore the per-lane check beside heldToProfile, so the line is named only for a lane a later drive can take up."
---

A paced drive that ends during its pause, after a move returned another profile's parked lane to in progress, prints a resume line that is then refused: the hold to a profile is read once for the run, not per lane.
