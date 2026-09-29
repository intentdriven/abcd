---
schema_version: 1
id: "iss-2609290925346181"
slug: "rm-rf-pwd-is-allowed-while-rm-rf-and-rm-rf-pwd-warn"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25: verify-fix4-guardResid"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/match.go"
---

rm -rf $PWD/../* is allowed while rm -rf ../* and rm -rf $PWD/* warn (rm-rf-working-directory): the same parent directory, spelled through $PWD, is not folded, since the lexical fold of .. reads only targets that begin at the root or the home. ./../* and x/../../* (the parent too) are allowed the same way.
