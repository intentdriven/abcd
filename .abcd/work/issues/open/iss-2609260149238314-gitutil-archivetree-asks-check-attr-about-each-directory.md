---
schema_version: 1
id: "iss-2609260149238314"
slug: "gitutil-archivetree-asks-check-attr-about-each-directory"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25 (fix round, review of lane launchkind)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gitutil/archive.go"
---

gitutil.ArchiveTree asks check-attr about each directory without a trailing slash, so the directory-only export-ignore form (dir/ export-ignore, at the root or in a nested .gitattributes) never matches and the non-plugin launch preview lists and scans files git archive HEAD omits; asking both dir and dir/ is also wrong, since archive asks a directory as dir/ alone and dir export-ignore followed by dir/ -export-ignore keeps it.
