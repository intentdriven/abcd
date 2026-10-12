---
schema_version: 1
id: "iss-2610121209284195"
slug: "the-check-that-refuses-to-delete-a-lane"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "security review of fix/discard-keeps-checked-out-branch (PR #969), 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/land.go"
remedy: "compare the listed branch ref with the lane's case-insensitively on a filesystem that folds case (fsutil.CaseFoldingFS), with a test that provokes both answers through a seam"
resolution: "branchCheckedOutElsewhere compares the listed branch ref with the lane's through sameBranchRef, through full Unicode case folding and NFC (refFold), as APFS resolves names, where the filesystem folds them, so a checkout on build/lane-1 counts as standing on a lane named build/Lane-1"
impact: fix
---

The check that refuses to delete a lane branch another checkout stands on (branchCheckedOutElsewhere, internal/core/implement/loop/land.go, used by the landing and both discards) compares branch names byte for byte. On a filesystem that folds case, macOS by default, git resolves a loose ref whatever case it is named in, so a run state naming build/Lane-1 reads and deletes the branch build/lane-1 with update-ref -d, while a worktree on build/lane-1 is not seen as standing on build/Lane-1 and its HEAD is left naming nothing. The same route exists without abcd: plain git branch -D build/Lane-1 deletes build/lane-1 from under a worktree, so under abcd-fixes-what-it-adds this is help only, offered because it changes nothing about how a repository works.

## Grounds

- pursued: on a folding filesystem a discard or landing of a mis-cased lane branch refuses while another checkout stands on it; a deletion from under that checkout shows it wrong

## Route

The route exists without abcd: on a case-folding filesystem, plain `git branch -D build/Lane-1` deletes `build/lane-1` from under the worktree that has it checked out (verified by the security review in a scratch repository, git 2.52.0). Under abcd-fixes-what-it-adds it is therefore not abcd's vulnerability; this is help, given because the comparison changes nothing about how the repository works.
