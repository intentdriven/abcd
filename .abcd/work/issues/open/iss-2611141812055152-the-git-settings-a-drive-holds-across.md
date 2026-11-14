---
schema_version: 1
id: "iss-2611141812055152"
slug: "the-git-settings-a-drive-holds-across"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/gitenvs.go"
remedy: "judge each such setting: pinned on the command line, held, or shown unread by any command the tree runs, with the table kept beside the held set"
---

The git settings a drive holds across its lanes are not yet judged one by one against git's documentation of the settings that name a program
