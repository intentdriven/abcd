---
schema_version: 1
id: "iss-2609230734017018"
slug: "itd-101-ac-4-promises-that-the-later-generated-checklist"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "autonomous run 2026-09-23 fidelity audit"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/cite/confirm.go"
---

itd-101 ac-4 promises that the later generated checklist page hands back a receipt file the confirm verb ingests; delivered reality: docs cite confirm --receipt ingests the one Receipt schema (internal/core/cite/confirm.go) and refresh prints the manual queue, but no generated checklist page exists in the tree — spc-17 records it as a rung that may land later, so the receipt-file half of the criterion has a consumer and no producer. Found by the itd-101 fidelity audit (rcp-c9215c58f907).
