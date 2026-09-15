---
schema_version: 1
id: "iss-2609012043443066"
slug: "deferring-the-orphan-sweep-to-the-commit-path-lengthens-the"
severity: "minor"
category: "observation"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/ingest.go"
---

Deferring the orphan sweep to the commit path lengthens the window in which an orphan's reading records can be dispositioned before they are deleted. The sweep now runs only when a later ingest validates (iss-2608311517509690), so an orphaned run's reading records sit in the committed ledger for longer, and during that window capture's disposition verb can act on one of them. A later sweep then removes the item record by its id grammar and leaves the disposition record dangling, pointing at an item that no longer exists. The dangling case existed before the move — any orphan could be dispositioned between its crash and the next invocation — but the move widens it from the next invocation to the next one that validates. A future change could have the sweep refuse, or report, an orphan whose records already carry a disposition.
