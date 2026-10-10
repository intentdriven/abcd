---
schema_version: 1
id: "iss-2608221456466438"
slug: "loadrepometa-reads-claude-plugin-plugin"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "all-dimensions bug-hunt round 7"
found_at: "internal/core/site/build.go"
resolution: "LoadRepoMeta screens the repository address for an executable scheme and refuses the build."
impact: fix
resolved_by:
  commit: "de19186"
---

LoadRepoMeta reads .claude-plugin/plugin.json repository into an href on every emitted page with no executableScheme check, unlike its two sibling href sources, so a javascript: address (or a scheme-less typo) passes every gate and ships on every page.