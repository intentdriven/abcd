---
schema_version: 1
id: "iss-2611060835391036"
slug: "testagroupwritablehomeisrefusedunderadir"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "v0.16.2 cut"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/fsutil/home_scope_mode_test.go"
remedy: "Have the test pick a temporary directory whose group is the account's own."
---

TestAGroupWritableHomeIsRefusedUnderADirectoryService fails on macOS when TMPDIR names the system's shared temporary directory: the test's folders take that directory's wheel group, so its case with no directory service is refused (`internal/fsutil/home_scope_mode_test.go:352`). Anyone running the package that way gets a red their change did not cause.
