---
schema_version: 1
id: "iss-2611120545484783"
slug: "a-run-whose-control-record-names-no"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/control.go"
remedy: "Record in the control record that a session stepped the run without a claim, or refuse to claim a run with no recorded driver whose record shows a step since its last claim, so such a run is not taken as undriven"
---

A run whose control record names no driver is claimed by a resumed start or a drain and then saved at this version, although a session whose own claim claimed nothing, because the process table or its host could not be read, may be the one driving it
