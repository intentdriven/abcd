---
schema_version: 1
id: "iss-2610230014564359"
slug: "the-worktree-store-s-spec"
severity: "minor"
category: "architectural-insight"
source: "impl-review"
found_during: "worktree-store build run-2610222231427062, step 1 review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/embargo/storepath/storepath.go"
related_intents: [itd-2609091014076309]
remedy: "fold into the spec's steps: step 2's ValidName refuses the reserved names the store's siblings use (embargo, read from storepath.Name, not copied), and steps 3 to 5 list the embargo store as abcd's own and never judge, archive or prune it, each with a test"
resolution: "Delivered by spc-2609301811532881: ValidName refuses the embargo store's name, read from storepath.Name and compared without case (internal/core/ahoy/worktree/worktree.go:194, :212; TestAddRefusesBeforeWritingAnything, TestValidNameIsOnePlainPathSegment); list renders the embargo store as a sibling with its reason in place of the state columns, never as a lane or a candidate (list.go:238-239, :262, :518; TestListShowsTheEmbargoStoreAsAbcdsOwn); prune keeps it on that reason before any judgement, archive or removal (prune.go:284; TestPruneNeverTouchesTheEmbargoStore)."
impact: internal
resolved_by:
  spec: "spc-2609301811532881"
---

The worktree store's spec (spc-2609301811532881) never mentions the embargo store, which lives inside the same lane as ~/.abcd.noindex/worktrees/<root-sha>/embargo/ (internal/core/embargo/storepath, Name = "embargo"): a later step's add could take the name embargo, and list, prune and quiet-candidate scans could judge or reclaim the embargo store's detached worktree as a stale lane, so an unpublished advisory's private records could be archived or removed; found by the ruthless review of the store's step 1 on 2026-10-23, which judged nothing reaches it yet because lane names cannot be embargo
