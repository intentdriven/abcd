---
schema_version: 1
id: "iss-2610252214455357"
slug: "decide-sends-every-record-and-mint-error"
severity: "nitpick"
category: "drift"
source: "agent-finding"
found_during: "brief single-source study, 2026-10-25, reader S07"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/decide.go"
remedy: "Send non-operand failures to exit 1, or list them under exit 2 on both pages."
---

decide sends every record and mint error, a mint-lock timeout or a write failure included, to exit 2 (internal/surface/cli/decide.go:94,110). commands/decide.md:129-133 and 24-decide.md:95-97 describe exit 2 as an operand refusal only, and capture keeps exit 1 for faults (06-capture.md:310).
