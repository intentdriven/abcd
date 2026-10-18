---
schema_version: 1
id: "iss-2610161930104545"
slug: "a-command-that-writes-into-a-directory"
severity: "minor"
category: "architectural-insight"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard"
remedy: "Bound the files a directory-writing command lands with the OS sandbox for launched agents; the approver keeps judging the directory it writes into."
---

A command that writes into a directory (tar -x, unzip -d, cp -r, git apply, git checkout) is judged by the headless approver by the directory it writes into, not by the files that land inside it, so a protected file arriving inside that directory is not checked. The OS sandbox for launched agents, spc-2610161751056096, is the change that bounds those writes. <!-- record-lint: forward-looking -->
