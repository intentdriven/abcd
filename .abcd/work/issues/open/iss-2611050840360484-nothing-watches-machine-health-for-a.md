---
schema_version: 1
id: "iss-2611050840360484"
slug: "nothing-watches-machine-health-for-a"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "coordinating build lanes, 2026-11-05"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/load_wait.go"
remedy: "A drive reads the same floors between stages and stops starting stages at a person's own threshold, naming the reading and the threshold and starting again when the reading falls. A test feeds a reading past the threshold and asserts no stage starts."
---

Nothing watches machine health for a long unattended drive: memory pressure, a growing system daemon or load can climb for hours while stages keep starting. The load gate reads load and memory at a test's start only (itd-2611012048006057), so a stage that is not yet a test starts whatever the machine reads, and a slow climb is never named.
