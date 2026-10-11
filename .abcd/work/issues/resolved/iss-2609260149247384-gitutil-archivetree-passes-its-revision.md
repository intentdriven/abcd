---
schema_version: 1
id: "iss-2609260149247384"
slug: "gitutil-archivetree-passes-its-revision"
severity: "nitpick"
category: "security"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25 (fix round, review of lane launchkind)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gitutil/archive.go"
resolution: "ArchiveTree puts --end-of-options before the positional revision."
impact: internal
resolved_by:
  commit: "af2d21a0"
---

gitutil.ArchiveTree passes its revision to git ls-tree as a positional argument without --end-of-options, so an option-shaped revision is parsed as a flag; the sole caller passes HEAD today.

## Grounds

- pursued: an option-shaped revision reaches git ls-tree as an object name and is refused as one; git printing ls-tree usage for such a revision would show it wrong
