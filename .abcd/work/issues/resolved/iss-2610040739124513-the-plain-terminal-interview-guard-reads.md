---
schema_version: 1
id: "iss-2610040739124513"
slug: "the-plain-terminal-interview-guard-reads"
severity: "major"
category: "security"
source: "user-observation"
found_during: "abcd-e8 final re-check of plain-Terminal step 4"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/interview/treewatch.go"
remedy: "Hash every <common>/worktrees/*/{HEAD,commondir,gitdir,config.worktree,locked} and each entry's presence, so registering a worktree or repointing its commondir or gitdir stops the interview; and read the preflight-receipts/ directory of every worktree git lists (mode, size, content hash), so a receipt written into any worktree's local tier during a turn stops it. Expected: each probe the re-check describes stops the interview with the path named; shown wrong if any of them leaves it running."
resolution: "The guard records every worktrees/ entry and its HEAD, commondir, gitdir, config.worktree and locked, and the push receipts of every worktree git lists; the re-check's probes are tests."
impact: fix
resolved_by:
  commit: "859a3c9e312197d37e21c2f030c34c60fd04a01f"
---

The plain-Terminal interview guard reads no worktree entry of the repository's common git directory except the current one's HEAD and config.worktree, and no other worktree's local tier: a role allowed only Write can register a worktree under .git/worktrees/ whose gitdir names its own turn directory (left out of the reading) and plant a push receipt there, or plant one in a sibling worktree's local tier, and preflight-receipt.sh check accepts it so the next push of that commit passes the pre-push gate; it can also repoint a sibling worktree's commondir at a fake common directory whose configuration carries an alias, which runs on the person's next git command there. Neither stops the interview.

## Grounds

- pursued: registering a worktree at the turn directory with a receipt there, a receipt in a sibling worktree's local tier, and a repointed commondir or gitdir each stop the interview with the path named (tests TestAWorktreeRouteToAPushOrAHookStopsTheInterview and the CLI cases); shown wrong if any of them lets the interview run on
