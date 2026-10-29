---
schema_version: 1
id: "iss-2610290730196083"
slug: "private-lab-sessions-and-ideate-verdicts"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "product thinker, 2026-10-29"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lab/lab.go"
remedy: "Add a lifeboat-style pair of verbs for private and unfinished work: one packs, for this repository, the private labs and ideate verdicts plus every local branch and worktree not on origin (commits, uncommitted diffs, untracked files, local-tier notes) into a single hashed, redacted bundle with a generated manifest of each branch's state and next step, readable only by the receiving account; the other, run by that account, verifies it, restores the private stores with owner-only modes and the branches as store worktrees, refuses the whole write on any conflict, and names what did not travel (receipts, run state, reservations), so nothing reaches the repository or the forge on the way."
---

Private and unfinished work cannot move between accounts or computers. Private lab sessions and ideate verdicts live only in the running account's own owner-only stores (~/.abcd.noindex/lab/<root-sha>/ and ~/.abcd.noindex/ideas/<root-sha>/). Unfinished work lives in local branches and store worktrees: commits not yet on origin, uncommitted diffs, untracked files and each worktree's local-tier notes. A person who moves to a second account or computer, or hands work to a colleague's account, can carry none of it over except by hand: a git bundle of the branches, a patch and a tarball per dirty worktree, and a hand-written status table and handover note saying which branch is which and what comes next. That route skips redaction, integrity checks and the owner-only modes the private stores rely on. It also says nothing about what cannot travel: preflight receipts, implement-loop run state and reservations held by the old checkout. On the receiving side nothing ingests the material, so later lab, ideate or build runs there cannot read it, and it does not stay private by default.
