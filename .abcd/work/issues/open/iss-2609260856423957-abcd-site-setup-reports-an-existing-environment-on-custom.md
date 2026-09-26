---
schema_version: 1
id: "iss-2609260856423957"
slug: "abcd-site-setup-reports-an-existing-environment-on-custom"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/site/setup.go"
---

abcd site setup reports an existing environment on custom deployment policies as current, or adds the missing rules and reports it written, with no restriction step, even when it also holds a rule broader than the default branch and tags v* (a branch policy `*`, for example). setupEnvironments marks an environment unrestricted only when it is not on custom policies or admits protected branches; it reads the existing policies only to find the missing ones, never to find the extra ones, so an environment that admits every branch through a custom rule passes as restricted and the deploy environment's secrets can be reached from unreviewed workflow content.
