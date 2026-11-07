---
schema_version: 1
id: "iss-2611070737101875"
slug: "make-precheck-does-not-run"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "follow-up list from a release cut"
origin: researcher-authored
production_mode: hand-written
found_at: "Makefile"
remedy: "Add both audit tests to precheck's named list beside the launch command-surface test; read the hermetic scan's speed first to confirm it fits precheck's minutes budget."
refines: [iss-2611061656036069]
---

make precheck does not run TestTestGitCallsAreHermetic (internal/gittest) or TestNoPrivateDelimiterCompare, so a test file spawning git outside the helper cost a 91-minute preflight and the delimiter test cost another 64-minute one. Both are cheap audits that precheck exists to run first.
