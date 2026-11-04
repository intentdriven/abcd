---
schema_version: 1
id: "iss-2611040337136199"
slug: "a-host-run-reviewer-s-return-has-no"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "the v0.17.0 cut and the overnight lanes, 2026-11-04"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/validate.go"
remedy: "Have the host record the model it started for each stage in the run record, read from the agent's own report rather than from the brief; a test hands back a return with no model and asserts the record says unknown rather than the tier asked."
---

A host-run reviewer's return has no model field and a receipt's model is self-reported, so nothing shows which model a stage used, although the driver asks for tier frontier: a subscription test on 2026-11-04 could not say what ran its reviewers.
