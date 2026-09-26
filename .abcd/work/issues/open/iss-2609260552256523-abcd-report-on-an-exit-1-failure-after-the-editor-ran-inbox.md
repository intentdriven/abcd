---
schema_version: 1
id: "iss-2609260552256523"
slug: "abcd-report-on-an-exit-1-failure-after-the-editor-ran-inbox"
severity: "minor"
category: "bug"
source: "drift-detection"
found_during: "v0.11.0 release gate: brief-surface cross-check (autonomous run A, abcd-a2)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/report.go"
---

`abcd report`: on an exit-1 failure after the editor ran (inbox not creatable, id draw exhausted, temp file not creatable) the kept editor draft is named only for ErrRefused errors (internal/surface/cli/report.go), so the text survives but the user is not told where it is; commands/report.md:53 documents only exit 2. Found by the v0.11.0 brief-surface cross-check (x-063).
