---
id: dcn-2610180720548066
slug: a-waiting-session-wakes-before-half-its
date: 2026-10-18
---
2026-10-18 — A waiting session wakes before half its lease (facilitator-on-delegation, itd-2610170634394993 planning prep): since no process stays alive and a branch is rebuilt only when its agent asks to land, a session waiting on a dependency wakes, through the checklist draft's timed wake, at an interval shorter than half its reservations' lease, so an idle waiter's reservations are renewed rather than lapsing after two hours and being taken over (grounds: itd-2609091034175565 two-hour lease, renewed at half; itd-2610170638184486 timed wake; the draft's take-over criterion)
