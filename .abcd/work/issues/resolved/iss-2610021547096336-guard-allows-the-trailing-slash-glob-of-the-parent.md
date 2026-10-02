---
schema_version: 1
id: "iss-2610021547096336"
slug: "guard-allows-the-trailing-slash-glob-of-the-parent"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25: sec-postTagA"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/defaults/guard.json"
remedy: "Add `../*/`, `$PWD/*/` and `${PWD}/*/` to rm-rf-working-directory's arg_values, as `*/` and `./*/` already are, so the trailing-slash glob of each directory the entry names warns as its `/*` form does; adding a value only adds a hit. Pin the spellings in a guard test watched failing first."
refines: [iss-2610021542183618]
resolution: "rm-rf-working-directory's arg_values name ../*/, $PWD/*/ and ${PWD}/*/ beside */ and ./*/, so the trailing-slash glob of the parent and of $PWD warns as its /* form does, and ../../*/ with it through the parent-chain reading; TestTheTrailingSlashGlobOfEachNamedDirectoryWarns, watched failing on all eight spellings first."
impact: fix
resolved_by:
  commit: "d2806e5dc"
---

The shell guard allows a recursive delete of the trailing-slash glob of the parent or of $PWD: `rm -rf ../*/`, `rm -rf $PWD/*/` and `rm -rf ${PWD}/*/` (and so `rm -rf ../../*/`) are an allow, while `rm -rf */` and `rm -rf ./*/` warn on rm-rf-working-directory. `../*/` globs every directory of the parent, the working directory among them, and `$PWD/*/` every directory of the working directory, as `./*/` does; the entry's arg_values name the trailing-slash glob for `*` and `./*` only. Confirmed at d94472abb while fixing iss-2610021542183618 (the parent-chain reading reaches `../*/` and finds no value for it).

## Grounds

- pursued: the trailing-slash glob of every directory the entry names warns; one that still allows, or a corpus line that moves toward allow, would show it wrong
