---
schema_version: 1
id: "iss-2610170814081317"
slug: "with-stacked-branches-abcd-land"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/landing/announce.go"
remedy: "Before announcing, skip an item whose record is already in resolved/ or wontfix/ on origin's default branch, or whose intent is in shipped/."
---

With stacked branches, abcd land announces items that are already merged: after the parent branch lands by a squash or rebase merge, its original commits are not on origin's default branch, so land reads their Refs:, Resolves: and Delivers: trailers again when the child branch lands and posts merging for them, for example merging iss-1 for an issue already merged and resolved.
