---
schema_version: 1
id: "iss-2610252214488399"
slug: "20-banlist-md-38-42-records-as-a-rough"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S09"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/banlist.go"
remedy: "Capture it, or make a private add refuse both flags, then cut the brief sentence to the record id."
---

20-banlist.md:38-42 records as a 'rough edge' that a private add accepts and silently drops the severity and successor flags (internal/surface/cli/banlist.go:170-171 passes neither to AddPrivate); no ledger record holds it.
