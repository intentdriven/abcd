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
---

When abcd site setup's forge stage is refused (gh absent or unauthenticated, or a forge call failing) it prints the gh secret set steps for the site environment but no step to create the environments, so a person following the steps sets the token and the first workflow run auto-creates site with no deployment policy, deployable from any branch.
