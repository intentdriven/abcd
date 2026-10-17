---
schema_version: 1
id: "iss-2610171626419054"
slug: "the-v0-15-1-release-review-found-at"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/launch.md"
remedy: "Rewrite the passage at commands/launch.md:112-114,122-124; to match the shipped behaviour the review names, and check its sibling chapters for the same claim."
---

The v0.15.1 release review found at commands/launch.md:112-114,122-124; commands/reserve.md:82-85: INCOMPLETE. Both pages describe the release hold as binding the implement loop's land step only. launch.md says 'every other session's land step (an `implement step` arming a merge ...) refuses to arm' and 'The hold binds abcd's own land step'. reserve.md says 'an `implement step` at the land fence in any other copy ... exits 3'. In the code, `abcd land` is held too: internal/core/landing/releasehold.go:11-29 calls claim.FetchMergeHold. commands/land.md documents 'While a release is being cut, exit 3', and the changelog line names both 'the implement lo…
