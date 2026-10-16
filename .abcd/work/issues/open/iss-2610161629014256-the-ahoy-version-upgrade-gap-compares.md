---
schema_version: 1
id: "iss-2610161629014256"
slug: "the-ahoy-version-upgrade-gap-compares"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "planning prep for the after-update drafts"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/detect.go"
remedy: "Compare as semantic versions: report an older binary as a downgrade (or not at all) and never let setup record a version older than the one already recorded; cover both directions with a test."
---

The ahoy version.upgrade gap compares the setup version and the running binary as plain strings, so a plugin older than the repository's recorded setup version is reported as an upgrade, and running setup would then record the older version.
