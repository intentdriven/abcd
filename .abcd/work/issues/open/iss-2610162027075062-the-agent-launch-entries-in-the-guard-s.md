---
schema_version: 1
id: "iss-2610162027075062"
slug: "the-agent-launch-entries-in-the-guard-s"
severity: "minor"
category: "architectural-insight"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard"
remedy: "Bound what a harness started under another name can reach with the OS sandbox for launched agents, and keep commands/guard.md and the guard brief page saying the agent-launch entries recognise a harness by its name only."
---

The agent-launch entries in the guard's registry recognise a harness by its program name, by the package that ships it, and behind the launchers the registry steps, so a harness binary copied or linked under another name and started by that name is not recognised as a harness at this layer; the OS sandbox for launched agents, spc-2610161751056096, is what bounds what such a run can reach. <!-- record-lint: forward-looking -->
