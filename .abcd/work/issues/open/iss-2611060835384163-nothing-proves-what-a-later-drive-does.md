---
schema_version: 1
id: "iss-2611060835384163"
slug: "nothing-proves-what-a-later-drive-does"
severity: "minor"
category: "tech-debt"
source: "agent-observation"
found_during: "v0.16.2 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/drain_drive_signal_test.go"
remedy: "Add a test that kills a drive mid-stage and starts another, and add the case to the live proof."
---

Nothing proves what a later drive does with a lane whose driver died hard rather than on a signal: no test kills a drive mid-stage and starts another, and the live proof does not cover it. An unattended run that loses its process this way resumes on behaviour nobody has watched.
