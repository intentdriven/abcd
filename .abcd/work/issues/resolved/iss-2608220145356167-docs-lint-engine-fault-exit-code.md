---
schema_version: 1
id: "iss-2608220145356167"
slug: "docs-lint-engine-fault-exit-code"
severity: "nitpick"
category: "bug"
source: "agent-finding"
found_during: "bughunt-b-round-7"
found_at: "internal/surface/cli/cli.go"
resolution: "docs lint engine/config faults return exitError code 2 with a path-scrubbed message"
impact: fix
resolved_by:
  commit: "4d06ab1"
---

abcd docs lint returns a bare engine error which cli.Run maps to exit 1, the same code as a blocker finding, while abcd lint and record-lint exit 2 for an engine fault; a consumer keying on exit >=2 reads a docs lint that never ran (a config containment refusal) as an ordinary findings-pass