---
schema_version: 1
id: "iss-2611070737163284"
slug: "the-drive-test-worlds-home-directory"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "follow-up list from a release cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/drain_drive_test.go"
remedy: "Name the world's home by a fixed or sanitised label, not the subtest name."
---

The drive test worlds' home directory carries the subtest's name, so a comma in a later subtest name silently changes what it exercises; one subtest of the panic-before-first-move test passed that way.
