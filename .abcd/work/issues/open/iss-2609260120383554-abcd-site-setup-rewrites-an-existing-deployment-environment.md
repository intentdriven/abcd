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
---

abcd site setup rewrites an existing deployment environment through PUT /repos/{r}/environments/{env} with only deployment_branch_policy in the body, and that endpoint replaces the whole protection set, so a site environment carrying required reviewers, a wait timer or prevent_self_review loses them after a confirmation that names only a branch restriction (setup.go plan.put, forge.go PutEnvironment).
