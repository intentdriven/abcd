---
id: dcn-2610271946266365
slug: the-worktree-store-s-notes-entry-api
date: 2026-10-27
---
2026-10-27 — The worktree store's notes-entry API (NewNotesEntry, NotesEntries, WriteNew, ReadFile, Flush), built ahead of its one named caller, the sweep's patch salvage in spc-2610230730269295, leaves the store stack and is kept on the local branch park/store-notes-entry until that caller is built, decided by the technical facilitator's session abcd-e8 on the product thinker's standing delegation of 2026-10-27. Grounds: AGENTS.md's wired-or-it-isn't-done rule refuses an exported API no production code calls, the review of 2026-10-27 found none on the store or the sweep branch, and dcn-2610231114308486 covers only PruneOne, PreRemove and Row.Path. The step that builds the salvage cherry-picks it back with its caller, and fixes the kind-suffix collision the review found (a worktree named x-salvage lists as x's salvage entry). <!-- record-lint: forward-looking -->
