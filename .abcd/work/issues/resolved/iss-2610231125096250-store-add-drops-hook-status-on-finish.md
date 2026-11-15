---
schema_version: 1
id: "iss-2610231125096250"
slug: "store-add-drops-hook-status-on-finish"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/worktree/worktree.go"
remedy: "Carry the hook's exit status into the refusal on the finishing-failure path too, naming both failures."
related_intents: [itd-2609091014076309]
resolution: "Fixed in this change: when the checkout hook fails and finishing the worktree then fails too, Add's refusal names the hook's failure first, then the finishing failure, and keeps git's failure as its cause (internal/core/ahoy/worktree/worktree.go:369; TestAddNamesTheHookFailureWhenTheFinishFailsToo)."
impact: fix
resolved_by:
  spec: "spc-2609301811532881"
---

When the worktree store's add finishes a worktree git made after a failing checkout hook, and the finishing itself fails (the start record or the local tier cannot be written), the refusal drops the hook's exit status, so a person sees only the finishing failure and not that the hook failed first. Found by the Opus review of the store's step 2 on 2026-10-23.
