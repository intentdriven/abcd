---
schema_version: 1
id: "iss-2609012043432648"
slug: "the-status-render-lists-the-ingest-stage-through-a-plain-pat"
severity: "minor"
category: "observation"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/status.go"
---

The status render lists the ingest stage through a plain path rather than through os.Root. Describe (internal/core/reading/status.go) calls os.ReadDir on repoRoot joined with IngestStageDir, so a hostile clone that force-adds .abcd/.work.local as a symlink pointing elsewhere can echo directory names that match the run-id grammar into the status render's orphaned_ingests (and, the same way, staged_runs — the pre-existing StagedRuns read has the same shape). Read-only: nothing is written or deleted through this path, and the write and delete side of the verb (the sweep, rollbackRun) is Root-contained and skips symlinks. Recorded so the read side is known to sit outside the containment the write side has.
