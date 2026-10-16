---
schema_version: 1
id: "iss-2610152149380984"
slug: "the-banlist-hook-state-hookspathstate-in"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "outbound switch fix round 2, 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/banlist_scaffold.go"
remedy: "Treat an empty last core.hooksPath value as 'git runs no hooks' and report the banlist hook as not armed, with the reason"
refines: [iss-2610020704152920]
---

The banlist hook state (hooksPathState in internal/core/ahoy/banlist_scaffold.go) follows the global core.hooksPath when the clone sets core.hooksPath to the empty string. Git runs no hooks at all in that case, but when the global value is .githooks the state reports the banlist hook as armed.
