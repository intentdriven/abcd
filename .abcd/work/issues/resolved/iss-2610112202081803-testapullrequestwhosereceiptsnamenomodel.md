---
schema_version: 1
id: "iss-2610112202081803"
slug: "testapullrequestwhosereceiptsnamenomodel"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "merge-queue run of #935, reported by a peer session"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/land_test.go"
remedy: "Add a bare-repository constructor to internal/gittest that applies the same repo-level maintenance config as NewRepo. Use it in newLandFixture and at every other git init --bare site in the tests. Then add a guard test that refuses a raw init --bare in a _test.go file outside gittest, so the sibling cannot come back."
resolution: "gittest.InitBare and gittest.CloneBare write NewRepo's maintenance-off config from one shared list; all thirteen hand-made bare repositories in tests use them, and a guard test refuses a new one."
impact: fix
resolved_by:
  commit: "41289dcef"
---

TestAPullRequestWhoseReceiptsNameNoModelIsNotOpened is flaky and twice dropped a merge-queue entry (the run that landed #935): t.TempDir cleanup fails with "origin.git: directory not empty". newLandFixture in internal/core/implement/loop/land_test.go creates its bare origin with a plain git init --bare, so the origin lacks the gc.auto=0, gc.autodetach=false, maintenance.auto=false and core.fsmonitor=false config that gittest.NewRepo writes for exactly this race (iss-252). A detached auto-gc child spawned by a push into the origin outlives the test and writes objects while cleanup removes the directory. About 17 other git init --bare calls in 15 test files share the gap.
