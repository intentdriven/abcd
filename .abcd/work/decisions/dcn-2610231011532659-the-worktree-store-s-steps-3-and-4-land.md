---
id: dcn-2610231011532659
slug: the-worktree-store-s-steps-3-and-4-land
date: 2026-10-23
---
2026-10-23 — The worktree store's steps 3 and 4 land with the merged judgement reading a branch's own commits from its reflog, after four review rounds closed every reflog shape that gives a false merged except one (iss-2610231011465325): a copy of a created branch, force-reset and left until its reflog expires. It is deferred to step 2 rather than chased further, because each closed shape exposed another and the sound fix is a different source, not a better reflog read: step 2's add records the branch's start commit in the lane, and the judgement then reads the branch's own commits from that. The case needs three rare acts together, every commit it holds is already on the default branch, and the store never deletes the branch. Facilitator, on the technical delegation.
