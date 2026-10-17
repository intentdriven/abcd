---
schema_version: 1
id: "iss-2610161332064271"
slug: "both-user-scope-inventories"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "deferred config-brief fixes, inventory check"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md"
remedy: "Add ideas/, sessions/, filter-roots, load-wait.json and dashboard/ to both inventories, each with its writer."
---

Both user-scope inventories (03-configuration.md and 01-ahoy's tree) omit files abcd reads or writes under ~/.abcd.noindex: ideas/<root-sha>/ (private ideate verdicts, internal/core/ideate), sessions/names.json (the session-name register, internal/core/sessionname/register.go), filter-roots (internal/gitutil/status.go), load-wait.json (internal/core/implement/load_wait.go) and dashboard/ (run.json, seen.json, control.lock; internal/surface/dashboard/control.go).
