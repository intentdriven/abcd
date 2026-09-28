---
schema_version: 1
id: "iss-2609260927217168"
slug: "site-setup-policy-read-unpaginated"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review3-sitesetup note (e)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/site/forge.go"
resolution: "site setup reads every page of the environments, deployment-policy and secret lists to the forge's total_count and fails closed on an unreadable or short page"
impact: fix
resolved_by:
  commit: "624c1b05"
---

site setup reads an environment's deployment branch policies with per_page=100 and no pagination (forge.go:112), so a broad branch rule past the 100th policy is invisible to the restriction check and the environment can read as restricted when it is not. Reaching it needs an admin who already put over 100 rules on one environment; the environments and secrets reads at :84 and :145 share the shape.

## Grounds

- pursued: setup sees an environment, policy or secret past the hundredth and never rewrites an existing environment it could not see; a list read that returns fewer entries than the forge's total without an error would show it wrong
