---
schema_version: 1
id: "iss-2609012037127981"
slug: "sibling-of-ghsa-fh9j-8xmg-m33f-cwe-59-cwe-400-found-on-the-s"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/lint.go"
resolution: "checkReceiptGate reads each receipt and the release-gate manifest with fsutil.ReadGuarded under a 4 MiB cap on the unresolved path, so a symlinked leaf is refused outright, and an unreadable receipt is a fail-closed finding rather than an aborted crawl; receipts_dir is contained. TestReceiptGateRefusesUnsafeReceipts covers a FIFO receipt, a receipt symlinked to an out-of-tree PROMOTE, and a FIFO manifest. (An equivalent fix, 40bab2fe, sat on the unmerged fix/security-sweep-continued branch and never reached main.) Amended 2026-09-26: this fix guarded the leaf alone, so it met its grounds only for a symlinked receipt file. A symlinked commit directory (`.abcd/work/reviews/<sha>` linked out of the tree) or a symlinked release-gate directory still carried either read to an out-of-tree regular file it accepted, and a forged PROMOTE reached that way satisfied the gate; iss-2609261016494611 closes that half by reading both through an os.Root at the repository root."
impact: fix
resolved_by:
  commit: "b48fd584"
---

Sibling of GHSA-fh9j-8xmg-m33f (CWE-59, CWE-400), found on the sweep and not fixed there: checkReceiptGate (internal/core/lint/lint.go) reads each review receipt under .abcd/work/reviews/<sha>/<gate>.json and the release-gate manifest through a bare os.ReadFile — no O_NOFOLLOW, no O_NONBLOCK, no regular-file check, no byte cap. The reviews store is committed and travels with a clone, so a committed FIFO at a receipt path hangs abcd lint (and make preflight through it), and a committed symlink is judged as if it were a receipt. The fix is the routing through fsutil.ReadGuarded that lint's readingoutstanding.go already uses for the reading family, with a cap for the receipt shape, and a test with a FIFO and a symlinked receipt. Distinct from iss-2609011423385217 (the manifest's stale pinned inputs) and from the record-store sweep records iss-2608301203521317 and iss-2608211914592726, which do not name the reviews store.

## Grounds

- pursued: an armed receipt gate never hangs and is never satisfied by a receipt reached through a link; a symlinked PROMOTE receipt yielding zero receipt_gate findings would show it wrong
