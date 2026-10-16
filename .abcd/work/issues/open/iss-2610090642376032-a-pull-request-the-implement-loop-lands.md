---
schema_version: 1
id: "iss-2610090642376032"
slug: "a-pull-request-the-implement-loop-lands"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "2026-10-07/08 autonomous drain and v0.13.3 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/land.go"
remedy: "After arming auto-merge, the landing confirms the pull request is in the queue (enqueue it when CLEAN and unqueued, toggle auto-merge as the fallback) and the run's watch treats not-queued-and-not-merged as a terminal state it reports, naming the failed merge-group check."
---

A pull request the implement loop lands can leave the merge queue silently. Twice it sat CLEAN with auto-merge armed and never entered the queue (#837, #844; the enqueue mutation then failed with a forge error and only toggling auto-merge queued it), and four times across this run and the v0.13.2 cut a failed merge-group check (the macOS job cap, or a flaky test) dropped it from the queue (#839, #854, #867, #869) with nothing re-queuing it. The loop arms auto-merge and walks away, so a run that trusts it stalls unseen; one drop cost about 45 minutes in this run.

2026-10-15, evidence from #1011: the merge queue removed it for failed_checks at 07:48Z (the ubuntu race step hit its 30-minute limit; run 38697714235), and that removal also cleared auto-merge. So "not armed and not queued" has two causes, a dropped arming and a failed queue run. A watch, or `abcd land` on a re-run, should read the timeline's RemovedFromMergeQueueEvent reason and name it before arming again, rather than re-queue silently into the same failure.
