---
id: dcn-2610180905283806
slug: the-overnight-stall-fix-ships-first
date: 2026-10-18
---
2026-10-18 — The overnight-stall fix ships first, ahead of the coordination intent (facilitator-on-delegation, itd-2610170634394993 planning prep): the fix for the critical overnight stall (an autonomous mode abcd holds, and a stop check that refuses to end a turn with owed work) sits in the checklist draft, which owns a session's modes, waits and stop check; it needs nothing on the shared remote, so it ships as the bundle's first slice, without waiting on this intent, and the security and bug-fix cut waits on that slice alone; this intent then adds what the stall's record asks for across sessions, the idle time a coordinator sees, through the central view's derived state and last-seen time in abcd status rather than in abcd peers as the record's remedy words it, per the decision naming abcd status as the central view (grounds: iss-2610170619247185 remedy; itd-2610170638184486 scope split; the draft's decisions 3 and 6)
