---
schema_version: 1
id: "iss-2608311230305308"
slug: "the-ingest-verb-permits-re-ingesting-a-run-id-that-already-c"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
resolution: "refuseARerun refuses a run id that already carries a commit marker or a refusal record."
impact: fix
resolved_by:
  intent: "itd-185"
  spec: "spc-63"
---

The ingest verb permits re-ingesting a run id that already committed: run_id is payload-chosen and nothing refuses a second ingest, so the second write overwrites run.json while the first run's reading records stay in the ledger unnamed by any run record, and a later refusal can drop refusal.json beside an existing run.json so one run directory asserts both that it committed and that it was refused. spc-63 and ingest.go both say a rerun is a new run with a new run id, never an amendment; that is a comment rather than a check.

## Grounds

- pursued: the finding is closed by a test that fails without the change; a later review or mutation run finding the same shape again would show this wrong
