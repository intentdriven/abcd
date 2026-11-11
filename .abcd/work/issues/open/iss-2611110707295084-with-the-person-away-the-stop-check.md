---
schema_version: 1
id: "iss-2611110707295084"
slug: "with-the-person-away-the-stop-check"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "review of the stop check's live-work change, 11 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/autonomy/livework.go"
remedy: "Have a build run's lane record its live agent per item in run state the stop check can read, and have the load gate's narrow and race waiters write the same marker the preflight's waiter writes, so each is proven the way the preflight is; count each as waiting under the same 18-hour bound. For a host's background agent abcd did not start, have the session record it on the checklist item it works, and count it only while the host's stop payload lists that agent as in flight, under the same 18-hour bound: the session's own word alone proves nothing."
---

With the person away, the stop check counts an owed item as waiting only while a full preflight queued or started at its head is live. An item whose next step is a lane agent working its branch, a race run or a narrow test run still reads as stuck after two wakes with the owed set unchanged, so a session spends turns saying it is waiting. This refines iss-2611101316053555, which built the preflight case; dcn-2611101818557506 says why a lane agent was left: abcd cannot yet prove from state it owns that such a process works on that item. dcn-2611101920386950 says a command other than the preflight's stages counts for nothing when run through the gate; no decision gives a reason for race and narrow runs, which are not built. A host's background agent that abcd did not start is not counted either: abcd holds no state that names it.
