---
schema_version: 1
id: "iss-2609290518278152"
slug: "fsutil-readdeclaration-refuses-a-home"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/fsutil/fsutil.go"
resolution: "fsutil.ReadDeclaration and ReadGuardedInRoot re-vet a file renamed into place between the vetting lstat and the open, up to 8 times, instead of refusing it on sight: a same-owner owner-only-writable regular file (a concurrent WriteFileAtomic) is read, anything a guard refuses is refused by that guard, and a replacement that never settles is still refused as a swap."
impact: fix
resolved_by:
  commit: "b342b2b29"
---

fsutil.ReadDeclaration refuses a home-scoped declaration as replaced between its vetting and its read when another abcd process of the same user atomically renames a new version into place inside the lstat-to-open window, so two abcd processes on one machine (two sessions, or a hook and a verb) can make one refuse its own ~/.abcd/config.json, rules.json or credentials index; the merge-group macOS leg of PR 744 failed TestConcurrentConnectsKeepEveryKeyAndBlock on it.

## Grounds

- pursued: two abcd processes on one machine no longer make one refuse its own ~/.abcd/config.json when the other rewrites it; TestReadDeclarationReadsABenignReplacementAfterRevetting or TestConcurrentConnectsKeepEveryKeyAndBlock failing with ErrDeclarationSwapped would show it wrong, and TestReadDeclarationRefusesANonBenignReplacement reading a symlink, FIFO, directory, group-writable or foreign-owned replacement would show the fix weakened the guard
