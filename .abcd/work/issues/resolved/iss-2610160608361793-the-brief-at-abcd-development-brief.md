---
schema_version: 1
id: "iss-2610160608361793"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.0 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/11-history.md"
remedy: "Say at 11-history.md:192-203 that the Go entrypoints always exit 0 while the shell wrappers exit 1 when no binary resolves or a legacy ~/.abcd stands."
resolution: "11-history.md now scopes the always-exit-0 claim to the staging entrypoints and states that their shell wrappers exit 1 when no binary resolves, or when a legacy ~/.abcd stands with no plugin binary in the plugin root"
impact: internal
---

The brief at .abcd/development/brief/04-surfaces/11-history.md:192-195 and :203 says both staging entrypoints exit 0 on every path, and that on SubagentStop every failure path is a diagnostic and an exit 0. The registered hook commands, hooks/session-end.sh and hooks/subagent-stop.sh, exit 1 when the plugin binary is missing or a legacy ~/.abcd stands; only `abcd hook session-end` and `abcd hook subagent-stop` themselves always exit 0.
