---
schema_version: 1
id: "iss-2611050753557564"
slug: "a-terminal-drive-ends-its-run-at-a-limit"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/terminal.go"
remedy: "Wait out a limit for as long as its reset is away, keep probing a limit with no reset on a capped backoff, keep waiting through outages and stuck queues at a capped interval, and end at a bound the person sets with --until, eight hours when none is given."
---

A terminal drive ends its run at a limit resetting past eight hours and after eight hours of outage or no movement, so an unattended run stops at the first weekly limit.
