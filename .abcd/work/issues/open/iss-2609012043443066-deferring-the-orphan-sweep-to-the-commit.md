---
schema_version: 1
id: "iss-2609012043443066"
slug: "deferring-the-orphan-sweep-to-the-commit"
severity: "minor"
category: "observation"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/ingest.go"
remedy: "Waits on ruling BY2: (a) have capture's Disposition in `internal/core/capture/reading.go` refuse an item whose run has no run.json commit marker, closing the window at the writer; (b) have sweepOrphanStages in `internal/core/reading/ingest.go` refuse to roll back a run while a disposition names one of its items; (c) keep the delete and report each orphaned disposition by id in the ingest result; each proven by a test with a crashed run (a stage and no run.json) whose item carries a disposition."
deferred_after: "v0.11.1"
deferral_reason: "ruling F owed to the product thinker: still real. sweepOrphanStages in internal/core/reading/ingest.go runs only once a later ingest validates, and rollbackRun removes an orphan's item records without asking whether a disposition names them, so a disposition written in that window is left dangling. The choices are the record's (the sweep refuses, or reports, such an orphan) plus one this lane found: capture disposition could refuse an item whose run has no commit marker (run.json), closing the window at the writer. Each changes a shipped verb's behaviour. (re-checked at e792a2314 by lane drainDQ3, run A, 2026-09-29)"
---

Deferring the orphan sweep to the commit path lengthens the window in which an orphan's reading records can be dispositioned before they are deleted. The sweep now runs only when a later ingest validates (iss-2608311517509690), so an orphaned run's reading records sit in the committed ledger for longer, and during that window capture's disposition verb can act on one of them. A later sweep then removes the item record by its id grammar and leaves the disposition record dangling, pointing at an item that no longer exists. The dangling case existed before the move — any orphan could be dispositioned between its crash and the next invocation — but the move widens it from the next invocation to the next one that validates. A future change could have the sweep refuse, or report, an orphan whose records already carry a disposition.

## Remedy grounds (2026-09-29)

- rollbackRun at the base removes an orphan's item records by the id grammar under the ledger lock and never reads dispositions, so the dangling case is real; option (a) is the lane-found writer-side close the deferral names.
- No outside-practice check: the choice is among behaviours of a shipped internal verb.
- Rejected: widening the sweep back to every invocation, which reopens the refused-run delete the commit-path move closed.
