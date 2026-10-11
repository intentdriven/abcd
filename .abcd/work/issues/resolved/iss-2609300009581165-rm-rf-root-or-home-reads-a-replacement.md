---
schema_version: 1
id: "iss-2609300009581165"
slug: "rm-rf-root-or-home-reads-a-replacement"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/unknown.go"
remedy: "In spellParameterAt, read a replacement's pattern by the same shape as a trim's: where it can take any length, begins with *, a glob, unknown text or a literal / and ends with *, a glob or unknown text, add its string's texts; where it also begins (past its leading *) with a glob or unknown text and is not anchored with /#, add / before each; keep ${X/foo/$HOME}, ${DIR/#\\~/$HOME} and ${name//[^a-z]/} allowed; verified on /bin/bash 3.2 and /bin/sh (dash has no replacement), test first."
resolution: "replacementTexts reads a replacement's pattern by the trim's shape rule: one that can match the whole path adds its string, one that can match all of it after the leading slash adds / and its string, so rm -rf ${X/?*/$HOME} and ${X/${X#?}} block while ${X/foo/$HOME} and ${DIR/#\\~/$HOME} stay allowed"
impact: fix
resolved_by:
  commit: "d82049fb7e8be1e429a8eb91297119aa171e9901"
---

rm-rf-root-or-home reads a pattern replacement as its variable unless its pattern is only *, but a replacement whose pattern can match the whole of an absolute path prints its string in its place, and one whose pattern can match all of the path after its leading slash prints / and its string: with X=/a/b, bash 3.2 and /bin/sh print the home for ${X/\/*/$HOME}, ${X/?*/$HOME} and ${X/$X/~}, and / for ${X/${X#?}} and ${X//[!\/]*/}, and rm -rf of each allows. Found while fixing iss-2609292320015665; iss-2609290426544292 read only the *-only pattern and left the rest alone to keep ${DIR/#\~/$HOME} allowed.

## Grounds

- pursued: TestReplacementsThatCanTakeTheWholeValueTheWrittenCompareReads blocks each form bash 3.2 and /bin/sh print as the root or the home and keeps the everyday replacements allowed; a replacement whose whole-value pattern still reads as the variable alone would show it wrong
