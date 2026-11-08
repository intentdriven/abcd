---
schema_version: 1
id: "iss-2611070830056851"
slug: "with-the-person-away-the-stop-check"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/autonomy/owedread.go"
remedy: "Let the owed read leave out an item another live session holds: a branch checked out in a worktree a live peer session started in, or one a session has marked as handed to a peer, with the reason shown once. Keep the item owed by someone: it stays named to the session that holds it, and returns to the others when that session ends. Cover two sessions on one account in a test: the holder is told, the other is released."
resolution: "The worktree half is built: a branch checked out in a worktree a live peer session started in (its occupancy marker) is left out of the owed read and named once as that session's; it stays owed to the session working there and returns to the others when that session ends. The unbuilt half, a session marking an owed item as handed to a peer, is held by its own open issue, iss-2611080546086771, captured in this commit."
impact: fix
resolved_by:
  commit: "229e491c60743cb3052f27443069e656771cbcdd"
---

With the person away, the stop check names the same owed item to every session on the account, because it reads what the checkout's branches owe and not what this session took on. A pull request a peer session is already working on, in its own worktree, is named to the others as theirs to fix. Told by the coordinator to leave it, a session cannot say so: the item stays named on each stop, and after two wakes a scheduled wake no longer releases the stop, so the only ways out are taking the peer's work or going against the coordinator.

## Grounds

- pursued: TestPeerSessionWorkNamedNotOwed reads the same tree as three sessions: the worktree's session is told of its branch and the other two are released; TestPeerWorkReturnsWhenThePeerEnds covers the return. Shown wrong if the item is named to every session again, or to none; the hand-over marking is iss-2611080546086771's to build.
