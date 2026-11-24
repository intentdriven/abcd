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
---

make precheck lists its audit packages by hand, and two audit tests (gittest's hand-made bare repository check and the loop's drive git table) are outside the list, so precheck passes a tree preflight refuses
