---
schema_version: 1
id: "iss-2609261016494611"
slug: "the-armed-receipt-gate-is-satisfied-by-a-forged-promote"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-lintA item 1"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/lint.go"
resolution: "checkReceiptGate reads every receipt and the release-gate manifest through fsutil.ReadGuardedInRoot on an os.Root opened at the repository root, so an ancestor link that leaves the tree is an error at the read itself and the leaf is still refused as a link; the commit directory is judged once before the gates and a link carrying it out of the tree is one fail-closed receipt_gate finding naming it. TestReceiptGateRefusesUnsafeReceipts covers a symlinked commit directory holding a forged PROMOTE and a symlinked manifest directory whose manifest a forged receipt echoes, both watched yielding zero findings before the change."
impact: fix
resolved_by:
  commit: "22d5de32"
---

The armed receipt gate is satisfied by a forged PROMOTE reached through a symlinked commit directory. checkReceiptGate (internal/core/lint/lint.go) reads each receipt with fsutil.ReadGuarded on the unresolved path `.abcd/work/reviews/<sha>/<gate>.json`, and holds only receipts_dir inside the repository once links are followed. O_NOFOLLOW refuses a symlinked leaf, but the kernel follows every ancestor, so a committed `.abcd/work/reviews/<sha>` that is a link to a directory outside the tree, holding a well-formed PROMOTE receipt, yields zero receipt_gate findings: the release gate passes on a receipt the tree does not hold. The release-gate manifest read has the same shape: a symlinked `.abcd/development/release-gate` directory hands the gate an out-of-tree manifest whose hash a forged receipt can echo. iss-2609012037127981 was resolved with grounds that the gate is never satisfied by a receipt reached through a link, so that resolution over-claims.

## Grounds

- pursued: an armed receipt gate is never satisfied by a receipt or manifest reached through a link at any component of its path; a forged PROMOTE under a symlinked `reviews/<sha>` or `release-gate` directory yielding zero receipt_gate findings would show it wrong
