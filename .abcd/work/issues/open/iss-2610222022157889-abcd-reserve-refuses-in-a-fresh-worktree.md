---
schema_version: 1
id: "iss-2610222022157889"
slug: "abcd-reserve-refuses-in-a-fresh-worktree"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "landing the go1.26.10 toolchain bump, 2026-10-22"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/copykey.go"
remedy: "When the checkout's .gitignore ignores .abcd/.work.local/ and the path is absent, create it (0700, no symlink followed, through the canonical fsutil writer) instead of refusing; keep the refusal for a path that exists as a non-directory or a link"
---

abcd reserve refuses in a fresh worktree: AGENTS.md sends a concurrent session to a plain git worktree add under the machine-scoped store and then to abcd reserve before any work, but a worktree just added has no .abcd/.work.local/ (the tier is gitignored and per-worktree), so tierPresent in internal/core/claim/copykey.go refuses with 'is not a directory in this checkout, so there is nowhere to keep this copy's key; run abcd ahoy to set it up' and nothing is pushed. Every new worktree hits it on its first reservation; on 2026-10-22 the go-1-26-10 worktree's reserve was refused this way, and a mkdir of the tier (made as a side effect of a preflight log directory) cleared it, with no ahoy run.

## Further cases, 2026-10-23

The hand-rolled coordination log records two more: abcd-f1 at 10:26 and again at 10:40 made a lane with a plain `git worktree add`, was refused by reserve for want of the local tier, and created the directory by hand before reserving. A second way to close the gap is at creation: the worktree store's add verb (itd-2609091014076309, step 2) lays `.abcd/.work.local/` when it makes the worktree. That covers the store's own lanes but not a plain `git worktree add`, which is what AGENTS.md tells a session to run today, so the remedy above stays the one that reaches every case.
