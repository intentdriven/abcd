---
schema_version: 1
id: "iss-2610142235017884"
slug: "kill-by-holder-and-kill-by-search-block"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "security review of feat/guard-port-blocks for v0.15.0"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/defaults/guard.json"
remedy: "Exempt -0, -s 0 (and -s0, --signal 0) and -l/-L from kill-by-holder and kill-by-search, with fixtures at allow and the real signals beside them at block; or, if exempting is judged unsafe, name the limit in commands/guard.md with the read-only alternative (lsof -t -i :3000, pgrep)."
---

kill-by-holder and kill-by-search block a kill that sends no signal: a signal-0 liveness probe (kill -0 $(lsof -t -i :3000), kill -s 0 $(...)) or a signal-name listing (kill -l $(...)) only asks whether the process exists, or names a signal, and stops nothing, yet blocks as a kill of whatever holds the port or matches the search would.
