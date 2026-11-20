---
schema_version: 1
id: "iss-2610102020447522"
slug: "two-internal-core-interview-tests-fail"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "abcd-60 drain run 2026-10-10, lane for iss-2610090642392144"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/interview"
remedy: "Find the interview tests' fixed exec WaitDelay / timeout budgets and derive them from the test's deadline (t.Deadline) or a scaled bound, as the openaiapi stream test now does with capWait, so a loaded runner slows them without failing them; reproduce under a CPU-bound sibling before and after."
resolution: "the quoted 'WaitDelay expired' is not a message any of these tests' own waits gives: in this package it can come only from the git reads the tree guard makes, whose WaitDelay miss was fixed under iss-2610100846469473 (no 50ms WaitDelay on a git with no deadline) and iss-2610272041031418 (a deadline-bound git read reads its output to the end before waiting). This change removes the fixed waits that remained in the tests: each stubbed turn's dispatch timeout, the three watchdogs on guarded readings and the stubbed turn's sleep now derive from the test binary's deadline (t.Deadline), three quarters of the time left, so a loaded runner given a longer deadline slows them without failing them, and a wait that runs out fails the test with its own message before the binary's timeout panic. Two plumbing tests hold the dispatch timeout and the sleep, read where they are built, and the shared budget the watchdogs call to both; that each watchdog calls it is held by reading the code, not by a test. The watchdogs guard against a hang only: the followed-link byte-budget tests plant a file that cannot be opened, so a reading that read it before refusing fails at once. No reproduction under a CPU-bound sibling was run, before or after, since a load experiment on this machine needs the person's consent; the change was proven red by the test deadline alone (-timeout 15s and 7m) and by moving the followed-link charge after the read"
impact: internal
---

Two internal/core/interview tests fail under machine load with 'WaitDelay expired' (a different test on each run), on 2026-10-10 in two consecutive make preflight runs of a drain lane at load averages of 31 to 57 on 16 cores; the package passes when run alone and under the race lane. The tests' subprocess wait is a fixed wall-clock budget that a loaded runner overruns, the same shape as the openaiapi stream test fixed for iss-2610080243367274.
