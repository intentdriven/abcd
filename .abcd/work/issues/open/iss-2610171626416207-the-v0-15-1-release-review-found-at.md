---
schema_version: 1
id: "iss-2610171626416207"
slug: "the-v0-15-1-release-review-found-at"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/peers.md"
remedy: "Rewrite the passage at commands/peers.md:78-80 to match the shipped behaviour the review names, and check its sibling chapters for the same claim."
---

The v0.15.1 release review found at commands/peers.md:78-80: STALE. The page says `released_by` is '`tombstone`, or `default-branch` when the record is already resolved...'. The code also emits `tag` (internal/core/claim/release.go:46, ReleasedByTag) for a release reservation whose tag vX.Y.Z is on origin, and commands/reserve.md documents it as '`released_by: tag`'. Fix: add `tag` (a release whose tag is on origin) to the list.
