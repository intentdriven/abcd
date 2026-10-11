---
schema_version: 1
id: "iss-2608270655498099"
slug: "site-build-s-two-lint-loadconfig"
severity: "nitpick"
category: "security"
source: "agent-finding"
found_during: "security-cut-agent-flagged-siblings-2026-08-27"
found_at: "internal/core/lint"
resolution: "the site build/check read the lint config via lint.LoadConfigInRoot inside the repo os.Root, containing an ancestor symlink like every other site read"
impact: internal
resolved_by:
  commit: "018d805a"
---

site build's two lint.LoadConfig(joinRepo(...)) reads delegate into the external lint package and are not ancestor-symlink contained like the rest of site's reads now are (post-#487). A committed directory symlink ancestor of the lint config path is followed. Separate hardening in the lint package. Flagged by the #487 fix agent.