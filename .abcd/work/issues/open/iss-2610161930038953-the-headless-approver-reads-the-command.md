---
schema_version: 1
id: "iss-2610161930038953"
slug: "the-headless-approver-reads-the-command"
severity: "minor"
category: "architectural-insight"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard"
remedy: "Bound the writes of a script an approved line runs with the OS sandbox for launched agents, and keep the approver's help saying it reads the line, not the script's writes."
---

The headless approver reads the command line it is asked about, not the writes of a script that line runs: a script the approver lets run is read for the registry's hazards only, so a file the script writes, a protected one included, is not checked by the approver. The OS sandbox for launched agents, spc-2610161751056096, is the change that bounds those writes. <!-- record-lint: forward-looking -->
