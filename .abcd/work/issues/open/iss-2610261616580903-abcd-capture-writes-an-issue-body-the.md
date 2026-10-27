---
schema_version: 1
id: "iss-2610261616580903"
slug: "abcd-capture-writes-an-issue-body-the"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "lowcpu run 2026-10-25: landing preflight of PR 1134"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/capture.go"
remedy: "Before writing, run the composed body through the same render check record_schema applies at the gate and refuse naming the construct and how to write it, as the grounds check already refuses text the site cannot render."
---

abcd capture writes an issue body the site renderer refuses, and nothing says so until the next record-lint run. The text is written as given: the capture path runs neither the prose cleaner the consistency ingest uses nor the body-render check record_schema applies at the gate. So a placeholder named after a void HTML element (input, link, source), or a link to a path the site cannot resolve, lands in the record and is refused only later. On 2026-10-26 two such records stopped a landing preflight after six hours in the load gate's queue.
