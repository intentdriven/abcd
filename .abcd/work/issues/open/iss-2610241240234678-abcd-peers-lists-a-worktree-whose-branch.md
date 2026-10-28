---
schema_version: 1
id: "iss-2610241240234678"
slug: "abcd-peers-lists-a-worktree-whose-branch"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "branch sweep survey, 2026-10-24"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/peers"
remedy: "classify a branch with no commits of its own as not-started rather than merged, and report merged only for a branch that carried commits which origin/main now contains"
refines: [iss-2610222001116066]
---

abcd peers lists a worktree whose branch has zero commits past an older main as merged (fix/external-review-events, created 70 minutes earlier by a live session and not yet started): an unstarted lane reads as spent work, which invites any sweep that trusts the listing to delete a peer's fresh worktree.
