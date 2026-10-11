---
schema_version: 1
id: "iss-150"
slug: "a-committed-positioning-registry-may"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "itd-102-security-review"
found_at: "internal/core/positioning/config.go"
resolution: "Surface.validate refuses a Files entry under .git/, so a credential-bearing remote URL cannot be quoted into identity output"
impact: fix
resolved_by:
  commit: "286784c1"
---

A committed positioning registry may name a surface inside .git (ValidRelPath accepts .git/config and it is inside the containment root), so abcd identity --json quotes .git/config — including a credential-bearing remote URL — into SurfaceResult.Found. Pre-existing; not an escape, and the output reaches only the operator's own stdout. Detector: a Config.Validate test refusing surface candidates under .git/. Found during the itd-102 containment review.