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
remedy: "Keep per-copy ownership and its sources (dcn-2610180936568358, dcn-2610181003463367), and keep an unattributable branch owed. When the copy has two or more live sessions, the stop hook names that breach of the one-session-per-checkout rule and the worktree store as its fix, beside the items, instead of presenting them as this session's debt alone."
---

Under away mode the stop hook's owed-work list held Eve's stop, wake after wake, on build/run-2610311213205613-lane-1 (a drain session's build lane) and chore/drain-session-records (the drain session's own branch), neither of which this session made or may push. The reader's ownership is per working copy by design (findOwn in internal/core/autonomy/owedread.go: this copy's HEAD reflog, the branches its guard recorded creating, the lanes of live runs in its local tier, and pull requests under its reservation key; dcn-2610180936568358, dcn-2610181003463367), and both branches entered it because the drain session and this one were driving from the same main checkout: the run state sits in that checkout's .abcd/.work.local/run/ and the drain's checkouts are in its reflog. The hook never says that two live sessions share the copy, so a session reads a peer's work as its own debt and can neither push it, which AGENTS.md forbids, nor stop.
