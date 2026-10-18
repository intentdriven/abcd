---
schema_version: 1
id: "iss-2610161930115246"
slug: "the-file-tools-a-headless-role-is"
severity: "minor"
category: "architectural-insight"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard"
remedy: "Bound the writes of the role's granted file tools with the OS sandbox for launched agents, and keep commands/guard.md and the guard brief page saying those tools write without the approver."
---

The file tools a headless role is granted (the --allowedTools list of the claude launch, internal/core/runner/claude.go) write files without asking the approver, so the approver's protected set does not cover what those tools write. The OS sandbox for launched agents, spc-2610161751056096, is the change that bounds those writes. <!-- record-lint: forward-looking -->
