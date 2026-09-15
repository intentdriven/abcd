---
schema_version: 1
id: "iss-2608301251398360"
slug: "a-flow-context-explicit-key-leaks-an-excluded-field-and-want"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "itd-183-round-9-ruthless"
found_at: "internal/core/reading/project.go"
resolution: "unresolvableFrontmatterShape gains flowExplicitKeyRe, which reports a question mark following a brace or a comma in a flow context as 'an explicit key in a flow mapping', beside nestedMappingRe for the block-sequence half. Both are refused whatever the key is named, which is the one fix this record and iss-2608301237450573's first half ask for: the floor stops recognising a nested key by its spelling and refuses the nesting."
impact: fix
resolved_by:
  intent: "itd-194"
  spec: "spc-2609021003136831"
---

a flow-context explicit key leaks an excluded field and wants the same fix as the compact block-sequence mapping

Found by the round-9 adversarial RUTHLESS review of build/itd-183.
PRE-EXISTING on the branch -- leaks on HEAD and on the parent alike, and named
in no residue list or record before now.

`meta: {? origin : <value>}` and `meta: {? "origin" : <value>}` are a valid
YAML `origin` key. `questionLineRe` is anchored to line start; `flowKeyRe`
requires `[{,]` immediately before the name; `excludedKeyLineRe` reads only
`meta`. So the key travels.

THE POINT OF THIS RECORD, in the reviewer's words: this is the same class as
the compact block-sequence mapping (`items:` / `  - origin: <value>`) already
recorded in iss-2608301237450573, and **the two want one fix, not two**. Both
are a real YAML key that the line-anchored and delimiter-anchored patterns
cannot see because it is nested. Fixing them one at a time is how this floor
has spent nine rounds.

Per the facilitator's ruling of 2026-08-30, this is seed material for the
exclusion-floor intent rather than another round on this branch: a spelling
arms race needs a design, not more patterns.

## Grounds

- pursued: we expect refusing the construction rather than learning the key's spelling to end the arms race this record names, because the refusal does not depend on which name is hidden; an explicit flow key still assembling, or an ordinary flow sequence being refused, would show it wrong
