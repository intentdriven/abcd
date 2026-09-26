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
resolution: "strayStoreNotes recognises the checkout root by file identity rather than spelling, so a case-variant working directory no longer reports the root's own store as a stray; genuine strays are still named."
impact: fix
resolved_by:
  commit: "67594f9b"
---

The stray-ledger note every capture verb prints reports the checkout's own ledger as a second ledger when the working directory is a case-variant spelling of the checkout on a case-insensitive filesystem (the APFS default): strayStoreNotes compares EvalSymlinks strings, which keep the caller's case, so the walk never meets the root and names ../<root>/.abcd/work/issues as a ledger below the root.

## Grounds

- pursued: one checkout has one ledger whatever case the caller types; TestStrayStoreNotesKnowsTheRootUnderAnotherCase would fail on a case-insensitive filesystem if the root's own store were reported again or a real stray were missed (it skips on a case-sensitive one)
