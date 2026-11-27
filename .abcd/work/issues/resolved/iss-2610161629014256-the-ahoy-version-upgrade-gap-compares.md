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
resolution: "Already fixed by 006217b6a: upgradeFrom (detect.go) raises version.upgrade only when both sides parse as release tags and launch.CoreGreater says the running one is newer, and stampedVersion (apply.go) keeps a recorded release the running version is not at least; TestDetectVersionIsDirectional and TestVersionStampNeverRecordsAnOlderVersion cover both directions. No plain-string version compare remains among the sibling sites."
impact: fix
shipped_in: v0.16.0
resolved_by:
  commit: "006217b6afd26ef0b93dd336243a5eab66bb6fa7"
---

The ahoy version.upgrade gap compares the setup version and the running binary as plain strings, so a plugin older than the repository's recorded setup version is reported as an upgrade, and running setup would then record the older version.
