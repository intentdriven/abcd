---
schema_version: 1
id: "iss-2610021542183618"
slug: "guard-allows-a-recursive-delete-of-a-parent-chain"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25: sec-postTagA"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/defaults/guard.json"
remedy: "Read a target whose leading run of `..` segments is two or more as that run written once (`../..` as `..`, `../../` as `../`, `../../*` as `../*`), beside the readings argValueMatches already compares, so the change only adds a hit; pin every parent-chain spelling (trailing `/` or `/.`, quoted, -r/-R/--recursive) to warn, and prove monotonicity by feeding one corpus file to base and head binaries. Grounds: rm deletes the operand's whole tree, and `../..` contains `..`."
---

The shell guard allows a recursive delete of a chain of parents: `rm -rf ../..`, `rm -rf ../../.`, `rm -rf ../../..` and `rm -rf ../../*` are an allow, while `rm -rf ..`, `rm -rf ../.` and `rm -rf ../*` warn on rm-rf-working-directory. The entry's arg_values name `.`, `..`, `../` and `../*` but no deeper chain, and foldParents reads `../../.` as `../..`, which no value names, so TestTrailingDotAfterAParentFolds pins only that the two spellings agree. A chain climbs above the working directory's parent and holds it, so it deletes at least what `..` deletes. Found by the post-tag security review of 7fb52a6b5..d94472abb (sec-postTagA, LOW); present at both ends of that range.
