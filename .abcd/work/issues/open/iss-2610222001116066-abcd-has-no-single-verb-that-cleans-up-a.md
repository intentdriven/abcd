---
schema_version: 1
id: "iss-2610222001116066"
slug: "abcd-has-no-single-verb-that-cleans-up-a"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "PT asked for it during the 2026-10-22 crash recovery"
origin: researcher-authored
production_mode: hand-written
found_at: "commands"
remedy: "add abcd sweep: a dry run lists every worktree, local and remote branch, reservation and inbox report with what it would do to each (retire merged and clean worktrees and branches, delete expired and released reservations, promote inbox reports into the ledger); the run then acts on the listed items, and leaves dirty worktrees, unpushed or unmerged work and anything another session holds for a person, named"
---

abcd has no single verb that cleans up a repository: after the 2026-10-22 crash this checkout held 189 worktrees (most on merged branches, some dirty), 86 reservation branches on origin (66 expired, 20 released), unmerged local branches, and inbox reports not yet pulled into the ledger, each needing its own manual pass. The person wants an abcd sweep verb that does everything a good cleanup needs: worktrees, branches, reservations, pulling in the inbox items, and so on.

## Evidence: hand-run survey and sweep, 2026-10-24

A survey of this checkout at 14:30 CEST, with 8 peer sessions live (4 busy), sorted every ref into "remove now", "keep" or "a person decides". It used origin/main ancestry, patch-equivalence (`git cherry`), the forge's PR state, worktree dirtiness, and the cwd of every live process.

- **Remote branches (18, excluding reservations):** 7 removed. Each one's tip was an ancestor of origin/main and had no open PR. Four came from PRs merged with `delete_branch_on_merge: true` set (#1055, #1071, #1072, #1073), so the forge setting alone does not clear them. Two had no PR because their content landed through a stacked PR (`feat/announce-step4`, `feat/drain-steps5-7`). One belonged to a closed PR whose content landed elsewhere (#1064). The deleted tips can be recovered by SHA: aaacdcad5, 59c8b54b0, 9e76b238f, 380ae0e9f, 005b38128, 84aec70af, b15e2e48a. 11 kept: 6 with open PRs (#982, #996, #1010, #1105, #1110, #1116), `abcd-status/reppl`, the queue ref for #1120, `capture/iss-2610240827243654` (queued), `design/autonomy-grant` (live work), and origin/main. Three open PRs are stale enough for a person to rule on: #982 (since 2026-10-18), #996 (2026-10-14) and #1010 (177 commits ahead).
- **Reservation branches (132):** 40 expired, 87 released and 5 live. None were deleted by hand: the reservation-sweep workflow (#1111) owns them and has not run yet. Its secrets `RESERVATION_SWEEP_APP_ID`/`RESERVATION_SWEEP_APP_KEY` did not appear in `gh secret list` from this account, so the first run may be the dry run.
- **Local branches (24) and worktrees (24):** none removed. Every one was either ahead of origin/main with no PR (that day's unlanded work), dirty, the cwd of a live process, or the embargo store (9 untracked private reports). The only clean candidates were `fix/external-review-events`, at the old main with zero commits and no process in its worktree, and `fix/embargo-guards`, whose worktree is dirty. The first was created 70 minutes earlier and could not be told apart from a live session's lane that had not started. Nothing links a worktree to the session that made it (iss-2608220750029993). `abcd peers` lists both as `merged`, which is wrong: iss-2610241240234678. Local `main` sits 35 behind origin/main, and the main checkout holds uncommitted changes and is the cwd of 12 processes.

What the survey shows a sweep needs:
1. Forge merge state and patch-equivalence, not just ancestry.
2. Live-session ownership of each worktree.
3. A third category, "a person decides", for stale open PRs.
4. Deference to the reservation workflow instead of acting itself.
5. Enforcement in the guard, so a hand-run delete cannot skip these checks. One run here did skip them: a check-then-act loop called `abcd guard check` with the wrong syntax, ignored the nonzero result and deleted anyway. The guard would have allowed the deletes regardless, because it has no rule for them (iss-2610241239492159). Under `go run`, exit codes collapse to 1 (iss-2610241240133009).
