---
schema_version: 1
id: "iss-2609261954288630"
slug: "launch-dry-run-json-and-launch-ship-json"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-drainSite"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/launch/bundle.go"
resolution: "IncludedFile keeps the absolute ResolvedPath (json:\"-\") that the render, the gates, the scan and the parity diff read every payload file through, and reports DisplayResolvedPath under the unchanged key resolved_path, relative to the repository, set once where the resolver emits each included file."
impact: fix
resolved_by:
  commit: "ae58e9260"
---

launch --dry-run --json and launch ship --json name every payload file absolutely: IncludedFile.ResolvedPath (internal/core/launch/bundle.go) is the file's absolute on-disk path and is tagged resolved_path, so the bundle's files list in the dry-run report and in a ship's payload.bundle carries the checkout's absolute path once per file, naming the developer's home whenever the checkout sits under it, against the iss-81 rule. Found in the drainPaths sweep by running the read-only --json verbs from a checkout under the home.

## Grounds

- pursued: every bundle file of a render from a checkout under HOME reports a repository-relative resolved_path and the render's JSON carries neither spelling of the home (TestTheRenderAndTheArchiveReportTheirPathsWithoutTheHome); an absolute resolved_path in launch --dry-run --json or in a ship's payload.bundle would show it wrong
