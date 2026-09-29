---
schema_version: 1
id: "iss-2609240646538011"
slug: "forge-ignores-merge-union-so-records-prs-go-dirty"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: ".gitattributes"
resolution: ".gitattributes and the one-writer-per-file principle say the union merge driver holds for a local merge only and that the forge conflicts regardless. The remedy the record asks to weigh, one file per decision, is already accepted as adr-2609151138420062 (delivery in itd-2609151138388536); this cost only strengthens it, so nothing further is weighed here."
impact: internal
resolved_by:
  commit: "edaf963a3"
---

The `merge=union` driver that .gitattributes gives `.abcd/work/DECISIONS.md` and `CHANGELOG.md` (the remedy iss-118 adopted) holds for a local merge only; the forge does not apply it. A pull request's mergeability and the merge queue's merge are computed on the forge, so two open pull requests that each append a DECISIONS.md entry conflict there as soon as one merges, while a local `git merge` of the same two is clean. In autonomous run A every records pull request went DIRTY whenever another records pull request merged first (#678, #679 and #681 on 2026-09-23), and each was recovered by a local merge in the lane's worktree, a push to the same branch behind a ten-to-twenty-minute pre-push preflight, and a re-armed auto-merge. Nothing tells an author that the union driver stops at the local clone. Wanted: the .gitattributes comment and the conventions that cite the driver say it holds locally only, and the remedy that needs no driver (one file per decision, as iss-2609100507439414 and iss-2608220150157511 propose) is weighed with this cost counted, since it removes the conflict on the forge as well.

## Grounds

- pursued: an author reading either convention learns the driver stops at the local clone; a convention citing merge=union as the remedy without that limit would show it wrong
