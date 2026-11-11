---
schema_version: 1
id: "iss-2611101317500649"
slug: "implement-step-and-implement-record"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/loop.go"
remedy: "Read a run with no live driver and no state change for a set number of days as stale, so it no longer counts as in progress, have step pick the run whose lane worktree it is run in, and list stale runs with a way to close them; a test leaves an old run idle and steps a newer one without --run."
---

implement step and implement record refuse to pick a run when more than one run in the checkout reads as in progress, and runs abandoned weeks ago still read so, so every step needs --run. On 7 Nov at 09:45 and 10:12, 11 and then 12 runs from October were listed as in progress, and on 9 Nov at 22:22 there were 14; each lane had to name its run. The selection is in internal/core/implement/loop/loop.go. This refines iss-2610101659347329, where implement receipt is refused the same way although its path names the run.
