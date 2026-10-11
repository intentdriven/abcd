---
schema_version: 1
id: "iss-2608261447039180"
slug: "an-unknown-frontmatter-property-hides-a"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "bughunt-b-round-9"
found_at: "internal/core/lint/schema.go"
resolution: "checkIssueRecordShape flags any issue-frontmatter key outside the shared issueschema known set, so an unknown-key record is no longer silently skipped and invisible"
impact: fix
resolved_by:
  commit: "b55c5c38"
---

an unknown frontmatter property hides a ledger record the same way a missing one did