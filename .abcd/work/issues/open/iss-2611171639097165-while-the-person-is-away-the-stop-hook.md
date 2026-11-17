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
found_at: "internal/core/mode"
remedy: "have the stop hook's owed-work check drop an item whose step is in the away queue (match the queue entry to the branch or record it names), or let abcd mode queue take the item's key so the hook reads it as handed over, and report the queued items once at the turn's end instead of holding the stop"
---

While the person is away, the stop hook keeps naming a branch whose only remaining step is the person's own (deleting or pushing a branch the session did not create) after that step has been queued with abcd mode queue, so the session cannot end a turn without a wake every few minutes for as long as the person is away; on 17 Nov it fired well over a hundred times for one queued deletion
