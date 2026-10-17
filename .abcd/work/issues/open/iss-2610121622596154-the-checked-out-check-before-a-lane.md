---
schema_version: 1
id: "iss-2610121622596154"
slug: "the-checked-out-check-before-a-lane"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "security review round 3 of fix/quota-deadline-and-folded-branch, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/land.go"
remedy: "after the full fold, map each rune to the smallest member of its unicode.SimpleFold orbit, then NFC (stdlib only; the reviewer verified it against every single-rune APFS equivalence class, 0 misses and 0 over-matches), with the U+13A0/U+AB70 pair added to branch_fold_test.go and watched failing first"
---

The checked-out check before a lane branch is deleted (sameBranchRef and refFold, internal/core/implement/loop/land.go) still misses one family of names APFS treats as one: Cherokee. golang.org/x/text's cases.Fold is not idempotent for Cherokee (it maps U+13A0 to U+AB70 and U+AB70 back to U+13A0, and U+13F0 with U+13F8), so a run state naming build/lane-<Cherokee small letter> passes the check while a worktree stands on build/lane-<Cherokee capital>, and update-ref -d deletes that branch from under it; reproduced end to end on APFS. The route exists without abcd (git branch -D with the other spelling does the same), so under abcd-fixes-what-it-adds this is help only, and lane branches abcd mints are ASCII. The person ruled on 2026-10-12 to ship the branch with this gap filed rather than take a fourth fix round.

## Route

The route exists without abcd: on APFS, `git branch -D` with the other Cherokee spelling deletes the branch from under a worktree in the same way. Under abcd-fixes-what-it-adds this is therefore help, not abcd's vulnerability; the person ruled on 2026-10-12 to ship the branch with the gap filed.
