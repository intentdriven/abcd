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

A bounded git commit whose hook never finishes overran its stop by far more than its design allows. In a preflight's race stage, TestAPickCommitAndASyncMergeWhoseHookNeverFinishesAreRefused (internal/core/implement/loop/pickgit_deadline_test.go:105) failed: the pick commit, under a 1s deadline with a pre-commit hook that execs sleep 60, came back after 49.7s against the test's 20s bound, while RunCommitCommandBounded's designed worst case is about 11s (the 1s limit, the 5s WaitDelay, then the 5s descendant sweep). The plain test stage passed the same package in the same run. The five-minute load was about 280 to 340, mostly Spotlight indexing and a sync client rather than the tests, on 16 cores. An unproven hypothesis: under that load git forks the hook after Cancel has taken its snapshot of git's descendants, so the sleep is outside the set the sweep signals and kills, and something else holds the run until the sleep nears its end. The overrun is in a test bound; no user path that hangs has been shown yet, though abcd implement step runs pick commits through the same call.
