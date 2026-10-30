---
id: dcn-2610231114308486
slug: the-worktree-store-s-pruneone-its
date: 2026-10-23
---
2026-10-23 — The worktree store's PruneOne, its PreRemove hook and Row's absolute path land in step 2 with no production caller, as the API the sweep (spc-2610230730269295) calls: its removal is the store's prune core called per worktree, with its liveness probe as the last check before each removal, and its rows need each worktree's absolute path. They stay uncalled only until the sweep lands, which the merge order puts after the store's step 2; if the sweep's plan changes so that nothing calls them, they are removed rather than left as scaffolding. Facilitator, on the technical delegation. <!-- record-lint: forward-looking -->
