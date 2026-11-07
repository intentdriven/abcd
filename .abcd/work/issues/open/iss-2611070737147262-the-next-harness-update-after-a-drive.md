---
schema_version: 1
id: "iss-2611070737147262"
slug: "the-next-harness-update-after-a-drive"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "follow-up list from a release cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/runner/wall.go"
remedy: "Carry the name across the re-execution and test the message after one."
---

The next harness update after a drive re-executes itself still says 'an unnamed version' instead of naming the release.
