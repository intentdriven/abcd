---
schema_version: 1
id: "iss-2609290925346181"
slug: "rm-rf-pwd-is-allowed-while-rm-rf-and-rm"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25: verify-fix4-guardResid"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/match.go"
resolution: "foldParents folds a $PWD, ${PWD} or relative beginning: a .. past it is the working directory's parent, so $PWD/../*, $PWD/.., ./../* and x/../../* warn as ../* does and $PWD/x/../* as $PWD/*. A relative path whose .. stays inside the working directory (x/../*) is compared as written, a residual named in 17-guard.md. TestWorkingDirectoryParentThroughPWD pins both sides."
impact: fix
resolved_by:
  commit: "4ef5013bd"
---

rm -rf $PWD/../* is allowed while rm -rf ../* and rm -rf $PWD/* warn (rm-rf-working-directory): the same parent directory, spelled through $PWD, is not folded, since the lexical fold of .. reads only targets that begin at the root or the home. ./../* and x/../../* (the parent too) are allowed the same way.

## Grounds

- pursued: a target reaching the working directory's parent through $PWD or a relative .. warns as ../* does; such a spelling that allows would show it wrong
