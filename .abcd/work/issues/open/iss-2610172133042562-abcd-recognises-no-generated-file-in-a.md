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
---

abcd recognises no generated file in a repository whose root .gitattributes has a byte outside ASCII anywhere, even in a comment line git ignores. abcd's own file has em dashes in its comments, so on abcd itself every reserve reports 'not recognised: generated files (.gitattributes on origin/main holds a byte outside ASCII)' and the shared CLI reference and surface manifest are never let through as shared.
