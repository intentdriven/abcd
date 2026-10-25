---
id: dcn-2610222224054666
slug: ctrl-c-or-a-hang-up-stops-a-drive-s
date: 2026-10-22
---
2026-10-22 — Ctrl-C or a hang-up stops a drive's running agent and leaves the open lane resumable (facilitator, on delegation, itd-2610222215485265): the runner's process group is stopped, the open lane keeps its worktree, branch and reservation, and the drive prints the run, the lane and when the reservation lapses, so the next drive or any host resumes it; nothing is handed back for an interruption, and a second signal exits at once as today (grounds: commands/drain.md on signals; the stuck hand-back's meaning)
