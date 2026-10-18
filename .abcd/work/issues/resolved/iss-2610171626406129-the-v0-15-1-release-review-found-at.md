---
schema_version: 1
id: "iss-2610171626406129"
slug: "the-v0-15-1-release-review-found-at"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/peers.md"
remedy: "Rewrite the passage at commands/peers.md:72-108 to match the shipped behaviour the review names, and check its sibling chapters for the same claim."
resolution: "The page now states what the shipped code does, re-verified against the code and the verb's --help at v0.15.1 before the edit."
impact: internal
resolved_by:
  commit: "a394eb10a"
---

The v0.15.1 release review found at commands/peers.md:72-108: INCOMPLETE. The payload list never names `release_holds`. The verb emits it (internal/surface/cli/peers.go:48, with the holdRow fields release, subject, holder, copy, effective_expiry, unreadable, own, unsighted and text at :99-109). `abcd peers --help` says it lists 'a line for each release being cut (a live rel-vX.Y.Z reservation, which holds every other session's merge until its tag is on origin)'. The changelog says '`abcd peers` shows the hold' and '`--json` release holds carry `own` and `unsighted`'. An agent following this page is never told to r…
