---
schema_version: 1
id: "iss-2609261950077063"
slug: "launch-archive-json-reports-the-archive-it-wrote-as-an"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-drainSite"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/launch/archive.go"
resolution: "PluginArchive keeps the absolute working Path (json:\"-\"), which the archive verb removes a refused archive through, and reports DisplayPath under the unchanged key path through fsutil.DisplayPath: relative to the repository for the release workflow's --out bin, the home redacted to ~ otherwise. The text written line prints the display field."
impact: fix
resolved_by:
  commit: "ae58e9260"
---

launch archive --json reports the archive it wrote as an absolute path: PluginArchive.Path (internal/core/launch/archive.go) is the --out directory made absolute by the front door and joined with the archive name, so archive.path names the developer's home whenever --out sits under it, against the iss-81 rule; the text report's written line prints the same value. Found in the drainPaths sweep of path-bearing --json fields.

## Grounds

- pursued: an archive written under HOME reports archive.path as ~/dist/<name> and one written inside the repository as bin/<name>, with the archive still readable where it was written (TestTheRenderAndTheArchiveReportTheirPathsWithoutTheHome, TestAnArchiveInsideTheRepositoryIsReportedRelativeToIt); an absolute archive.path would show it wrong
