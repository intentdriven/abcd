---
schema_version: 1
id: "iss-2611120545409910"
slug: "the-audit-test-of-the-state-file-s"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/save_frame_test.go"
remedy: "Have the audit also flag any function outside the two writers that passes the state path, or a value assigned from it, to another call, and any assignment to a field of a wholeSave value; or keep wholeSave and its fields in a file of their own and flag any edit to that file"
---

The audit test of the state file's writers flags a function only when it both names the state path and calls a write itself, and a wholeSave value only when it is built as a literal with fields, so a write made through a helper that is handed the path, or a right assembled by assigning its fields, passes the test unnoticed
