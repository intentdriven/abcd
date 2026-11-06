---
schema_version: 1
id: "iss-2611060835314533"
slug: "a-drive-that-ends-during-an-outage"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "v0.16.2 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/terminal.go"
remedy: "Close the record on the way out when a probe answers, or say at the drive's end that it stands and how to clear it."
---

A drive that ends during an outage leaves any outage record it opened still open, with nothing left to probe it (`terminal.go:1783`). A later step with no session, or a later drive with no bound, is held until someone probes or clears the record, so work waits on a service that may already be back.
