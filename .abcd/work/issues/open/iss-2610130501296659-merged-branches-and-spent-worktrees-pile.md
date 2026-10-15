---
schema_version: 1
id: "iss-2610130501296659"
slug: "merged-branches-and-spent-worktrees-pile"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "session close-down after the v0.14.0 cut, 2026-10-13"
origin: researcher-authored
production_mode: hand-written
remedy: "A drain, a release cut and a session close each end with a residue sweep: delete local and remote branches whose tip origin/main contains (or another kept branch contains), remove their clean worktrees after archiving each .abcd/.work.local under ~/.abcd.noindex/archive/<root-sha>/, never touch an unmerged, dirty or handover-named branch, and list what it could not prove safe; build it as the mechanism of itd-118 and the reclaim half of itd-2609091014076309"
found_at: "commands/drain.md"
related_intents: [itd-118, itd-2609091014076309]
---

Merged branches and spent worktrees pile up because nothing removes them: by the close of 2026-10-12 the checkout held 137 local branches (83 already merged into main), 94 worktrees and 4 merged remote branches, and a person had to ask for the cleanup by hand. The person asked (2026-10-13) that a future drain and similar runs do this automatically. Done by hand that day with a read-only inventory (merged into origin/main, contained in a kept branch, PR state, worktree clean, local-tier file count) and a dry run before acting: 92 local branches, 40 worktrees and 4 remote branches removed, every worktree's local tier archived first. Draft itd-118 (merged work leaves no residue) and planned itd-2609091014076309 (the worktree store and its reclaim) describe the want; neither runs at the end of a drain or a cut.
