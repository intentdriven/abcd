---
schema_version: 1
id: "iss-2610252214594592"
slug: "04-surfaces-readme-md-152-s-agents-block"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S11"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/README.md"
remedy: "Add guard approve to the row, or replace the hand table with a pointer to helpPlacements and the snapshot."
---

04-surfaces/README.md:152's agents-block row omits guard approve, which helpgroups.go:143 annotates into the agents block and the snapshot records with block agents.
