---
id: dcn-2611262057069959
slug: while-the-person-is-away-a-stop-inside-a
date: 2026-11-26
---
2026-11-26 — While the person is away, a stop inside a session's conversation window with work owed is allowed only while the session holds a qualifying wake, the one the away stop rule already asks for (a cron that fires before the away deadline, the host's lateness included); with nothing owed it is allowed as before, and with work owed and no wake it is held as row 5, naming the window. Grounds: the person is talking to that session, so it may wait for their reply, but a stop with work owed and nothing scheduled leaves no event to run the check again, so the session would stay idle for the rest of the away period once the window ended (iss-2611260602137306). A wake that fires inside the window is enough too: its stop is judged again, under the window or past it. Reverse by returning the allowed verdict before the wake check in the conversing branch of autonomy.Judge.
