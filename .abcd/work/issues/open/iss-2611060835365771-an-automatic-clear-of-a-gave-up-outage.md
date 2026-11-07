---
schema_version: 1
id: "iss-2611060835365771"
slug: "an-automatic-clear-of-a-gave-up-outage"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "v0.16.2 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/outage.go"
remedy: "Give the automatic clear its own log entry, and surface that a give-up happened."
---

An automatic clear of a gave-up outage record is logged as a clear by hand and drops NotifyPending (`internal/core/implement/outage.go:708`, `internal/surface/cli/implement_outage.go:105`). The log says a person acted where none did, and a person checking later is not told the run gave up.
