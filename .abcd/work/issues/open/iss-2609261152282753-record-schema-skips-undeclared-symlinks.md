---
schema_version: 1
id: "iss-2609261152282753"
slug: "record-schema-skips-undeclared-symlinks"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: fix3-lintA risks"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/schema.go"
---

record_schema skips two symlink shapes in a record store with nothing said: (a) in a bucketed store, a symlinked entry at the store root with an undeclared name and no .md suffix (e.g. foo -> dir) is neither reported as an undeclared bucket nor read, because a symlink DirEntry is not a directory and falls to the markdown suffix test; (b) inside a declared bucket, or in a flat store, a symlinked subdirectory is skipped the same way, while a real subdirectory draws the undeclared-subdirectory finding. Either one hides a lifecycle state from every check. The sibling of iss-2609261133371466, which named only a DECLARED bucket that is a link.
