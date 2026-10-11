---
schema_version: 1
id: "iss-2608270500209138"
slug: "the-aws-access-key-secret-pattern"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "github-ledger-dedup-2026-08-27"
found_at: "internal/adapter/scanner/patterns.go"
resolution: "the AWS access-key secret pattern covers the STS/temporary prefix family, not only AKIA (#358)"
impact: fix
---

the aws_access_key secret pattern matches only AKIA, so an ASIA-prefixed temporary STS access key ID ships and writes to history un-redacted while its shape-identical AKIA sibling is a hard_fail block. GitHub mirror: #358