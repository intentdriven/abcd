---
schema_version: 1
id: "iss-2609252001486609"
slug: "a-stale-mirror-diffs-against-an-older-release-with-exit-0"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/launch_deep.go"
---

A stale mirror diffs against an older release with exit 0 and no note: in a full (non-shallow) clone whose newest release tag is missing while CHANGELOG.md dates that release — a fork whose tags froze at fork time, a mirror that never fetched the latest tag — unanchoredBaseline (internal/surface/cli/launch_deep.go) returns the newest tag it finds before CHANGELOG.md is read, so launch --dry-run reports parity against v0.9.0 (6 added, 21 changed, 103 unchanged) where v0.10.0 is dated, with no refusal and no note. commands/launch.md promises that a checkout missing the previous release's tag while CHANGELOG.md dates a release is not a first launch and that parity.refused names that release; this checkout is missing exactly that tag and does not refuse. The ship-to-tag window (CHANGELOG.md dates 0.11.0 just cut, tag v0.10.0) carries the same evidence with the opposite meaning, so the refusal there must not call the release just cut the previous release, and --baseline stays the escape.
