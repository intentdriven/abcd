---
schema_version: 1
id: "iss-2609292320015665"
slug: "rm-rf-root-or-home-reads-a-trimmed-expansion-as-its-variable"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/unknown.go"
remedy: "In spellParameterAt, read a trim's pattern by its shape (readPattern): a suffix trim whose pattern can take any length (a *, unknown text or an extglob group) and whose first element past any run of * is a glob or unknown text ($, ${, $(, a backtick), and a prefix trim whose last such element is, spell as the variable, / and nothing; keep a trim whose anchored element is literal text, or whose pattern matches a fixed width, as the variable alone, and pin ${DIR%/}, ${f%.txt}, ${p##*/}, ${p%/*}, ${p#$HOME/} and ${X%?} as allowed, test first; verified on /bin/bash 3.2, /bin/sh and /bin/dash."
---

rm-rf-root-or-home reads a trimmed expansion as its variable alone, but a suffix trim whose pattern is unknown text or begins with a glob can leave only the leading slash of an absolute path: with X=/a/b, bash 3.2 prints / for ${X%${X#?}} and ${X%%[!/]*}, so rm -rf ${X%${X#?}} deletes the root and allows. Found while fixing iss-2609290426544292, whose written spelling now holds a set of texts; a trim can add / to that set, but reading every trim as the root would refuse the everyday ${DIR%/} and ${f%.*}, so the rule needs the pattern's shape.
