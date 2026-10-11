---
schema_version: 1
id: "iss-2609260933592838"
slug: "archive-tree-reads-attributes-from-the"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review2-launchkind side note"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gitutil/archive.go"
resolution: "ArchiveTree reads the archived revision's own .gitattributes: --cached when the index holds exactly the revision's attribute files (any git), --source=<rev> when they differ (git 2.40+), and on an older git in that state a named refusal with the remedy rather than the index's answer. TestArchiveTreeReadsAttributesFromTheRevision holds the listing to git archive HEAD with a staged attributes change in both directions; TestArchiveTreeRefusesWhenItCannotReadTheRevisionsAttributes pins the old-git refusal."
impact: fix
resolved_by:
  commit: "680834da8"
---

gitutil.ArchiveTree reads export attributes with `git check-attr --cached` (the index) while `git archive` reads them from the archived tree, so the two disagree when a .gitattributes change is staged but not committed: the launch listing can include or omit a path the released archive does the opposite with. `check-attr --source=<rev>` (git 2.40 or later) reads the same tree git archive does.

## Grounds

- pursued: the launch listing equals git archive's tree whatever is staged; a staged .gitattributes change that moves a file in the listing but not in the archive would show it wrong
