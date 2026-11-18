---
schema_version: 1
id: "iss-2611171639097165"
slug: "while-the-person-is-away-the-stop-hook"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "autonomous run while the person was away"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/autonomy/owedread.go"
remedy: "Widen the stop check's hand-back (owedReader.handBack and settleHandBacks) so that, while the person is away, an item whose only remaining step is the person's own and which a queued step names by its exact branch is handed back whatever the hold-pushes setting and whatever the item's kind; it stays owed to the person, is reported once at the turn's end, and counts as done for no other check."
---

While the person is away, the stop hook keeps naming an owed branch whose only remaining step is the person's own after that step has been queued with `abcd mode queue`. A deletion of a branch the session did not create is one such step, and a push the session may not send is another. The session cannot move the item, so every turn ends on the same demand to schedule a look a few minutes ahead, for as long as the person is away.

The stop check already hands back a queued item (`handBack` in `internal/core/autonomy/owedread.go`, accepted by `settleHandBacks` in `judge.go`), but only for an unpushed branch, only when the away mode holds pushes, and only when a preflight receipt names the head and the head contains the base tip. A run whose standing instruction allows pushes does not hold them, and an item with no upstream is not of the unpushed kind, so neither queued item of this run was handed back.

The match to the queue entry stays on the whole branch name (`namesBranch`), as it is today, so a queued step never silences an item it does not name. Two neighbouring records address other causes of the same repeated stop: iss-2611152247568320 (a look-before time already past, and items another session holds), and iss-2611150851542274 (a lane a live drive holds, resolved).
