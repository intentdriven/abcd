---
schema_version: 1
id: "iss-2609300726507446"
slug: "structural-closure-of-iss-2609300651115290-s-class-an-ifs"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/payload.go"
remedy: "Read the IFS of a line as unknown (cap its unquoted default, alternative, trim and replacement words and unquoted HOME/PWD) when any layer's text holds the name IFS or a name built from an expansion, read lexically over the raw text whatever the context: an expansion touching a name byte or another expansion, an expansion an assignment operator follows, or an expansion standing as the operand of a builtin that assigns the names it is handed. Every reported form is one of these shapes whatever context holds it, so the rule is their superset; it replaces the per-context readings (namingCommands words, markedAssignment, arithmeticNamesIFS). Grounds: the bash manual (Shell Arithmetic evaluates a variable's value as an expression; subscripts, $[ ], (( )), for (( )), [[ -eq ]], substring offsets and integer attributes are arithmetic) and the shells' printf output."
---

Structural closure of iss-2609300651115290's class: an IFS assigned through a name built from an expansion still passed the guard in contexts the per-context reading never reached. I=I; : $[${I}FS=1], a[${I}FS=1]=x, : ${a[${I}FS=1]}, declare -i n; n=${I}FS=1, (( ${I}FS++ )), : ${X:${I}FS=1} and x=${I}FS=1; : $((x)) each set IFS in bash 3.2, /bin/sh and bash 5.3, which then hand rm "" and / for rm -rf ${U:-1/1}; all were allowed. Rounds 1 and 3 each listed the contexts that can set IFS and each re-verify found another, so the fix is one fail-closed rule, not another context.
