---
schema_version: 1
id: "iss-2609260927214634"
slug: "site-setup-default-branch-not-from-forge"
severity: "minor"
category: "drift"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review3-sitesetup note (f)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/launch/scaffold/scaffold.go"
resolution: "site setup reads the repository's default branch from the forge (repos/<o>/<r> default_branch) for the workflow, the environments' branch rule and the push step, and falls back to the checkout's branch only when the forge cannot answer, naming the fallback in the notes"
impact: fix
resolved_by:
  commit: "c03d1136"
---

site setup compares the environment's deployment branch rules against a default branch derived by scaffold.deriveBranch (origin/HEAD, else the checked-out HEAD, else main), not the forge's default branch, so on a checkout with no origin/HEAD run from a feature branch a correctly restricted environment is told to remove its rule for main. The advice is wrong but nothing is written; the spec's 'default-branch name read from the forge' does not describe the code.

## Grounds

- pursued: a checkout on a feature branch with no origin/HEAD gets environments restricted to the forge's default branch; a run whose forge answers yet restricts to the checkout's branch, or a fallback with no note, would show it wrong
