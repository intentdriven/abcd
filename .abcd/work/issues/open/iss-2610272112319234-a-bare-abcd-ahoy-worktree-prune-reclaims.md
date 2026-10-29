---
schema_version: 1
id: "iss-2610272112319234"
slug: "a-bare-abcd-ahoy-worktree-prune-reclaims"
severity: "minor"
category: "ux"
source: "impl-review"
found_during: "store stack review, lane E salvage"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/worktree/prune.go"
remedy: "Name a merged worktree's ignored paths outside the tier before reclaiming it: either keep it until named with --yes, or list them in the dry run and the row; the product thinker picks, since keeping would hold every worktree with build output."
---

A bare abcd ahoy worktree prune reclaims a merged, clean worktree without saying that the files git ignores outside the local tier go with it: the tier is archived, but an ignored .env.local, local database or nested checkout is deleted unarchived and the row reads reclaimed. Only a quiet candidate's dossier names them. This matches git worktree remove's own judgement, which the spec adopts on purpose.
