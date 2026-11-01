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
resolution: "indexDigest opens the index through an openIndex seam; TestAnIndexSwappedAfterItsLstatIsNotCopied swaps a symbolic link, a named pipe and a same-bytes regular file in between the Lstat and the open, reaching the ELOOP refusal and the post-open SameFile/IsRegular check, and asserts each yields a fresh uncopied value"
impact: internal
resolved_by:
  commit: "f71b34e390f0ab5129e7a83c477e0ab17167ba1f"
---

No test reaches indexDigest's ELOOP refusal or its SameFile check after the open (internal/core/implement/loop/autoplan.go, about lines 267 and 275): TestAnIndexTheDigestCannotCopyReadsAsChangedEveryTime reaches the Lstat branch and the copy bound, not the open's branches, because a symlink or a FIFO is refused before the open.

## Grounds

- pursued: we expect each swap to reach its open-time branch and yield an uncopied value; shown wrong if removing the ELOOP case, the SameFile check, O_NOFOLLOW or O_NONBLOCK leaves the test passing (each was tried and each made it fail)
