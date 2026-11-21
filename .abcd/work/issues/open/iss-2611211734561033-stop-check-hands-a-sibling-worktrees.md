---
schema_version: 1
id: "iss-2611211734561033"
slug: "stop-check-hands-a-sibling-worktrees"
severity: "major"
category: "bug"
source: "agent-observation"
found_during: "the away mode set on 2026-11-21 during lab-261121170045-71e07e7"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/guard_branches.go"
remedy: "Find why the created-branch record took this name with no session key although a host session made it, and record the key at that point, so that a keyless entry arises only from a record an older version wrote. Keep the reader's rules as they are: a branch the stopping session created, or a lane of a run it drives, stays owed to it, and its own branch in a worktree it made stays in its sight. For an entry that still has no key, word the stop's text so that a session which did not make the branch is told to leave it, not to do the local work and push. Establish separately why a branch made through the store's add verb was not owed to the session that made it. Grounds: the concurrency rule that a diff a session did not make is a peer's work, and the two earlier fixes of the same reader, iss-2611020719483054 and iss-2610311406481214."
---

While the away mode holds, the stop check handed a branch one session had just made to every other session as work it owes. On 2026-11-21, within minutes of the away mode being set, two sessions each had a turn end answered with one owed item: a branch a third session had made minutes earlier and committed to in its own worktree of the store, not yet pushed, with the instruction to do the local work and then push. Both recognised it as a peer's and left it. A session that obeyed would have pushed a peer's commits before their checks had run. All three sessions had been started in the main checkout. The branch had been made with git branch at a commit, in a shell line that first changed into a store worktree, and then checked out through the worktree store's add verb with its existing-branch option. The created-branch record of the start copy held the name with no session key, as it holds every one of its 31 entries, the newest among them. The reader's rules are deliberate and stay: a branch another worktree has checked out is dropped unless this copy's sessions recorded creating it or made a worktree for it, a branch the stopping session created is owed to it, and a branch with no session key stays owed (iss-2611020719483054, iss-2610311406481214). So the keyless entry was owed to every session whose stop check reads that copy. The same evening the stop check told the session that had made the other branch of the same work, through the store's add verb with a new branch, that nothing was owed while that branch held five commits not pushed: that branch's name is not in the record at all.
