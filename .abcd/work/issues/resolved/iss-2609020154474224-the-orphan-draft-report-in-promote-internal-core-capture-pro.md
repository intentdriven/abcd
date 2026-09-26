---
schema_version: 1
id: "iss-2609020154474224"
slug: "the-orphan-draft-report-in-promote-internal-core-capture-pro"
severity: "minor"
category: "ux"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/promote.go"
resolution: "The orphan-draft remedy is delimited as a CommonMark code span whose fence is one backtick longer than any run inside the command (codeSpan), so grounds carrying backticks cannot close it and the delimited text is always the whole command."
impact: fix
resolved_by:
  commit: "14ab2e07"
---

The orphan-draft report in Promote (internal/core/capture/promote.go) delimits its remedy with backticks — complete the link with (backtick)abcd capture promote ...(backtick) — and the remedy now carries the promotion's own grounds, which are free prose. Grounds containing a backtick therefore close the delimiter early, so a reader (or a test, or any tool) copying the text between the backticks gets a truncated command; the remedy itself is well formed and runs when copied whole. The fix must establish a delimiter the argument cannot close, so what sits between the delimiters is always the whole remedy.

## Grounds

- pursued: whatever reads the text between the remedy's delimiters gets the whole command; grounds with a backtick run yielding a truncated span would show it wrong
