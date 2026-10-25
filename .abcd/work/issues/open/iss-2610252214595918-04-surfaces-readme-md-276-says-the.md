---
schema_version: 1
id: "iss-2610252214595918"
slug: "04-surfaces-readme-md-276-says-the"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S11"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/README.md"
remedy: "Name stop and stop-failure in the hook row, or point at the snapshot instead of counting."
---

04-surfaces/README.md:276 says the hidden hook has 'six host hook entrypoints'; the binary registers eight (stop, stop-failure added: internal/surface/cli/hook_stop.go:90, :259; hooks/hooks.json).
