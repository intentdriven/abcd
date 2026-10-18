---
schema_version: 1
id: "iss-2610172133042562"
slug: "abcd-recognises-no-generated-file-in-a"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "reserving the v0.15.1 docs-drift records"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/generated.go"
remedy: "Judge the plain-declaration ASCII rule only on lines git reads: skip a line whose first non-blank byte is '#', as git's attribute parser does, so a non-ASCII comment no longer voids the declaration while a non-ASCII pattern or attribute still does"
resolution: "The plain-declaration ASCII rule now judges only the lines git reads: a blank line, or one whose first non-blank byte is '#', decides nothing, as git's attribute parser skips both whole. A byte-order mark still voids the declaration. TestACommentOutsideASCIIDecidesNothing fails on the old code with the reason abcd printed on its own repository. A review pair found no case where abcd says generated and git check-attr does not, across 5,775 and 4,032 differential cases."
impact: fix
resolved_by:
  commit: "09c4ad251"
---

abcd recognises no generated file in a repository whose root .gitattributes has a byte outside ASCII anywhere, even in a comment line git ignores. abcd's own file has em dashes in its comments, so on abcd itself every reserve reports 'not recognised: generated files (.gitattributes on origin/main holds a byte outside ASCII)' and the shared CLI reference and surface manifest are never let through as shared.

## Grounds

- pursued: a repository whose root .gitattributes has prose comments outside ASCII shares its declared generated files; a declaration git reads as unsetting a file that abcd then calls generated would show it wrong
