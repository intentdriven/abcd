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
resolution: "itd-101's fourth criterion promises a LATER checklist page, and spc-17 records it as a later rung; the false help text is iss-2609240519418856"
impact: internal
---

itd-101 ac-4 promises that the later generated checklist page hands back a receipt file the confirm verb ingests; delivered reality: docs cite confirm --receipt ingests the one Receipt schema (internal/core/cite/confirm.go) and refresh prints the manual queue, but no generated checklist page exists in the tree — spc-17 records it as a rung that may land later, so the receipt-file half of the criterion has a consumer and no producer. Found by the itd-101 fidelity audit (rcp-c9215c58f907).

## Grounds

- pursued: the receipt consumer exists and its producer is a recorded later rung; shown wrong if spc-17 stops naming the checklist page as a later rung
