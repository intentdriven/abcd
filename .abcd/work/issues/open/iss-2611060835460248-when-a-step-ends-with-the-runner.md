---
schema_version: 1
id: "iss-2611060835460248"
slug: "when-a-step-ends-with-the-runner"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "v0.16.2 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/stop.go"
remedy: "Judge an await by whether a launch is live, and test it."
---

When a step ends with the runner returning no receipt, the await has no claim and reads as unknown under the drive's own live process (`stop.go:501`), so the first signal to the drive settles nothing, keeps the claims and prints that an agent is still in its call when none is (`terminal.go:464`). The person waits on an agent that is not there.
