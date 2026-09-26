---
schema_version: 1
id: "iss-2609260149247384"
slug: "gitutil-archivetree-passes-its-revision-to-git-ls-tree-as-a"
severity: "nitpick"
category: "security"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25 (fix round, review of lane launchkind)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gitutil/archive.go"
---

gitutil.ArchiveTree passes its revision to git ls-tree as a positional argument without --end-of-options, so an option-shaped revision is parsed as a flag; the sole caller passes HEAD today.
