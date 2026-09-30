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
resolution: "namesIFS counts a declaration, read, mapfile, getopts or let word, a printf -v or wait -p target, and an assignment-shaped word whose assigned name holds an expansion's mark, and the tokenizer raises segment.arithmeticAssigns for an arithmetic body that names IFS or assigns through an expansion; TestIFSNamedThroughAMarkTheWrittenCompareReads."
impact: fix
resolved_by:
  commit: "3c68b005c"
---

rm-rf-root-or-home is bypassed by an IFS named through an expansion: with I=I, `export ${I}FS=x; rm -rf ${U:-x/x}` allows and hands rm "" and / on bash 3.2, /bin/sh, dash and bash 5.3, and so do `eval "I${F:-F}S=x"`, `declare|typeset|readonly|local ${I}FS=x`, `read -r ${I}FS`, `printf -v ${I}FS x` and `: $((IFS=1))`. namesIFS matched the literal text IFS only, and the tokenizer steps over arithmetic.

## Grounds

- pursued: every form the review named, and the arithmetic, let, getopts, mapfile and wait -p siblings, now blocks while IFS= read -r f and IFS=, read -ra arr with a quoted operand stay allowed; a name built some way the guard still does not read (a sourced file, a nameref set before the line) would show it wrong, and 17-guard.md names those as residuals.
