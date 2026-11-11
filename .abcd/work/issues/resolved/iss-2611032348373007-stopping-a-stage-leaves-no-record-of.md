---
schema_version: 1
id: "iss-2611032348373007"
slug: "stopping-a-stage-leaves-no-record-of"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "coordinating the build lanes and the walled subscription test, 2026-11-03"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop"
remedy: "Record the worktree's git status in the run record when a stage stops, and show a lane's uncommitted files on its status line, so a resume sees them before starting an agent; a test stops a stage with a dirty worktree and asserts both."
related_intents: [itd-2611041549461012]
resolution: "A stop now saves a lane's uncommitted work aside and records where it went and how many files, a take-over saves it aside before any fresh agent starts, and abcd implement resume counts each lane's uncommitted files. The record keeps the saved-aside path and a count rather than raw git status, and implement status does not show it; iss-2611080058096268 tracks the status display."
impact: fix
resolved_by:
  intent: "itd-2611041549461012"
  commit: "f175c4db5c64edfd20fd1f509472af560a6b8a1c"
---

Stopping a stage leaves no record of what it wrote, and nothing shows a lane's uncommitted work: on 2026-11-03 a coordinator stopped an implementer and recorded that it had written nothing, while its worktree held ten changed files, found only by the next session's own git status; implement status and step show nothing of them, so a resume would start a fresh implementer over them.
