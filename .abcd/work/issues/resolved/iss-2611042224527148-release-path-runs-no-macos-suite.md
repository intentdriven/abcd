---
schema_version: 1
id: "iss-2611042224527148"
slug: "release-path-runs-no-macos-suite"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "review of moving the macOS suite out of the merge queue"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/release.yml"
remedy: "Add the macOS shard jobs to release.yml's path before anything is tagged or published, required by the publishing jobs, so that a shard with no runner or a failing test stops the release until re-run; a test holds the publish jobs' needs to them."
resolution: "release.yml runs macos-suite.yml's three shards on the commit it is about to tag, and tag, release and site all need the macos-gate job over them; TestEveryPublishingJobNeedsTheMacOSGate holds that"
impact: internal
---

The release path runs no macOS suite, though the product thinker ruled that each release cut runs its own before releasing, whatever the nightly said. With macOS out of the merge queue, a macOS-only regression merged after a nightly run could be tagged and published on Linux checks alone, since every job in release.yml and auto-release.yml runs on ubuntu and a red nightly stops nothing.
