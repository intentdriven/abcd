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
resolution: "site setup reads an existing environment's custom rules for extra ones too: a rule beyond the default branch and tags v* makes it unrestricted, with a step naming the rules to remove, and setup writes nothing to it"
impact: internal
resolved_by:
  commit: "c13c3e94"
---

abcd site setup reports an existing environment on custom deployment policies as current, or adds the missing rules and reports it written, with no restriction step, even when it also holds a rule broader than the default branch and tags v* (a branch policy `*`, for example). setupEnvironments marks an environment unrestricted only when it is not on custom policies or admits protected branches; it reads the existing policies only to find the missing ones, never to find the extra ones, so an environment that admits every branch through a custom rule passes as restricted and the deploy environment's secrets can be reached from unreviewed workflow content.

## Grounds

- pursued: an environment holding any rule beyond branch <default> and tag v* is reported unrestricted with a restriction step and receives no forge write; a run against such an environment that reports it current or written, or writes a policy to it, would show it wrong
