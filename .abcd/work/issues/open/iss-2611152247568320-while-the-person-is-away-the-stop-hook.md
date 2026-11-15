---
schema_version: 1
id: "iss-2611152247568320"
slug: "while-the-person-is-away-the-stop-hook"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "itd-2610301914350541 step 4 build"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/hook_stop.go"
remedy: "Compute the look-before time from now when the owed item's own deadline has passed, never state a time in the past, and leave out of a session's owed set a branch or pull request another live session or drive holds"
---

While the person is away, the stop hook asks the session to schedule a look before a stated time that has already passed: at 22:46 and again at 22:48 GMT on 2026-11-15 it asked for a cron firing before 22:44 GMT, which no cron can meet, so every turn ends with the same unmeetable demand. In the same session it named a pull request a peer session owns (announce-close-1115) as owed work to merge in, though the session may not move a peer's branch; iss-2611150851542274 covers the same shape for a branch a live drive owns. Later the same session was told to renew a reservation a running drive holds (iss-2611040554354084, held from the main checkout and already resolved on origin), and the stop would not release while it stayed unrenewed.
