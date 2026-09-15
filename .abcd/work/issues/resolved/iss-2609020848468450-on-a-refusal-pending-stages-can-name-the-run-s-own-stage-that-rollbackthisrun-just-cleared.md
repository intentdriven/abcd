---
schema_version: 1
id: "iss-2609020848468450"
slug: "on-a-refusal-pending-stages-can-name-the-run-s-own-stage-that-rollbackthisrun-just-cleared"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "ingest-help-fix-2026-09-02"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/ingest.go"
resolution: "rollbackThisRun now subtracts the stage it clears from the result's pending_stages, so a refusal's disclosure names only stages that still stand; pinned by the end-to-end refusal test, which now plants the run's own leftover stage."
impact: fix
resolved_by:
  commit: "6d0934a2"
---

On the refusal path the ingest result's pending_stages is set from the orphan list gathered before the refusal and is never recomputed after rollbackThisRun runs, so it can name the refused run's own stage as pending although that stage was just cleared. The disclosure the surface renders is then wrong by one entry in exactly the case it exists to describe. Found while pinning the ingest help's sentences (iss-2609020843067458). Fix: recompute or filter the pending list after the run's own rollback, and pin it with the same end-to-end refusal test that now proves the record set.

## Grounds

- pursued: the expectation is that pending_stages names only stages still standing after the invocation; the falsifier is a refusal whose own earlier attempt left a stage reporting that stage as pending after clearing it, which the end-to-end refusal test now asserts against.
