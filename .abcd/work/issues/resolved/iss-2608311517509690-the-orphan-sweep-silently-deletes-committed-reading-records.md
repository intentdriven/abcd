---
schema_version: 1
id: "iss-2608311517509690"
slug: "the-orphan-sweep-silently-deletes-committed-reading-records"
severity: "critical"
category: "bug"
source: "agent-finding"
found_during: "itd-185 fidelity audit rcp-fe3450ca55ff"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/ingest.go"
resolution: "The ingest now probes the stage root read-only, validates the whole payload, and only then sweeps: the sweep is the verb's first delete in the committed tier and a refused run never reaches it. A refused or failed run reports the orphans it left in place as pending_stages, and the front door renders the result on the error path whenever the core says it has something to disclose, in text and JSON alike."
impact: fix
resolved_by:
  commit: "2b8e82b2483443d37c4d1794b9855662029861e8"
---

The orphan sweep silently deletes committed reading records. With an orphaned stage present and its record already in the ledger, an ingest whose payload fails at the type check deleted a committed record from the durable tier and printed only the type error: the rolled-back and cleared-stage fields are dropped on the error path, and the JSON render emits an error object without them, because the surface renders those fields only when a refusal record exists. The code's own comment says this must not happen. It is the worst shape a defect can take here, because the loss is durable, silent, and triggered by a payload that was itself refused: an operator sees a validation error and has no way to learn that a record was destroyed while it was being reported.

## Grounds

- pursued: validate-then-mutate makes the invariant structural rather than rendered; if a refused ingest ever again removes a ledger record, TestARefusedRunLeavesTheOrphanInPlace and TestIngestDisclosesTheOrphanItLeftOnRefusal go red
