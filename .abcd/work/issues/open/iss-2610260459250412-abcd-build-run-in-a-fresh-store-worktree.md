---
schema_version: 1
id: "iss-2610260459250412"
slug: "abcd-build-run-in-a-fresh-store-worktree"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "restarting build itd-82 after PR #1128"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop"
remedy: "When the checkout is a linked worktree whose main checkout has .abcd/.work.local/, make the worktree's own tier (the main checkout's tier proves the repository is managed), and otherwise have the refusal name the exact mkdir of the tier rather than abcd ahoy."
---

abcd build, run in a fresh store worktree of a checkout abcd manages, refuses at the state stage because .abcd/.work.local/ is not a directory there, and its remedy names abcd ahoy, which installs abcd rather than making the tier. AGENTS.md sends every concurrent session to its own store worktree made with a plain git worktree add, which never carries the gitignored tier, so every build or drain started that way is refused until the tier is made by hand with mkdir. Seen on 2026-10-26 restarting build itd-82 from ~/.abcd.noindex/worktrees/<root-sha>/itd-82-build.

The same refusal from abcd reserve is iss-2610222022157889: both come from the tier a plain git worktree add never makes, so one fix of where that tier is made should settle both.
