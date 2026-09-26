---
schema_version: 1
id: "iss-2609262211082803"
slug: "the-reading-ledger-capture-verbs-disposition-admit-surprise"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli.go"
---

The reading-ledger capture verbs (disposition, admit, surprise, reframe) and capture mentions exit 1 on refusals of their own input, while capture migrate exits 2 on every fault: both contradict the capture page and brief claim that every refusal of a ledger verb's input exits 2 and exit 1 is left to a fault (iss-2609260552251398). Found when the drain1 lane met the reading-ledger verbs at the integ6 re-merge.
