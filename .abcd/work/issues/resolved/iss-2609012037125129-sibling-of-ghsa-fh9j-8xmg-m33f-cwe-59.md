---
schema_version: 1
id: "iss-2609012037125129"
slug: "sibling-of-ghsa-fh9j-8xmg-m33f-cwe-59"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/glossary/index.go"
resolution: "Resolved on its merits: fixed at base by 746a5d2c (shipped in v0.8.0). ScanInRoot reads every term file through fsutil.ReadGuardedInRoot with the term-family cap maxTermBytes (index.go:188), so a FIFO, a symlinked leaf, an out-of-root ancestor link and an oversized file are all refused; the cap the record left open was chosen there. The record's requested test in the term store was missing and is added in ef2b4ae4a (guarded_read_test.go: FIFO within a deadline, symlinked term, one byte past the cap), each killed on a scratch copy with the pre-advisory os.ReadFile restored."
impact: fix
shipped_in: v0.8.0
resolved_by:
  commit: "746a5d2c2"
---

Sibling of GHSA-fh9j-8xmg-m33f (CWE-59, CWE-400), found on the sweep and not fixed there: glossary readTerm (internal/core/glossary/index.go) loads every term file the index walk finds through a bare os.ReadFile — no O_NOFOLLOW, no O_NONBLOCK, no regular-file check, no byte cap. The glossary store is committed and travels with a clone, so a committed FIFO at a term name hangs every verb that builds the index, and a committed symlink reads an out-of-tree file as a term. The fix is the same one-line routing through fsutil.ReadGuarded that the issue and reading families now use, plus a byte cap chosen for the term family (the glossary has no cap constant of its own yet, which is the one decision that kept this out of the advisory fix) and a test with a FIFO and a symlinked leaf in the term store. Already-captured siblings are iss-2608301203521317 (lint scanRecordStores) and iss-2608211914592726 (the residual lint sweep).

## Grounds

- pursued: a glossary scan never blocks on, follows, or over-reads a committed term file; a FIFO, link or oversized term the scan reads would show it wrong
