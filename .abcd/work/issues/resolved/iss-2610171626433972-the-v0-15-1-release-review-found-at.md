---
schema_version: 1
id: "iss-2610171626433972"
slug: "the-v0-15-1-release-review-found-at"
severity: "nitpick"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/status.md"
remedy: "Rewrite the passage at commands/status.md:53-54 to match the shipped behaviour the review names, and check its sibling chapters for the same claim."
resolution: "The page now states what the shipped code does, re-verified against the code and the verb's --help at v0.15.1 before the edit."
impact: internal
resolved_by:
  commit: "a394eb10a"
---

The v0.15.1 release review found at commands/status.md:53-54: Change narration on a user-facing page: '`branch` is the full branch name, `abcd-status/maya` (it was `maya` before step 3's fix rounds)'. It narrates an internal build step of an unreleased spec; `abcd status` is new in this release, so no user ever saw `maya`. It breaks the present-tense rule, and docs-currency-lint misses it because the page is under commands/. Fix: delete the parenthesis.
