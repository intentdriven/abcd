---
schema_version: 1
id: "iss-2610252214130450"
slug: "the-folder-kind-table-and-text-say-a-git"
severity: "nitpick"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S01"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md"
remedy: "Say 'a .git entry (directory or file)' at 01-ahoy.md:558-573, or point at detect.go."
---

The folder-kind table and text say a '.git/ directory' marks a repository, but gitPresent accepts a .git file too (linked worktree, submodule; internal/core/ahoy/detect.go:220-229).
