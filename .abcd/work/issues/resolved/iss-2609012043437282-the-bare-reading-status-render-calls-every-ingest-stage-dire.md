---
schema_version: 1
id: "iss-2609012043437282"
slug: "the-bare-reading-status-render-calls-every-ingest-stage-dire"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/status.go"
resolution: "Describe now probes each stage's commit marker (ReadingsRecordDir/<id>/run.json) — the same probe the sweep's rollbackRun makes — and reports the two states apart: orphaned_ingests for a run with no marker, whose records the next validating ingest rolls back, and the new leftover_stages for a run that committed and whose stage merely failed to clear, whose records stay. The CLI render prints a separate leftover-stage line that says the run committed and only the stage goes, and commands/reading.md tells a host to report both keys and never to promise a rollback for a leftover. TestTheBareRenderTellsALeftoverStageFromAnOrphan (core) and TestTheStatusRenderTellsALeftoverStageFromAnOrphan (cli) pin it; both were watched failing to build before the field existed."
impact: fix
---

The bare reading status render calls every ingest stage directory an orphan, and promises a rollback that a committed run will never get. Describe (internal/core/reading/status.go) lists IngestStageDir and reports every well-formed run-id directory under orphaned_ingests, described as a run with no commit marker whose reading records the next validating ingest will roll back. But a stage also survives when the commit path's RemoveAll fails AFTER run.json has landed (the Degraded branch at the end of write in ingest.go): that run is complete, and the sweep's rollbackRun already tells the two apart by probing ReadingsRecordDir/<id>/run.json and leaving a committed run's records alone. So for that case the CLI interrupted line (renderReadingStatus in internal/surface/cli/reading.go) and the orphaned_ingests paragraph on commands/reading.md state something false: the records are not for a run that never happened and will not be rolled back; only the stage will be cleared. Both reviewers found it independently. The fix must make Describe probe run.json and report the two cases apart — an orphan whose records will be rolled back, versus a committed run whose leftover stage will merely be cleared — and make the CLI render and the plugin page say the right thing for each.

## Grounds

- pursued: the sweep already drew this line by probing run.json, so the render draws the same line with the same probe rather than inventing a second rule; a new key rather than a re-labelled one keeps orphaned_ingests meaning what the page says it means
