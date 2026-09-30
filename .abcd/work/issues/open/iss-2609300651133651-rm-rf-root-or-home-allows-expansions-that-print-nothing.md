---
schema_version: 1
id: "iss-2609300651133651"
slug: "rm-rf-root-or-home-allows-expansions-that-print-nothing"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/unknown.go"
remedy: "Read an empty default word as the empty text, and a subscript alone, a case change and an @ transform as the value or nothing; drop the empty text wherever ifsSplits reads a site, since it splits into no field. Grounds: bash 3.2, /bin/sh and bash 5.3 print / for ${X:-}/, ${X-}/, ${A[0]}/ and ${A[@]}/ with X and A unset, and bash 5.3 for ${X^}/ and ${X@P}/ with X empty."
---

rm-rf-root-or-home allows expansions that print nothing beside a root: `rm -rf ${X:-}/`, `${X-}/`, `$HOME${X:-}`, `${A[0]}/`, `${A[@]}/`, and bash 5 `${X^}/` and `${X@P}/` hand rm / or the home, while `${1:-}/` blocks. spellWord read an empty word as no text, and the subscript, case and @ operators returned the value alone.
