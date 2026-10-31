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
remedy: "Attribute each branch to the session that made it (the run state's session, the worktree's creating session, or a branch-to-session record written when a verb cuts it) and list only this session's branches as owed; name a peer's unpushed branch as information, never as an item that holds the stop."
---

Under away mode the stop hook's owed-work list names branches another session owns as this session's owed work: on 2026-10-31 it held Eve's stop, wake after wake, on build/run-2610311213205613-lane-1 (a drain session's build lane, 'has commits not pushed') and chore/drain-session-records (the drain session's own branch), neither of which this session made or may push. The reader walks every local branch of the checkout with no notion of which session started it, so a session either pushes a peer's work, which AGENTS.md forbids, or cannot stop.
