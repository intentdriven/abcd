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
found_at: "internal/core/autonomy/owedread.go"
remedy: "Have the owed-work read treat a branch that is checked out in another worktree of the store as that worktree's: name it to a session standing elsewhere as information, never as work it owes, whether or not a session key is recorded for the branch. Keep a branch with no session key owed only to sessions standing in the worktree that has it checked out. Grounds: the concurrency rule that a diff a session did not make is a peer's work, and the two earlier fixes of the same reader, iss-2611020719483054 and iss-2610311406481214, which this case falls outside."
---

While the away mode holds, the stop check hands a branch checked out in one session's worktree to every other session as work it owes. On 2026-11-21, within minutes of the away mode being set, two sessions standing in other checkouts (the main checkout and a sibling worktree) each had a turn end answered with one owed item: a branch another session had just made and committed to in its own worktree of the store, not yet pushed, with the instruction to do the local work and then push. Both recognised it as a peer's and left it. A session that obeyed would have pushed a peer's commits before their checks had run. The branch had been made with git branch at a commit and then checked out through the worktree store's own add verb with its existing-branch option, from a session started in the main checkout. Two earlier fixes scoped this reader to the stopping session (iss-2611020719483054, iss-2610311406481214) and kept a branch with no session key owed; this case is a branch owed to everyone.
