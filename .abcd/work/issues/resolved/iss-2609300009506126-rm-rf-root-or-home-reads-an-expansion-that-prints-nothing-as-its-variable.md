---
schema_version: 1
id: "iss-2609300009506126"
slug: "rm-rf-root-or-home-reads-an-expansion-that-prints-nothing-as-its-variable"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/unknown.go"
remedy: "In spellParameterAt, add the empty text to a longest trim (%% or ##) whose pattern can match a whole absolute path (it can take any length, begins with *, a glob, unknown text or a literal /, and ends with *, a glob or unknown text), to a trim whose anchored end is a glob or unknown text, to a substring, and to a trim, replacement or substring after a subscript (bash 3.2 prints nothing there for a scalar); verified on /bin/bash 3.2, /bin/sh and /bin/dash; pin ${X%%*} alone, ${p%/*} and ${X%%.*}/ as allowed, test first."
resolution: "a longest trim whose pattern can match the whole path, a substring, and a trim, replacement or substring after a subscript now also spell as nothing, so rm -rf ${X%%*}/ and ${X[0]%zzz}/ block while the expansion alone stays allowed"
impact: fix
resolved_by:
  commit: "d82049fb7e8be1e429a8eb91297119aa171e9901"
---

rm-rf-root-or-home reads an expansion that prints nothing whatever the value is as its variable, so the text around it is never read as the whole word: with X=/a/b, bash 3.2, /bin/sh and dash print nothing for ${X%%*}, ${X##*}, ${X%%/*} and ${X:0:0}, so rm -rf ${X%%*}/ deletes the root and rm -rf $HOME/${X%%*} the home, and both allow. bash 3.2, the /bin/bash and /bin/sh of macOS, also prints nothing for a trim, a replacement or a substring after a scalar's subscript (${X[0]%zzz}/ is /). Found while fixing iss-2609292320015665, the trim that leaves only the root; an alternative's empty text was added by iss-2609290426544292, and these are its siblings.

## Grounds

- pursued: TestExpansionsThatPrintNothingTheWrittenCompareReads blocks each form bash 3.2 and /bin/sh print as the root or the home and allows the expansion standing alone; a structurally empty expansion whose neighbour text still reads only as the variable would show it wrong
