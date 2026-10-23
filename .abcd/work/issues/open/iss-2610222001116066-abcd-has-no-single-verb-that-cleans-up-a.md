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
