---
schema_version: 1
id: "iss-2609300651115290"
slug: "rm-rf-root-or-home-is-bypassed-by-an-ifs-named-through-an"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/payload.go"
remedy: "Count as naming IFS any declaration, read, mapfile, getopts or let word, any printf -v or wait -p target, and any assignment-shaped word whose name holds an expansion's mark, plus an arithmetic expression that names IFS or assigns through an expansion; fail closed. Grounds: bash 3.2, /bin/sh, dash and bash 5.3 split ${U:-x/x} into an empty field and / after each form."
---

rm-rf-root-or-home is bypassed by an IFS named through an expansion: with I=I, `export ${I}FS=x; rm -rf ${U:-x/x}` allows and hands rm "" and / on bash 3.2, /bin/sh, dash and bash 5.3, and so do `eval "I${F:-F}S=x"`, `declare|typeset|readonly|local ${I}FS=x`, `read -r ${I}FS`, `printf -v ${I}FS x` and `: $((IFS=1))`. namesIFS matched the literal text IFS only, and the tokenizer steps over arithmetic.
