---
schema_version: 1
id: "iss-2609301913458174"
slug: "concurrent-audit-ingests-file-duplicate-owed-check-issues"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/auditowed.go"
remedy: "Hold the ledger lock (withLedgerLock) across the open-carrier scan and the Capture through a Capture-under-held-lock seam, so the link-never-double check and the filing are one step; collapse several open carriers of one receipt to the oldest (the others declined as its duplicates with the ledger's wontfix --duplicates closure); a passing re-run resolves every open carrier. Grounds: the reviewer's four-goroutine probe filed four issues 6/6 runs at 8f49c2e2e."
---

Concurrent audit ingests file duplicate owed-check issues. Two or more ingests of one failed after-merge audit verdict each ran fileAuditOwed's open-issue scan (internal/core/capture/auditowed.go) outside the ledger lock, saw no carrier, and each filed an issue: four goroutines filed four, none linked. An ingest whose intent write failed after its filing left the same shape, an open carrier no flag names, and a passing re-run resolved only the flagged one, leaving the rest open.
