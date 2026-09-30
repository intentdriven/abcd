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
---

The loop armed auto-merge in a project whose rules require no approval: the implement loop's arm step (internal/core/implement/loop/land.go landArm) armed 'gh pr merge --auto' wherever the ruleset mirror named a merge queue, without reading whether any ruleset requires a person's approval, so in a managed repository with a merge queue and no review rule, agent-written code merged on CI alone.
