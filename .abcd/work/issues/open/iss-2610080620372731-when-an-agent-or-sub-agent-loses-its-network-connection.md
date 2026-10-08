---
schema_version: 1
id: "iss-2610080620372731"
slug: "when-an-agent-or-sub-agent-loses-its-network-connection"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "the guard-refusal session of 2026-10-07"
origin: researcher-authored
production_mode: hand-written
found_at: "agents and sub-agents in autonomous runs (internal/core/implement)"
remedy: "Retry a lost connection on a widening schedule (wait 1 minute, retry; 5 minutes, retry; 10 minutes, retry; then an hour), and record the outage and each retry in a machine-local channel, such as the shared run state and run log, so sibling agents and sub-agents see it and hold off without reaching each other over the network."
---

When an agent or sub-agent loses its network connection mid-task, the work stalls or fails instead of waiting the outage out, and this keeps happening on the product thinker's machine. An agent that notices a lost connection should pause and retry on a widening schedule: wait one minute and try again, then five minutes, then ten, then an hour. Sibling agents should also learn of the outage without depending on each other over the network, since they may have lost the connection too: a channel on the machine itself, such as the run state and run log abcd already shares between sessions, would let one agent record the outage and the others hold off instead of each failing on its own.
