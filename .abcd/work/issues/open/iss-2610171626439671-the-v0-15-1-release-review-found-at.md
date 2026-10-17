---
schema_version: 1
id: "iss-2610171626439671"
slug: "the-v0-15-1-release-review-found-at"
severity: "nitpick"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/launch.md"
remedy: "Rewrite the passage at commands/launch.md:115-116 to match the shipped behaviour the review names, and check its sibling chapters for the same claim."
---

The v0.15.1 release review found at commands/launch.md:115-116: The page is addressed to the person cutting the release and says '`/abcd:peers` shows "release vX.Y.Z being cut by @you: merges are held"'. Run from the cutting copy, peers renders Hold.Line(), which is 'you are cutting release vX.Y.Z from this copy: every other session's merges are held' (internal/core/claim/release.go:122-126; peers.go:470). The quoted line is what other copies see. commands/reserve.md:91-95 states both correctly. Fix: say that other sessions see the quoted line and this copy sees the 'you are cutting' line.
