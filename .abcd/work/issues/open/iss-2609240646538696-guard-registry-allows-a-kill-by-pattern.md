---
schema_version: 1
id: "iss-2609240646538696"
slug: "guard-registry-allows-a-kill-by-pattern"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/defaults/guard.json"
---

The shell guard's bundled hazard registry has no entry for a kill by pattern: `abcd guard check --command "pkill -f 'make preflight'"` answers allow, and so does `killall make`. On 2026-09-23 a lane agent in autonomous run A stopped its own gate with `pkill -f "make preflight"` on a machine where several sessions ran gates at once. The pattern matched every session's preflight, so two peer lanes lost theirs, and the loss read as an unexplained SIGTERM for about half an hour until the agent's command was found. The run's lane brief then forbade pattern kills, which is the discipline rung. Wanted: a registry entry that blocks, or at least warns on, `pkill -f`, `pkill` and `killall` with a name or pattern, and a kill whose pid list comes from a pattern search, naming the successor: kill the pid you recorded when you started the process, or its own process group. The LOAD rule domain (iss-2609210828122412) already states the same rule for a test's own children.
