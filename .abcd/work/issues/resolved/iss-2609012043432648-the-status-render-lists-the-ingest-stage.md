---
schema_version: 1
id: "iss-2609012043432648"
slug: "the-status-render-lists-the-ingest-stage"
severity: "minor"
category: "observation"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/status.go"
resolution: "The bare reading render lists the assembly parking area and the ingest stage through the one os.Root it already opened, via readDirIn, so a local tier, stage or parking area linked out of the checkout refuses the render instead of echoing names from outside it; TestTheBareRenderListsTheLocalTierThroughTheOneRoot covers all three links."
impact: fix
resolved_by:
  commit: "4d31718ca"
---

The status render lists the ingest stage through a plain path rather than through os.Root. Describe (internal/core/reading/status.go) calls os.ReadDir on repoRoot joined with IngestStageDir, so a hostile clone that force-adds .abcd/.work.local as a symlink pointing elsewhere can echo directory names that match the run-id grammar into the status render's orphaned_ingests (and, the same way, staged_runs — the pre-existing StagedRuns read has the same shape). Read-only: nothing is written or deleted through this path, and the write and delete side of the verb (the sweep, rollbackRun) is Root-contained and skips symlinks. Recorded so the read side is known to sit outside the containment the write side has.

## Grounds

- pursued: a status render whose listings go through the repository root cannot report a run-id-shaped name that lives outside the checkout; a symlinked local tier that still yields entries in staged_runs or orphaned_ingests would show it wrong
