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
---

The shell guard reads a dot-glob path segment as a name, although bash 3.2 and /bin/sh (no globskipdots) let a segment written with a leading dot match the name .. and expand ~/.?/*, ~/.[.]/*, ~/.[!x]/*, ~/.?*/* and ~/.*/* to include ~/../* (the home's parent's entries, the home among them) and /.?/* and ~/../.*/* to the root's entries; every one is allowed. bash 5.3 leaves them literal. echo alone showed the expansions.
