---
schema_version: 1
id: "iss-2610222125175640"
slug: "after-macos-process-ids-wrap-the-load"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "lane D, reported by abcd-ae from the live process table"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/machineload/queue.go"
remedy: "Order both the queue and the open-moment stand-back by process start time read from the process table, earliest first, with a stable tie-break inside the start tolerance (pid only within a tie), and count a waiter as ahead by start alone; handle waiters from binaries that still wait on every lower pid without reintroducing the deadlock iss-2610161634043273 described; sweep every other site that orders or ranks processes by pid."
resolution: "The load gate's queue counts every queued waiter that started earlier as ahead, whatever its pid, with starts within machineload.StartTolerance a tie broken by pid inside one chained group, all derived from one table's ages. A waiter ahead by start with a higher pid is crossed: an abcd from v0.15.0 or earlier waits on every lower pid and so may wait on this one, and every wait cycle passes through a crossed wait, so a wait stops counting crossed waiters once they alone have held its gate shut for implement.WaitCrossedCap (five minutes) on end; WaitBackoffCap bounds only the open look and could not break that cycle. The open look stands an opener back from another that opened in the same moment unless it precedes it by start and by pid alike (machineload.StandBackFrom): the later start stands back from the earlier whatever the pids, and the earlier from a lower pid too, which v0.15.1 and earlier run past. The LOAD rule, the help, the command page, the brief and the managed block say so."
impact: fix
---

After macOS process ids wrap, the load gate's queue is no longer first-come-first-served and an old waiter can be starved by every new starter. On 2026-10-22 waiters started 19 s and 9 min earlier held pids 8034 and 12051 while hour-old waiters held 58628 and 73003. The fix for iss-2610161634043273 orders the queue by start time but counts a queued waiter as ahead only when it is ahead by start AND has the lower pid (internal/core/machineload/queue.go, queueOrder/queuedAhead), so after a wrap neither waits on the other; the open-moment stand-back in implement.WaitLoad (openLook, internal/core/implement/load_wait.go ~546) then yields to the lower pid, so the hour-old waiter stands back for each young low-pid starter until its backoffs run out or its timeout (8 h) ends, and a release-blocking preflight can be held all night.

## Grounds

- pursued: an earlier-started waiter is admitted before any later one whatever their pids, and a mutual wait with a pid-ordered waiter ends within WaitCrossedCap; a waiter held behind a later start, or a cycle of waits lasting past the cap while nothing else shuts the gate, would show it wrong
