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
resolution: "ArchiveTree asks each directory as dir/, the form git archive asks, and a test holds its listing to git archive HEAD over every export-ignore form."
impact: fix
resolved_by:
  commit: "6401b0be"
---

gitutil.ArchiveTree asks check-attr about each directory without a trailing slash, so the directory-only export-ignore form (dir/ export-ignore, at the root or in a nested .gitattributes) never matches and the non-plugin launch preview lists and scans files git archive HEAD omits; asking both dir and dir/ is also wrong, since archive asks a directory as dir/ alone and dir export-ignore followed by dir/ -export-ignore keeps it.

## Grounds

- pursued: ArchiveTree lists exactly the non-directory members git archive HEAD writes, directory-only export-ignore forms and dir/ re-inclusion included; a .gitattributes form under which the two listings differ would show it wrong
