---
schema_version: 1
id: "iss-2609261205185463"
slug: "scribe-ingest-lists-a-run-s-reading-items-through-a"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review2-scribe"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/scribe/ingest.go"
---

scribe ingest lists a run's reading items through a directory whose own link status it never judges: runItems in internal/core/scribe/ingest.go refuses a symlinked .abcd, .abcd/work, issues root or status directory through capture.RefuseRedirectedLedger, but a symlinked .abcd/work/issues/readings or run directory is followed by os.ReadDir, so the listing is drawn from outside the ledger (probed by review2-scribe: nil) where scribe assemble refuses the same link. The readings directory and the run directory are to be Lstat-refused by the primitive capture's own ledger walk uses.
