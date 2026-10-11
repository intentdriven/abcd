---
schema_version: 1
id: "iss-2608261437041050"
slug: "a-ledger-record-missing-schema-version"
severity: "major"
category: "bug"
source: "agent-observation"
found_during: "bughunt-b-round-9"
found_at: "internal/core/lint/schema.go"
resolution: "record_schema enforces the issue store's required frontmatter from the shared issueschema leaf; status board renders skips; live record repaired"
impact: fix
resolved_by:
  commit: "9a24c32d"
---

a ledger record missing schema_version vanishes from capture surfaces while every gate stays green