---
schema_version: 1
id: "iss-2611101316053555"
slug: "with-the-person-away-the-stop-check"
severity: "major"
category: "bug"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/autonomy/judge.go"
remedy: "Count an owed item as waiting, not stuck, while a live load-gate waiter, preflight, race run or lane agent of this session works on its head, and let a scheduled wake release the stop for it however many wakes have passed; a test holds a preflight running for the head over three wakes and finds the stop released each time."
resolution: "With the person away, an owed set counts as waiting while every capped item in it has a live full preflight of this account, queued or started at that item's head from a worktree of this checkout, and no receipt names that head: a qualifying wake releases the stop however many wakes have passed, for at most 18 hours from the first check that found every capped item so covered, counted afresh when the capped set changes. One capped item without such a run keeps the whole set on the two-wake rule. Built for the preflight's whole run only: a lane agent, a race or narrow test run and a host's background agent are not counted (left to iss-2611110707295084)."
impact: fix
resolved_by:
  commit: "64ddd11f5"
---

With the person away, the stop check treats an owed item as stuck when its next step is a test or an agent that is still running. After two wakes with the owed set unchanged it stops letting a scheduled wake release the stop (row 4 of the plan in internal/core/autonomy/judge.go), although the item can only move once a preflight, a load-gate wait or a lane agent finishes. On 7 Nov at 08:05 six paid turns in a row each said only that the session was still waiting while a 45-minute preflight ran for the item named; the same was seen on 7 Nov at 20:05 and 21:35 with a branch waiting at the load gate for its receipt, on 9 Nov at 22:44 with a push waiting on its preflight, and on 10 Nov at 04:58 with a background lane agent working the branch's fix round. Sessions held turns open with foreground waits to get past it.
