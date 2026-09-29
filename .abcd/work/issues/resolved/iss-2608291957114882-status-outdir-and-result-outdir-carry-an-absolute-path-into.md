---
schema_version: 1
id: "iss-2608291957114882"
slug: "status-outdir-and-result-outdir-carry-an-absolute-path-into"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "v0.6.9-security-review"
found_at: "internal/core/site/build.go"
resolution: "Status.OutDir, Result.OutDir and CheckResult.OutDir go through site.displayOutDir: repo-relative inside the repository (fsutil.RepoRel), home redacted to ~ outside it (fsutil.RedactHome), a relative --out as given. The lifeboat sweep is iss-2609261848326365; launch ship's payload.dest is captured as iss-2609261848338673."
impact: fix
resolved_by:
  commit: "2a05a0e1"
---

Status.OutDir and Result.OutDir carry an absolute path into abcd site --json and site build --json when --out is absolute; the iss-81 rule is that machine output never carries a developer-identity path and fsutil.RepoRel is the canonical primitive, unused here

## Grounds

- pursued: an absolute --out inside the repo reports as its repo-relative path and one under HOME as ~/…, on the board, the build and the check (TestTheSiteVerbsReportTheOutputDirectoryWithoutTheHomePath); an absolute path in any of the three would show it wrong
