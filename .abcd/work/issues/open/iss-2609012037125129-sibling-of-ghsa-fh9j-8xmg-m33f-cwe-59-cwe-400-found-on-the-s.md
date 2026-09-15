---
schema_version: 1
id: "iss-2609012037125129"
slug: "sibling-of-ghsa-fh9j-8xmg-m33f-cwe-59-cwe-400-found-on-the-s"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/glossary/index.go"
---

Sibling of GHSA-fh9j-8xmg-m33f (CWE-59, CWE-400), found on the sweep and not fixed there: glossary readTerm (internal/core/glossary/index.go) loads every term file the index walk finds through a bare os.ReadFile — no O_NOFOLLOW, no O_NONBLOCK, no regular-file check, no byte cap. The glossary store is committed and travels with a clone, so a committed FIFO at a term name hangs every verb that builds the index, and a committed symlink reads an out-of-tree file as a term. The fix is the same one-line routing through fsutil.ReadGuarded that the issue and reading families now use, plus a byte cap chosen for the term family (the glossary has no cap constant of its own yet, which is the one decision that kept this out of the advisory fix) and a test with a FIFO and a symlinked leaf in the term store. Already-captured siblings are iss-2608301203521317 (lint scanRecordStores) and iss-2608211914592726 (the residual lint sweep).
