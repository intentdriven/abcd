---
schema_version: 1
id: "iss-2609300057304812"
slug: "the-guard-allows-rm-rf-x-x-and-x-x2f-which-bash-3-2-bin-sh"
severity: "major"
category: "security"
source: "impl-review"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/unknown.go"
remedy: "spellWord decodes an ANSI-C string with readAnsiCQuote, reads a locale string as the double-quoted string it holds, spells $! as its number or nothing, and reads any other $ that opens no expansion as the literal $ bash prints; grounds: bash 3.2, /bin/sh and bash 5.3 print / for each form under printf, and dash prints $/ (never the root)."
---

The guard allows rm -rf ${X:-$'/'}, ${X:-$"/"} and ${X:-$'\x2f'}, which bash 3.2, /bin/sh and bash 5.3 all print as / with X unset: spellWord reads a $ that opens no name as a word it cannot read and returns nil, so a default or alternative word written as an ANSI-C or a locale string is never read (review-guardSet MAJOR-1). The same reading misses ${X:-$!/}, which is / with no background job.
