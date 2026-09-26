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
---

site setup reads an environment's deployment branch policies with per_page=100 and no pagination (forge.go:112), so a broad branch rule past the 100th policy is invisible to the restriction check and the environment can read as restricted when it is not. Reaching it needs an admin who already put over 100 rules on one environment; the environments and secrets reads at :84 and :145 share the shape.
