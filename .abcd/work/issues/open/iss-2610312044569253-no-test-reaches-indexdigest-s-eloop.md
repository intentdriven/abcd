---
schema_version: 1
id: "iss-2610312044569253"
slug: "no-test-reaches-indexdigest-s-eloop"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "landing PR #1301 (run-2610310228179894, lane-1)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/autoplan.go"
remedy: "Add a test that drives each branch through a seam (an injectable open, or a swap between the Lstat and the open) and asserts each yields an uncopied value, watched fail first."
---

No test reaches indexDigest's ELOOP refusal or its SameFile check after the open (internal/core/implement/loop/autoplan.go, about lines 267 and 275): TestAnIndexTheDigestCannotCopyReadsAsChangedEveryTime reaches only the Lstat branch, because a symlink or a FIFO is refused before the open.
