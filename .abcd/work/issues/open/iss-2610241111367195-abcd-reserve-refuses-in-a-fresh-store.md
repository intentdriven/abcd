---
schema_version: 1
id: "iss-2610241111367195"
slug: "abcd-reserve-refuses-in-a-fresh-store"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "setup redesign build, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/reserve.go"
remedy: "create the local tier (.abcd/.work.local/) on demand in reserve, as the transcript store creates itself, or have the documented worktree step create it; name the one command in the refusal rather than ahoy"
refines: [iss-2610222022157889]
---

abcd reserve refuses in a fresh store worktree: a worktree made with git worktree add under ~/.abcd.noindex/worktrees/<root-sha>/ has no .abcd/.work.local/, so reserve says 'nowhere to keep this copy's key; run abcd ahoy to set it up' and pushes nothing, although AGENTS.md tells every concurrent session to make exactly such a worktree and reserve before starting.
