---
schema_version: 1
id: "iss-2608241542245317"
slug: "capture-followup-comment-verb"
severity: "minor"
category: "future-work-seed"
source: "agent-finding"
found_during: "full-repo"
found_at: "internal/core/capture/workflow.go"
deferred_after: "v0.11.1"
deferral_reason: "planning F owed to the product thinker: capture has no followup or comment verb and no --relates flag (its sub-verbs are admit, defer, disposition, link, list, mentions, migrate, promote, reframe, resolve, surprise and wontfix). related_issues has gained writers in promote and migrate, but a free relation writer and appended follow-up sections are a new verb with no planned intent. (re-checked at e792a2314 by lane drainDQ3, run A, 2026-09-29)"
---

capture has no way to record comments or updates on an existing issue -- issues are create-once records whose only later mutations are status transitions, so triage evidence, corrections and new reproductions accumulate in conversation or unsanctioned hand-edits instead of the ledger; add a capture followup verb appending timestamped redacted body sections under the ledger lock, and give the schema's unwritten related_issues field its writer via a --relates flag (same schema-grows-a-writer precedent as resolved_by); design sketch persisted at .abcd/.work.local/scratch/review-2026-08-24/design-capture-followup.md review 2026-08-24