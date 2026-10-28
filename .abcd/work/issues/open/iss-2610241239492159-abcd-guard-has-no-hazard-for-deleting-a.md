---
schema_version: 1
id: "iss-2610241239492159"
slug: "abcd-guard-has-no-hazard-for-deleting-a"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "branch sweep survey, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard"
remedy: "add guard hazards for remote branch deletion (push --delete and push :ref), git branch -D and git worktree remove (with or without --force) that block in hook mode unless the target is provably merged into origin/main (ancestry, patch-equivalence or forge merge state), has no open PR, is checked out in no worktree, is the cwd of no live process and holds no live reservation; the block names abcd sweep as the safe successor, and a person's explicit override is recorded"
---

abcd guard has no hazard for deleting a branch or a worktree: git push origin --delete, git branch -D and git worktree remove are all allowed whatever the target holds, so the safety of a cleanup rests on the agent remembering its own checks. On 2026-10-24 a hand-run sweep's check-then-act loop invoked abcd guard check with the wrong syntax, read the nonzero exit as an objection, printed it and deleted seven remote branches anyway; the deletes were safe only because the survey before them was right, and the hook would have allowed them regardless. A guard has to be enforced at the hook, not hoped for in a script.
