---
schema_version: 1
id: "iss-2611120545489574"
slug: "a-claim-writes-the-run-s-control-record"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/control.go"
remedy: "Write the control record only after the record line is saved, under the same lock, or check that the line fits before the control record is written"
---

A claim writes the run's control record before the record-only save appends its line to the run's record, and that save cannot drop located findings to make room as a whole save does, so on a state file near its size bound the claim returns an error with the new driver already recorded and no line saying so
