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
resolution: "A full clone whose newest release tag is core-older than the release CHANGELOG.md dates newest is unanchored: the parity diff refuses naming the dated release, as a tagless clone does, and names both readings — a missing tag (fetch the tags, or --fetch-baseline) or a release cut and not tagged yet (--baseline <tag>); the refusal no longer calls the named release the previous release, and a full clone holding the tag still diffs against it."
impact: fix
resolved_by:
  commit: "72b618c6"
---

A stale mirror diffs against an older release with exit 0 and no note: in a full (non-shallow) clone whose newest release tag is missing while CHANGELOG.md dates that release — a fork whose tags froze at fork time, a mirror that never fetched the latest tag — unanchoredBaseline (internal/surface/cli/launch_deep.go) returns the newest tag it finds before CHANGELOG.md is read, so launch --dry-run reports parity against v0.9.0 (6 added, 21 changed, 103 unchanged) where v0.10.0 is dated, with no refusal and no note. commands/launch.md promises that a checkout missing the previous release's tag while CHANGELOG.md dates a release is not a first launch and that parity.refused names that release; this checkout is missing exactly that tag and does not refuse. The ship-to-tag window (CHANGELOG.md dates 0.11.0 just cut, tag v0.10.0) carries the same evidence with the opposite meaning, so the refusal there must not call the release just cut the previous release, and --baseline stays the escape.

## Grounds

- pursued: a clone of this tree with v0.10.0 deleted refuses parity naming v0.10.0 and --baseline v0.9.0, and with the tag restored diffs against v0.10.0 (3 changed, 127 unchanged); a stale mirror diffing against v0.9.0, a full clone refusing, or the untagged-cut window calling the cut the previous release would show it wrong
