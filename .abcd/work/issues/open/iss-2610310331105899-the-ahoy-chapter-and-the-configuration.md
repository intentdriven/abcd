---
schema_version: 1
id: "iss-2610310331105899"
slug: "the-ahoy-chapter-and-the-configuration"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.1 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md"
remedy: "Restate the worktree store in 04-surfaces/01-ahoy.md and 05-internals/03-configuration.md as shipped, citing ahoy worktree add, list and prune, and correct the matching lines in AGENTS.md and the CONCURRENCY domain of .abcd/rules.json."
---

The ahoy chapter and the configuration chapter still describe the worktree store's verbs, reclaim and notes archive as not built, though ahoy worktree add, list and prune ship and prune writes the archive.
