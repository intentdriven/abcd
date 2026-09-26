---
schema_version: 1
id: "iss-2609260120383554"
slug: "abcd-site-setup-rewrites-an-existing-deployment-environment"
severity: "major"
category: "security"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25 (fix round, review of lane sitesetup)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/site/setup.go"
resolution: "Setup writes the forge's environment endpoint only for an absent environment; an existing one gains missing policies through their own endpoint or, when it admits more refs, is reported unrestricted with the restriction as a remaining step. Impact internal: the verb is unreleased and ships with itd-2609061543533170."
impact: internal
resolved_by:
  spec: "spc-2609212141407459"
  commit: "ab227d68"
---

abcd site setup rewrites an existing deployment environment through PUT /repos/{r}/environments/{env} with only deployment_branch_policy in the body, and that endpoint replaces the whole protection set, so a site environment carrying required reviewers, a wait timer or prevent_self_review loses them after a confirmation that names only a branch restriction (setup.go plan.put, forge.go PutEnvironment).

## Grounds

- pursued: an existing environment's reviewers and wait timer survive any setup run; a forge write to the environment endpoint for an environment that already exists would show it wrong
