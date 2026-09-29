---
schema_version: 1
id: "iss-263"
slug: "dispatch-unparseable-issue-reads-as-not-found"
severity: "nitpick"
category: "ux"
source: "impl-review"
found_during: "spc-26 build, ruthless-reviewer note"
found_at: "internal/core/record/record.go"
resolution: "Overtaken by 4323948fc: describeIssue consults the skipped roster capture.List returns and faults with ErrSkippedRecord naming the file and the reader's own parse error, never 'not found'. The record's own shape, a never-closed block, is pinned by TestDescribeUnparseableIssueIsNotNotFound in 1ebed1fa0."
impact: internal
resolved_by:
  commit: "4323948fc"
---

describeIssue discards ListResult.Skipped, so an issue file that exists but is unparseable (broken frontmatter) makes abcd iss-N report 'not found in the issue ledger' — a diagnostic that misleads about a record physically present in a status dir. Surface the skip roster in the fault: 'iss-N present but unreadable at <path>: <parse error>'.

## Grounds

- pursued: abcd iss-N on a present but unparseable issue file names the file and the parse error; a broken-frontmatter issue that answers not found in the issue ledger would show it wrong
