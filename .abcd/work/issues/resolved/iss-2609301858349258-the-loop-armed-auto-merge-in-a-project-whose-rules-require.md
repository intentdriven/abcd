---
schema_version: 1
id: "iss-2609301858349258"
slug: "the-loop-armed-auto-merge-in-a-project-whose-rules-require"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
remedy: "Arm only when an active ruleset on the default branch, read from the same mirror at the lane's base, requires a person's approval (a pull_request rule with required_approving_review_count >= 1, or require_code_owner_review with a CODEOWNERS file naming an owner); otherwise leave the pull request open for a person, say so in the step note, the run record and implement status, and never arm it later. Grounds: the product thinker's ruling AM1 (2026-09-30), and the forge's own semantics that a code-owner review applies only where CODEOWNERS names an owner."
resolution: "The arm step arms auto-merge only where an active ruleset on the default branch requires a person's approval (approving count >= 1, or a code-owner review with a CODEOWNERS owner); otherwise the pull request is left open for a person and never armed later (ruling AM1)."
impact: fix
resolved_by:
  commit: "22eeb1f72"
---

The loop armed auto-merge in a project whose rules require no approval: the implement loop's arm step (internal/core/implement/loop/land.go landArm) armed 'gh pr merge --auto' wherever the ruleset mirror named a merge queue, without reading whether any ruleset requires a person's approval, so in a managed repository with a merge queue and no review rule, agent-written code merged on CI alone.

## Grounds

- pursued: a merge queue with no approval rule leaves the pull request open with no 'pr merge' call, and this repository's own mirror still arms (land_approval_test.go); shown wrong if the stub forge logs 'pr merge' for an unreviewed queue, or not for this repository's mirror
