---
schema_version: 1
id: "iss-2611232155463657"
slug: "make-precheck-lists-its-audit-packages"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "release run 2026-11-23"
origin: researcher-authored
production_mode: hand-written
found_at: "Makefile"
remedy: "add a test that finds every source-reading audit test and refuses when precheck does not run it, and add the two missing ones"
resolution: "TestPrecheckRunsEveryAuditNamedTest refuses an audit-named test reaching the tree that precheck does not run; the selection now sees an ancestor handed to a helper and a walk from the parent, which brings in the drive's git table"
impact: internal
---

make precheck lists its audit packages by hand, and two audit tests (gittest's hand-made bare repository check and the loop's drive git table) are outside the list, so precheck passes a tree preflight refuses
