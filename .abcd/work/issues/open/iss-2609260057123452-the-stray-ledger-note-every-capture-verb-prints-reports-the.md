---
schema_version: 1
id: "iss-2609260057123452"
slug: "the-stray-ledger-note-every-capture-verb-prints-reports-the"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli.go"
---

The stray-ledger note every capture verb prints reports the checkout's own ledger as a second ledger when the working directory is a case-variant spelling of the checkout on a case-insensitive filesystem (the APFS default): strayStoreNotes compares EvalSymlinks strings, which keep the caller's case, so the walk never meets the root and names ../<root>/.abcd/work/issues as a ledger below the root.
