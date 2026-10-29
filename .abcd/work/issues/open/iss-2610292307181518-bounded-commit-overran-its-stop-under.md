---
schema_version: 1
id: "iss-2610292307181518"
slug: "bounded-commit-overran-its-stop-under"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "preflight of the records PR for rcp-6083de2cef52 (race stage)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gitutil/bounded.go"
remedy: "Reproduce the overrun with a hook that forks late (one that sleeps before starting its child) and find what held the run past the WaitDelay and the sweep; if it is the descendant snapshot, re-read the process tree when the sweep runs, keeping git in abcd's process group as bounded.go intends, then keep the test's bound and add the late-fork case to it"
---

A bounded git commit whose hook never finishes overran its stop by far more than its design allows. In a preflight's race stage, TestAPickCommitAndASyncMergeWhoseHookNeverFinishesAreRefused (internal/core/implement/loop/pickgit_deadline_test.go:105) failed: the pick commit, under a 1s deadline with a pre-commit hook that execs sleep 60, came back after 49.7s against the test's 20s bound, while the worst stop the code allows is about 21s (the 1s limit, a process-table read under its 5s cap, the 5s WaitDelay, then the descendant sweep's 5s grace and its reads of up to 5s each). The plain test stage passed the same package in the same run. The five-minute load was about 280 to 340, mostly Spotlight indexing and a sync client rather than the tests, on 16 cores. An unproven hypothesis: under that load git forks the hook after Cancel has taken its snapshot of git's descendants, so the sleep is outside the set the sweep signals and kills, and something else holds the run until the sleep nears its end. The overrun is in a test bound; no user path that hangs has been shown yet, though abcd implement step runs pick commits through the same call.

## Evidence

Reproduced on an unloaded machine on 2026-10-29 with a copy of the code that times each phase and can pause the stop between the process-table snapshot and the signals. The hook in each case execs a long sleep.

- With nothing injected and a 1s limit, the call came back in 1.04s: the snapshot took about 19ms and found the hook, and the sweep took about 19ms. Nothing was left running.
- With a 6ms limit and a 2s pause after the snapshot, the snapshot found no descendant, git started the hook during the pause, and the call came back in 7.03s (the limit, the pause and the whole 5s WaitDelay). The hook's sleep was still running afterwards. So a child git starts after the snapshot does escape the stop, and it holds git's output until the WaitDelay closes it.
- That escape costs at most the WaitDelay. It cannot account for 49.7s: with the test's 1s limit it comes to about 6s.
- The escape is closed: the stop now freezes git with SIGSTOP before it reads git's descendants, and each descendant as it finds it, so nothing is started while the stop is under way, and continues every process it stopped once it has signalled it. The sweep freezes what is still running the same way before it kills it. Its reads can now take twice one read's 5s limit, and the sweep's freeze one more, so the worst stop the code allows is about 26s, reached only when every ps run takes nearly its whole limit and still answers. The overrun this record reports stays open.

Three spans have no bound of their own and are the candidates for the overrun. Starting git is not covered by the deadline, since the context is consulted only once the process exists. The wait for git to exit after the kill blocks for as long as git cannot die, as in a long uninterruptible disk wait under heavy I/O. And the process-table read's own wait has no WaitDelay, so its 5s cap holds only if ps can die when it is killed. None has been shown to be the cause.

