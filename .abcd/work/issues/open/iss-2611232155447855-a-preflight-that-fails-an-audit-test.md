---
schema_version: 1
id: "iss-2611232155447855"
slug: "a-preflight-that-fails-an-audit-test"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "release run 2026-11-23"
origin: researcher-authored
production_mode: hand-written
found_at: "Makefile"
remedy: "run precheck's audit tests before preflight waits at the load gate, and stop the run at once when one fails"
refines: [iss-2610161639025702]
---

A preflight that fails an audit test fails only after it has waited at the load gate, often for an hour or more, although the audit tests need no queue and finish in minutes
