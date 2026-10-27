---
schema_version: 1
id: "iss-2610272352153282"
slug: "abcd-peers-names-one-sibling-worktree"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "review pair of fix/peers-cap-warning after its rebase, 2026-10-27"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/copykey.go"
remedy: "Make siblingCopies the one worktree listing: return each sibling's key with both its path and its short name, and have siblingCopy look the key up in that map, so every surface names a sibling from one reading and only the rendering (path or name) differs"
---

abcd peers names one sibling worktree two ways in a single listing: a reservation row and the ceiling show its folder name (siblingCopies, a capped worktree listing excluding this copy by key), while a release hold or refusal from the same worktree shows its full home-redacted path (siblingCopy, an uncapped streamed listing excluding this copy by path), and a copied worktree reads as 'lane-c or lane-d' on one side and whichever path was found first on the other
