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
---

The load gate's queue orders waiters by pid, so a waiter with a high pid starves once the machine's process ids wrap. machineload.QueueAhead counts another waiter as ahead when its pid is lower than this one's. After a reboot on 2026-10-16 the ids wrapped: waiters started between 16:07 and 17:26, with pids 50460 to 71218, all queued ahead of one started at 16:04 with pid 96274. That waiter could only wait out its timeout while lower pids kept arriving. The docs and the LOAD rule promise one waiter at a time in queue order.
