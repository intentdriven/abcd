---
schema_version: 1
id: "iss-2610230814127624"
slug: "the-status-line-deadline-tests-race-a"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "lane C preflight of feat/local-model-pin-r2, 2026-10-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/statusline_deadline_test.go"
remedy: "Drive the status line's git deadline from an injected clock or a fake git that blocks on a channel, so the tests assert the deadline path deterministically instead of racing a real 2-second budget against git on a loaded machine"
---

Two status-line deadline tests fail under machine load and pass in isolation: TestStatuslineSlowBranchStillRendersThePartialRow (stdout empty, the call returned in 2.003s against a 2s git budget) and TestStatuslineKillsTheRootCommitGit (the root-commit git timed out at 2s, so the 'git did not say whether abcd manages the checkout' note was replaced by 'git did not name the checkout within 2s'). Both failed in a make preflight of feat/local-model-pin-r2 on 2026-10-23 at a load of roughly 27 to 40 on 16 cores, and both pass on origin/main and on that branch when run alone. The first is the test iss-2610090642407850 resolved on 2026-10-09; its timing margin is gone again under load, and the second shows the same root cause: a real 2-second budget raced against git on a loaded machine.
