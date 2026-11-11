---
schema_version: 1
id: "iss-2611080546086771"
slug: "with-the-person-away-a-session-the"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "run-2611080159059570 lane-1 round-2 fix"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/autonomy"
remedy: "Give a session a way to mark an owed item as handed to a live peer session, and have the owed read leave an item so marked out of that session's owed work, naming the peer once. Keep the item owed by someone: it returns to the marking session when the peer ends. Cover two sessions on one account in a test: the marking session is released, and the item is named to it again once the peer ends."
---

With the person away, a session the coordinator tells to leave an owed item to a peer has no way to record that it handed the item over, so the stop check keeps naming the item to it. The worktree case is already covered: a branch checked out in a worktree a live peer started in reads as that peer's. Uncovered is a peer that works from the shared copy, or holds no worktree, record or lane for the item: nothing the session can write marks the item as handed over, so it is named on every stop until the wake cap no longer releases the stop. This is the half of iss-2611070830056851 that run-2611080159059570 did not build.
