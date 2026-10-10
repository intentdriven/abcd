---
schema_version: 1
id: "iss-2610102020447522"
slug: "two-internal-core-interview-tests-fail-under-machine-load"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "abcd-60 drain run 2026-10-10, lane for iss-2610090642392144"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/interview"
remedy: "Find the interview tests' fixed exec WaitDelay / timeout budgets and derive them from the test's deadline (t.Deadline) or a scaled bound, as the openaiapi stream test now does with capWait, so a loaded runner slows them without failing them; reproduce under a CPU-bound sibling before and after."
---

Two internal/core/interview tests fail under machine load with 'WaitDelay expired' (a different test on each run), on 2026-10-10 in two consecutive make preflight runs of a drain lane at load averages of 31 to 57 on 16 cores; the package passes when run alone and under the race lane. The tests' subprocess wait is a fixed wall-clock budget that a loaded runner overruns, the same shape as the openaiapi stream test fixed for iss-2610080243367274.
