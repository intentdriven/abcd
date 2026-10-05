---
schema_version: 1
id: "iss-2610050728100598"
slug: "the-rename-repair-loop-misses-worktrees-named-with-a-slash"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "another account's rename to ~/.abcd.noindex, read from its saved Terminal output on 2026-10-05"
origin: researcher-authored
production_mode: hand-written
found_at: "hooks/hooks.json"
remedy: "Replace the repair loop everywhere it is printed or documented (hooks/hooks.json, hooks/bootstrap.sh, docs/how-to/install.md, docs/how-to/upgrade-to-v0.13.0.md) with one that finds each worktree by its .git file at any depth, for example: find ~/.abcd.noindex/worktrees -name .git -type f | while read -r g; do git -C \"${g%/.git}\" worktree repair; done, and say that 'repair: gitdir incorrect' lines are repairs, not failures; better, give it a verb that walks the store, repairs each worktree and reports any it could not."
resolution: "The printed repair is now one find that stops at each worktree's .git file at any depth and runs git worktree repair there, carried byte for byte by the stop lines, the hooks' wrapper, bootstrap.sh, the install and upgrade pages and the brief, and saying that a 'repair: gitdir incorrect' line is a link fixed; TestPrintedRepairReachesWorktreesAtAnyDepth runs it through sh against a moved store holding two-level, slash-named three-level and legacy one-level worktrees and asserts every back-link names the new place with no error line. abcd ahoy now reports, read-only, each store worktree still unlinked as store.worktree_unlinked with that worktree's own repair line."
impact: fix
resolved_by:
  commit: "5da884ff6"
---

The rename remedy abcd prints and documents, mv ~/.abcd ~/.abcd.noindex && for w in ~/.abcd.noindex/worktrees/*/*; do git -C "$w" worktree repair; done, assumes every worktree sits exactly two levels down, at worktrees/<root-sha>/<name>. A worktree named after its branch, such as docs/capture-sibling-repo-owner-redaction, sits three levels down, so the loop runs git in the docs/ folder above it, fails with 'not a git repository', and never repairs it. One left over at one level, with no root-sha key, gets git run on each of its files ('cannot change to …/AGENTS.md: Not a directory'). Seen on 2026-10-05 in a second account on this machine: eleven 'not a git repository' lines and five 'Not a directory' lines, and afterwards worktrees/488a0aa9…/docs/capture-sibling-repo-owner-redaction still had its main checkout pointing back at the old ~/.abcd path, so git treats it as missing and a git worktree prune would drop its entry. The 'repair: gitdir incorrect' lines in the same output are successful repairs that read like errors. The store there also held empty branch-prefix folders left by removed worktrees (check/, chore/, fix/ …) and a 117 KB test log at its top level.

## Grounds

- pursued: a person who renames the folder and runs the printed repair finds every store worktree, at any depth, linked back from its repository, and abcd ahoy reports none unlinked; a worktree still listed prunable after the repair, or one ahoy fails to name, would show it wrong
