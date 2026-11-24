---
schema_version: 1
id: "iss-2611241052289841"
slug: "finished-worktrees-and-branches-pile-up"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "worktree cleanup after the v0.17.1 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy"
remedy: "When abcd sees a pull request it armed merge (at land, or when peers re-checks it), it copies the worktree's receipts and logs to the main checkout's local tier, removes the worktree and deletes its branch, keeping a bundle so the removal can be undone. The guard treats a worktree whose branch the default branch carries (merged, or its tree carried) as provably kept when what remains untracked is only abcd's own local-tier records and gitignored build output. The same check runs at session start for older leftovers."
---

Finished worktrees and branches pile up until someone removes them by hand. ahoy worktree prune reclaims only merged, clean lanes when it is run, and nothing runs it. A lane whose work reached main squashed or superseded, or whose local tier holds logs, scratch notes, reservation markers or a built binary, is refused by the guard's delete check. On 24 November six superseded worktrees needed their local tiers emptied by hand, and then a person's own command, before they could go. Each one left behind stays in the stop check, the status sweeps and the lane list.

## Also seen in another managed repository (2026-11-24)

A session working another abcd-managed repository landed three pull requests with `abcd land`, merged them, and published and deployed their releases. Eight worktrees and their local branches stayed behind. The guard refused each removal as unproven, because every worktree that has been through `abcd land` holds untracked files: the review receipts under `.abcd/.work.local/` (21 in one case) and, for a release worktree, the built `dist/`. Only the owner's forced removal or a guard allow could clear them. The owner asked why the session was asking at all, when the pull requests had merged, and why the clean-up had not happened on its own once the merge succeeded. With the worktrees gone, `git branch -d` on the same branches went through without trouble.

So the gap is product-wide, not this repository's alone. Two shapes of the fix are open: abcd removes the worktree itself once it sees its armed pull request merge (at `land`, or when `peers` re-checks a merge it armed), knowing which local-tier files are its own disposable receipts; or the guard treats a worktree whose branch is merged into the default branch as provably kept when what remains untracked is only abcd's own local-tier records and gitignored build output. Either way, anything worth keeping is copied to the main checkout's local tier by abcd first, not by the agent.
