---
schema_version: 1
id: "iss-2610161634043273"
slug: "the-load-gate-s-queue-orders-waiters-by"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "peer report from abcd-a9 on 2026-10-16, after a reboot"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/machineload/queue.go"
remedy: "Order queued waiters by when each started, read from the process table with the session register's start tolerance; break a tie within it by pid, and put a waiter whose start time cannot be read behind every waiter whose start time is known."
resolution: "Queued waiters are ordered by when they started, read from one process table's ages; starts within machineload.StartTolerance (the session register's tolerance, now shared) are a tie broken by pid, waiters linked by ties form one pid-ordered group, and a waiter whose start time cannot be read queues behind every known start. A waiter counts another as ahead only when it is ahead by that order and has the lower pid too (dcn-2610162239308903): start order alone deadlocked against waiters built before it, which wait on every lower pid, and with the pid check every wait points down the pids, so no cycle can form whatever mix of code runs. After a wrap an early high-pid waiter is held only behind earlier starts; two whose start and pid disagree wait on neither and are kept apart by the settle re-check and, within one account, the hold. The LOAD rule, the help, the command page and the brief say so."
impact: fix
resolved_by:
  commit: "3335db8298f65e5505fc78726e3d212d84e043ff"
---

The load gate's queue orders waiters by pid, so a waiter with a high pid starves once the machine's process ids wrap. machineload.QueueAhead counts another waiter as ahead when its pid is lower than this one's. After a reboot on 2026-10-16 the ids wrapped: waiters started between 16:07 and 17:26, with pids 50460 to 71218, all queued ahead of one started at 16:04 with pid 96274. That waiter could only wait out its timeout while lower pids kept arriving. The docs and the LOAD rule promise one waiter at a time in queue order.

## Grounds

- pursued: a waiter that started earlier is admitted before any later one whatever their pids, so a wrap of the process ids no longer starves it; a waiter queued behind a later start with a known start time, or a three-waiter cycle at one gate, would show it wrong
