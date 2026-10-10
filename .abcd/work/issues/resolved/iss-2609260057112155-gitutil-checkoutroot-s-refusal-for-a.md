---
schema_version: 1
id: "iss-2609260057112155"
slug: "gitutil-checkoutroot-s-refusal-for-a"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gitutil/repo.go"
resolution: "CheckoutRoot's refusal names a core.worktree setting naming a working tree that does not contain the directory as the fourth cause."
impact: fix
resolved_by:
  commit: "079daddf"
---

gitutil.CheckoutRoot's refusal for a repo-shaped tree git will not answer for names three causes (git absent from PATH, the repository unreadable, its ownership refused) and omits the fourth that Toplevel's shape check refuses: a core.worktree setting whose working tree does not contain the working directory. A person in that checkout is offered three causes, none of which is theirs.

## Grounds

- pursued: every cause Toplevel can refuse on is named; TestCheckoutRootRefusalNamesAWorktreeSettingPointingElsewhere drives real git with core.worktree set and would fail if the cause were dropped
