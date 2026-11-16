---
schema_version: 1
id: "iss-2611141347040644"
slug: "the-model-named-in-a-commit-s"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/land.go"
remedy: "bound the id's length where the trailer is composed"
---

The model named in a commit's assisted-by trailer is bounded in its characters and not in its length
