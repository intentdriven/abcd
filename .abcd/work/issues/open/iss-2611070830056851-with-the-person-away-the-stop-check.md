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
---

With the person away, the stop check names the same owed item to every session on the account, because it reads what the checkout's branches owe and not what this session took on. A pull request a peer session is already working on, in its own worktree, is named to the others as theirs to fix. Told by the coordinator to leave it, a session cannot say so: the item stays named on each stop, and after two wakes a scheduled wake no longer releases the stop, so the only ways out are taking the peer's work or going against the coordinator.
