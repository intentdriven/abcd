---
schema_version: 1
id: "iss-2609260933592838"
slug: "archive-tree-reads-attributes-from-the-index"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review2-launchkind side note"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gitutil/archive.go"
---

gitutil.ArchiveTree reads export attributes with `git check-attr --cached` (the index) while `git archive` reads them from the archived tree, so the two disagree when a .gitattributes change is staged but not committed: the launch listing can include or omit a path the released archive does the opposite with. `check-attr --source=<rev>` (git 2.40 or later) reads the same tree git archive does.
