---
schema_version: 1
id: "iss-2609020154404742"
slug: "prioritise-and-openidset-internal-core-capture-workflow-go-b"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/workflow.go"
resolution: "The blocked_by projection now counts the whole of open/, not the part of it that parsed. openBlockingIDs takes one scan's issues AND its skipped roster and adds back the id each skipped file's name claims, read through issFileNumRe — the one detection grammar the scan, the resolver and record-lint share; List reaches it through openIDSet and Status through the scan it already has, so the two call sites cannot drift. A skipped record therefore goes on blocking its dependents, and the skip is reported in the same result, so the cause is visible rather than silent. TestASkippedOpenRecordStillBlocksItsDependents plants a blocker made unreadable in place (a body over the read cap, under its own name) and proves the dependent still reports it in blocked_by_open, in list and in the status board alike; watched failing first with an empty blocked_by_open."
impact: fix
---

prioritise and openIDSet (internal/core/capture/workflow.go) build the blocked_by projection from scanLedger's ISSUES alone and drop its Skipped roster, so an open record the guarded reader refused — a FIFO, a body over the read cap, a symlinked leaf — is invisible to the predicate. A dependent whose blocked_by names that record renders with an empty blocked_by_open and sorts ahead of genuinely unblocked work in both list and the status board, while its own blocked_by still names the blocker: the board understates the blocking, in the direction that invites work nobody can start. Both call sites are affected — List's openIDSet(ir) and Status's inline idSet(open). The fix must establish that a record skipped from open/ still counts as open for the projection, resolving the unreadable case toward still-blocking, with a test that plants an unreadable blocker and asserts the dependent still reports it blocking in list and in status.

## Grounds

- pursued: the unreadable case resolves toward still-blocking because understating progress is recoverable while inviting work whose blocker nobody can read is not, and recovering the id from the filename keeps the projection on the same grammar the scan used to decide the file was a record at all
