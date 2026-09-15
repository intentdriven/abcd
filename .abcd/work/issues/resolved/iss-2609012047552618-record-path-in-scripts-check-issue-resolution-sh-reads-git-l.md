---
schema_version: 1
id: "iss-2609012047552618"
slug: "record-path-in-scripts-check-issue-resolution-sh-reads-git-l"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/check-issue-resolution.sh"
resolution: "check-issue-resolution.sh now runs every git call through a wrapper that sets core.quotePath=false, so a record whose slug carries a non-ASCII byte lists as its real path: record_path finds its status folder, frontmatter_commit's git show reaches it, and the stale-branch diagnosis holds for it. The sibling calls (both ls-tree listings, diff --name-status, diff --name-only) shared the defect and are covered by the same wrapper. A case with the slug iss-998-é.md in the stale-branch topology failed against the previous script and passes now; every earlier case stays green."
impact: fix
---

record_path in scripts/check-issue-resolution.sh reads git ls-tree --name-only, which C-quotes a path holding a non-ASCII byte (a record such as iss-999-é.md lists as "…/iss-999-\303\251.md", quotes included), so status_of yields a quoted path whose first component is a quoted-string prefix rather than open, resolved or wontfix, and the RS001 diagnosis added for the stale-branch shape silently regresses to the generic 'does not enter' text for exactly the records whose slug carries an accent. Found by the ruthless review of the hygiene branch. The fix is to list with -c core.quotePath=false (or -z) and to add a case with a non-ASCII slug to check-issue-resolution-cases.sh.

## Grounds

- pursued: one script-wide setting is safer than a flag on each call because the next path-listing call added to the script inherits it; if a case with a quoted path ever regresses, the wrapper was bypassed with command git
