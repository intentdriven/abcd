---
schema_version: 1
id: "iss-2610021542183618"
slug: "guard-allows-a-recursive-delete-of-a"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25: sec-postTagA"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/defaults/guard.json"
remedy: "Read a target whose leading run of `..` segments is two or more as that run written once (`../..` as `..`, `../../` as `../`, `../../*` as `../*`), beside the readings argValueMatches already compares, so the change only adds a hit; pin every parent-chain spelling (trailing `/` or `/.`, quoted, -r/-R/--recursive) to warn, and prove monotonicity by feeding one corpus file to base and head binaries. Grounds: rm deletes the operand's whole tree, and `../..` contains `..`."
resolution: "argValueMatches also reads a target with its leading run of two or more .. segments written once (parentRun), beside the readings it already compares, so rm -rf ../.., ../../., ../../.., ../../* and their quoted, trailing-slash, $PWD-led and folded spellings warn on rm-rf-working-directory as .. and ../* do; TestAParentChainReadsAsOneParent, watched failing on 37 spellings first. A 1094-line corpus fed to base d94472abb and head binaries changes 304 lines, 295 allow to warn and 9 warn to warn with the entry made precise, and none toward allow."
impact: fix
resolved_by:
  commit: "c1d13ba08"
---

The shell guard allows a recursive delete of a chain of parents: `rm -rf ../..`, `rm -rf ../../.`, `rm -rf ../../..` and `rm -rf ../../*` are an allow, while `rm -rf ..`, `rm -rf ../.` and `rm -rf ../*` warn on rm-rf-working-directory. The entry's arg_values name `.`, `..`, `../` and `../*` but no deeper chain, and foldParents reads `../../.` as `../..`, which no value names, so TestTrailingDotAfterAParentFolds pins only that the two spellings agree. A chain climbs above the working directory's parent and holds it, so it deletes at least what `..` deletes. Found by the post-tag security review of 7fb52a6b5..d94472abb (sec-postTagA, LOW); present at both ends of that range.

## Grounds

- pursued: every pure parent chain reaches the verdict its one-parent form does and no verdict weakens; a chain spelling that still allows, or a corpus line that moves toward allow, would show it wrong
