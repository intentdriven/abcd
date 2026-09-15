---
schema_version: 1
id: "iss-2608291957114882"
slug: "status-outdir-and-result-outdir-carry-an-absolute-path-into"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "v0.6.9-security-review"
found_at: "internal/core/site/build.go"
---

Status.OutDir and Result.OutDir carry an absolute path into abcd site --json and site build --json when --out is absolute; the iss-81 rule is that machine output never carries a developer-identity path and fsutil.RepoRel is the canonical primitive, unused here
