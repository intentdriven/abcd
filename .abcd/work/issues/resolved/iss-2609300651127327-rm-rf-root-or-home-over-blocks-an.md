---
schema_version: 1
id: "iss-2609300651127327"
slug: "rm-rf-root-or-home-over-blocks-an"
severity: "major"
category: "bug"
source: "review-followup"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/unknown.go"
remedy: "Read a colon default, assignment or error message (${1:-w}, ${1:=w}, ${1:?}) without the empty text an emptyable parameter adds, and keep it under the colonless forms, where a set but empty parameter prints its empty value. Grounds: bash 3.2, /bin/sh, dash and bash 5.3 print dist/ for ${1:-dist}/ with no argument or an empty one."
resolution: "spellParameterAt reads the colon default, assignment and error message from the value without the empty text an emptyable parameter adds, and keeps it under the colonless forms; TestColonDefaultsOfAnEmptyParameterTheWrittenCompareReads."
impact: fix
resolved_by:
  commit: "8b6ddeb18"
---

rm-rf-root-or-home over-blocks an everyday clean line: `rm -rf "${1:-build}"/*`, `rm -rf ${1:-dist}/`, `rm -rf "${1:-dist}/"*` and `rm -rf ${@:-x}/` block, and `rm -rf ./${1:-dist}` warns, though no shell prints the empty value under `:-`. The empty text an emptyable parameter can print was kept under the colon operators.

## Grounds

- pursued: rm -rf "${1:-build}"/*, ${1:-dist}/, "${1:-dist}/"* and ${@:-x}/ allow again and ./${1:-dist} no longer warns, while ${1:-/}, ${1-}/ and ${1-dist}/ block; a shell that prints the empty value under :- would show it wrong, and bash 3.2, /bin/sh, dash and bash 5.3 do not.
