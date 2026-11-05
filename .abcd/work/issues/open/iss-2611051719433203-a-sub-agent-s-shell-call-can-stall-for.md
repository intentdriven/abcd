---
schema_version: 1
id: "iss-2611051719433203"
slug: "a-sub-agent-s-shell-call-can-stall-for"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "build run, 5 November"
origin: researcher-authored
production_mode: hand-written
remedy: "A coordinator running agents through a host keeps a watch over each agent's last activity, the implement page names the practice, and abcd's own status names a stalled agent where abcd started it."
---

A sub-agent's shell call can stall for hours while the agent still reads as running, with nothing that says so; on 5 November one build lane sat for over five hours and was noticed only by reading its worktree. The agents here are the host's, which abcd does not see, so no drain lane can take this and it needs a person's reading of where the watch belongs.
