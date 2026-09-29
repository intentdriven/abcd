---
schema_version: 1
id: "iss-2609290925333487"
slug: "the-shell-guard-reads-a-dot-glob-path-segment-as-a-name"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25: verify-fix4-guardResid"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/match.go"
resolution: "dotParents reads a non-final segment written with a leading . whose glob can match .. (.?, .*, .[.], .[!x], ..*) as .., and the folded reading is compared beside the target as written, so ~/.?/*, ~/.*/*, ~/../.*/* and /.?/* block. A glob whose leading . is not written (??, ?., [.]?) and a final dot-glob (~/.?) are compared as written; on bash 5.3, which leaves the segment unexpanded, the block is a stated over-block. TestDotGlobSegmentsThatCanMatchParent pins both sides."
impact: fix
resolved_by:
  commit: "4ef5013bd"
---

The shell guard reads a dot-glob path segment as a name, although bash 3.2 and /bin/sh (no globskipdots) let a segment written with a leading dot match the name .. and expand ~/.?/*, ~/.[.]/*, ~/.[!x]/*, ~/.?*/* and ~/.*/* to include ~/../* (the home's parent's entries, the home among them) and /.?/* and ~/../.*/* to the root's entries; every one is allowed. bash 5.3 leaves them literal. echo alone showed the expansions.

## Grounds

- pursued: a target whose dot-glob segment, read as .., reaches the root or the home blocks, as bash 3.2 and /bin/sh expand it; an echo on either shell showing such a glob reaching the root or the home under a spelling that still allows would show it wrong
