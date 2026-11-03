---
schema_version: 1
id: "iss-2611030450363508"
slug: "the-build-loop-s-validate-stage-passed"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "coordinating the build lanes, 2026-11-03"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop"
remedy: "Run make precheck (or preflight's lint stage) as part of the implementer's definition of done before its receipt is accepted, and keep AGENTS.md a set margin under the readers' cap with its detail in the rules domains."
---

The build loop's validate stage passed run-2611030127507347 through two rounds on a head whose preflight then failed two lint tests (AGENTS.md past the 32768-byte readers' cap, and a raceEnabled read in init that the plain-guards scan cannot classify): neither the implementer's definition of done nor the reviewers ran the lint gates, so the failure appeared only after the records commit. AGENTS.md also stood 16 bytes under its cap on main, so any lane adding a clause trips it.
