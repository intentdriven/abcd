---
schema_version: 1
id: "iss-2610280051498465"
slug: "a-file-an-agent-writes-with-the-host-s"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "review of the fix for iss-2610240859590333, 2026-10-28"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/hook_occupancy.go"
remedy: "Add a PreToolUse hook on the host's Write and Edit tools that resolves the target file's working tree and, when occupancy.Check reports it shared, denies the call with the same message the verbs give, naming the worktree store; keep linked worktrees and single-session checkouts unrefused, and test it with a fixture machine as hook_occupancy_test.go does."
resolution: "fixed by the guard hook taking the host's file tools (Write, Edit, MultiEdit, NotebookEdit): a file that lands in a main checkout two or more live sessions share, outside its ignored local tier, is denied with the record verbs' refusal"
impact: fix
resolved_by:
  commit: "99c814d01"
---

A file an agent writes with the host's own edit tools into a main checkout two or more live sessions share is not refused. On 2026-10-28 the record writers (capture, intent, decide, spec close and the index generator) began refusing a shared main checkout through a per-session occupancy marker (iss-2610240859590333), but an agent's direct Write or Edit of a record file, or of any tracked file, goes through the host's file tools, which the shell guard never sees. The 2026-10-24 incident used the verbs, so the verbs were closed first; the same mistake made with the file tools lands the same way.
