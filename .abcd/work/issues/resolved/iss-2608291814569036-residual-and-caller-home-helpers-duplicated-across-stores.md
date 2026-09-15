---
schema_version: 1
id: "iss-2608291814569036"
slug: "residual-and-caller-home-helpers-duplicated-across-stores"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "ultra-v0.6.8-followup"
found_at: "internal/core/memory/redact.go"
resolution: "scanner.CallerHome, scanner.BlockingResidual and scanner.SurvivingCallerHome in internal/adapter/scanner/residual.go are the one home; history, memory and ProbeIdentity call them, and the shared home carries the unit tests the copies never had"
impact: internal
---

ultra-v0.6.8 C7: memoryBlockingResidual and memoryCallerHome in internal/core/memory/redact.go are byte-for-byte copies of history.blockingResidual (internal/core/history/history.go) and history.callerHome (internal/core/history/store.go); scanner.ProbeIdentity carries the same home resolution inline. Two committed-store redactors that are meant to agree on what counts as a leak now have to be fixed twice (C3 is the concrete cost). Both packages already import internal/adapter/scanner, so one exported home there has no cycle risk. Neither copy has a unit test of its own (the below-cap conv-3 note); the shared home gets them.
