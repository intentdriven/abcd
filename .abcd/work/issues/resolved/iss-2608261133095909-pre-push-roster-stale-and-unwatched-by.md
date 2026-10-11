---
schema_version: 1
id: "iss-2608261133095909"
slug: "pre-push-roster-stale-and-unwatched-by"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "bughunt-round-8"
found_at: ".githooks/pre-push:11"
resolution: "the hook names the four lint gates plus site-render and joins the preflight-gates detector surface list, which fails on the stale roster before the correction"
impact: internal
resolved_by:
  commit: "3ea12cd6"
---

the pre-push hook restates a three-gate preflight roster while the recipe has five, and the preflight-gates drift detector omits the hook from its surface list, so the iss-182 drift recurred on the one unwatched surface