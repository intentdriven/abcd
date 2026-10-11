---
schema_version: 1
id: "iss-2608221456599229"
slug: "capture-list-status-json-emit-skipped"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "all-dimensions bug-hunt round 7"
found_at: "internal/core/capture/workflow.go"
resolution: "relativiseLedgerPaths scrubs skipped[].error at the same choke point as the path field."
impact: fix
resolved_by:
  commit: "7a883f5"
---

capture list/status --json emit skipped[].error carrying a raw absolute path beside the deliberately-relativised skipped[].path, in an exit-0 success envelope the CLI error-path scrubber never sees.