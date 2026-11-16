---
schema_version: 1
id: "iss-2611160001261097"
slug: "the-guard-approver-s-list-of-git"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/approvepaths.go"
remedy: "judge each such setting against git's documentation and list it, with one list serving the approver and the hook"
---

The guard approver's list of git settings that name a program is not yet judged one by one against git's documentation
