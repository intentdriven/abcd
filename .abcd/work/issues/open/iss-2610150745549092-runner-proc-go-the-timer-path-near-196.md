---
schema_version: 1
id: "iss-2610150745549092"
slug: "runner-proc-go-the-timer-path-near-196"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "abcd land security review round 2, 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/runner/proc.go"
remedy: "Move internal/core/runner/proc.go and internal/core/tools/install.go onto procgroup.Bound (exec.Cmd.Cancel through CommandContext), so exec serialises the group kill with its own reaping and the kill can never address a process group whose leader has already been reaped"
---

runner/proc.go (the timer path near :196) and tools/install.go (near :325) kill a command's process group with syscall.Kill(-pid) from a select that races the goroutine calling cmd.Wait: when the leader exits at the moment the timer fires, Wait can reap it first, and in principle the group id could be reused by then, so the SIGKILL would reach another group. The window is tiny and the pattern predates the abcd land change, which moved both onto procgroup.Lead/Kill token for token; abcd land's own gate is immune because exec serialises its Cancel. Route exists without abcd: no, this is abcd's own process handling, so abcd fixes it.
