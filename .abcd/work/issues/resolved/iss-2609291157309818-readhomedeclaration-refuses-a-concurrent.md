---
schema_version: 1
id: "iss-2609291157309818"
slug: "readhomedeclaration-refuses-a-concurrent"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/fsutil/home.go"
resolution: "fsutil.readDeclarationIn, the descriptor read behind ReadHomeDeclaration and ReadHomeDeclarationDenying, re-judges a leaf renamed into place between its vetting Lstat and its open from scratch, up to 8 times, as ReadDeclaration does: a same-owner regular file every guard and the reader's deny mask admit is read, anything a guard refuses is refused by that guard, and a replacement that never settles is still refused as a swap."
impact: fix
resolved_by:
  commit: "ecd41fdf5"
---

fsutil.ReadHomeDeclaration still refuses a home-scoped declaration as replaced between its vetting and its read when a concurrent abcd renames a new version into place inside the lstat-to-open window: the re-vetting fix for iss-2609290518278152 (b342b2b29) went into the path-based ReadDeclaration, and the integration merge that landed it (24e78506b) kept the descriptor-based readDeclarationIn from a07ad672f, which every home-scoped reader (layered.Load, oracle, credential, rules, statusline) goes through and which refuses a replacement on sight; so iss-2609290518278152 regressed and TestConcurrentConnectsKeepEveryKeyAndBlock failed the Linux CI leg of PR 748 with ErrDeclarationSwapped from ~/.abcd/config.json

## Grounds

- pursued: two abcd processes on one machine no longer make one refuse its own ~/.abcd/config.json, rules.json or credential store when the other rewrites it; TestReadHomeDeclarationReadsABenignReplacementAfterRevetting or TestConcurrentConnectsKeepEveryKeyAndBlock failing with ErrDeclarationSwapped would show it wrong, and TestReadHomeDeclarationRefusesANonBenignReplacement reading a symlink, FIFO, directory, group-writable, foreign-owned or deny-mode replacement would show the fix weakened the guard
