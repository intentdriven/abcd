---
schema_version: 1
id: "iss-2609261036363663"
slug: "scribe-assemble-refuses-a-symlink-at-an-allow-list-directory"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-scribe"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/scribe/assemble.go"
resolution: "scribe assemble and the ingest's item listing judge every directory above the allow list through capture.RefuseRedirectedLedger, the ledgerDirs read form every capture verb applies, and refuse a symlinked ancestor"
impact: internal
resolved_by:
  commit: "c74a932d"
---

scribe assemble refuses a symlink at an allow-list directory and inside it but not at its ancestors (.abcd, .abcd/work, .abcd/work/issues): os.Root follows an in-root link, so a committed issues -> docs/shadow link carries shipped-tree content into the scribe context under ledger paths

## Grounds

- pursued: a committed symlink at .abcd/work or .abcd/work/issues refuses the assembly and the ingest before any ledger content is read; a shadowed shipped file reaching the context under a ledger path would show it wrong
