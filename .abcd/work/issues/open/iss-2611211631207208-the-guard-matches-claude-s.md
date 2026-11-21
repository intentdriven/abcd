---
schema_version: 1
id: "iss-2611211631207208"
slug: "the-guard-matches-claude-s"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "review-of-guard-coverage"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard"
remedy: "Refuse a claude settings flag that sets the bypass permission mode or any allow rule, reading the JSON on the line and a readable named file"
refines: [iss-2610240932215918]
---

The guard matches claude's permission-mode flag but not its settings flag, whose JSON or file can set the same mode or allow rules.
