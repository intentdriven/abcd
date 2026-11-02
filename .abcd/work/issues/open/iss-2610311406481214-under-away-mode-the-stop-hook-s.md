---
schema_version: 1
id: "iss-2610311406481214"
slug: "under-away-mode-the-stop-hook-s"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "landing PR #1291 under away mode"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/autonomy/owedread.go"
remedy: "Revising dcn-2610180936568358 and dcn-2610181003463367 only by adding a session key: the guard's created-branch record (CreatedBranches) and a run's state carry the session that cut the branch or started the run, so the stop hook in a copy two live sessions share lists each session's own branches as its debt and names the other's as information; every per-copy source stays, a branch with no session key stays owed, and the hook names the shared copy as a breach of the one-session-per-checkout rule with the worktree store as its fix."
---

Under away mode the stop hook's owed-work list held Eve's stop, wake after wake, on build/run-2610311213205613-lane-1 (a drain session's build lane) and chore/drain-session-records (the drain session's own branch), neither of which this session made or may push. The reader's ownership is per working copy by design (findOwn in internal/core/autonomy/owedread.go: this copy's HEAD reflog, the branches its guard recorded creating, the lanes of live runs in its local tier, and pull requests under its reservation key; dcn-2610180936568358, dcn-2610181003463367), and both branches entered it because the drain session and this one (Eve) were both driving from the same main checkout, the drain from the start and Eve for its builds and records: the run state sits in that checkout's .abcd/.work.local/run/ and the drain's checkouts are in its reflog. The hook never says that two live sessions share the copy, so a session reads a peer's work as its own debt and can neither push it, which AGENTS.md forbids, nor stop.
