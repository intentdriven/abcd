---
schema_version: 1
id: "iss-2610151422063401"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.2 release review, iss35 brief-surface crosscheck, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/11-history.md"
remedy: "Rewrite .abcd/development/brief/04-surfaces/11-history.md:216-219 to match the shipped surface: The shipped SessionStart hook (hooks/hooks.json -> hooks/session-start.sh) exits 2 when the plugin binary is not installed or when both ~/.abcd and ~/.abcd.noindex exist, and otherwise propagates the non-zero exit of bootstrap.sh or of `abcd hook…"
---

The brief at .abcd/development/brief/04-surfaces/11-history.md:216-219 says: The session-start hook that carries the notice channel 'always exits successfully, because a notice is not a failure and a failing session-start hook shows the user an empty error banner'. The shipped surface differs: The shipped SessionStart hook (hooks/hooks.json -> hooks/session-start.sh) exits 2 when the plugin binary is not installed or when both ~/.abcd and ~/.abcd.noindex exist, and otherwise propagates the non-zero exit of bootstrap.sh or of `abcd hook session-start`/`prompt-router-reset`; only the Go entrypoint itself is the always-exit-0 path. The chapter states the property for 'the hook' without that scope. (v0.14.2 release crosscheck, false-claim, raised by opus in a10.)
