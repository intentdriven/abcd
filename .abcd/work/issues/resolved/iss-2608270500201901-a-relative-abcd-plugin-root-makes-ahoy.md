---
schema_version: 1
id: "iss-2608270500201901"
slug: "a-relative-abcd-plugin-root-makes-ahoy"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "github-ledger-dedup-2026-08-27"
found_at: "internal/core/ahoy (resolvePluginRoot)"
resolution: "ahoy install absolutises ABCD_PLUGIN_ROOT, refusing a self-referential or dangling PATH link (#334)"
impact: fix
---

a relative ABCD_PLUGIN_ROOT makes ahoy install write a self-referential (abcd -> abcd, ELOOP) or dangling PATH entry and report it as wrote:, because the anti-dangling guard stats against the process CWD while the kernel resolves against the link's directory. GitHub mirror: #334