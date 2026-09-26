---
schema_version: 1
id: "iss-2609260120380835"
slug: "when-abcd-site-setup-s-forge-stage-is-refused-gh-absent-or"
severity: "minor"
category: "security"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25 (fix round, review of lane sitesetup)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/site/setup.go"
resolution: "A refused forge stage prints the create-the-environments step ahead of the secret steps, as the unreachable path does. Impact internal: the verb is unreleased and ships with itd-2609061543533170."
impact: internal
resolved_by:
  spec: "spc-2609212141407459"
  commit: "7247b6cc"
---

When abcd site setup's forge stage is refused (gh absent or unauthenticated, or a forge call failing) it prints the gh secret set steps for the site environment but no step to create the environments, so a person following the steps sets the token and the first workflow run auto-creates site with no deployment policy, deployable from any branch.

## Grounds

- pursued: a person following the remaining steps of any refused or unreachable run protects the environment before setting a secret; a remaining list with a secret step and no environment step ahead of it would show it wrong
