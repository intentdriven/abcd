---
schema_version: 1
id: "iss-2611140612489896"
slug: "the-banlist-report-answers-unarmed-where"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/banlist_scaffold.go"
remedy: "Report a fourth hooks-path state, that abcd could not tell, where git cannot read core.hooksPath, and name the read"
---

The banlist report answers unarmed where the hooks path cannot be read; it should say it could not tell
