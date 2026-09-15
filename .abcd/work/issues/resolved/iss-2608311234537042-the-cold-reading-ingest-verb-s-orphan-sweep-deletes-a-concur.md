---
schema_version: 1
id: "iss-2608311234537042"
slug: "the-cold-reading-ingest-verb-s-orphan-sweep-deletes-a-concur"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
resolution: "An flock on the stage root serialises the sweep and the write, so a peer invocation waits instead of rolling a live run back."
impact: fix
resolved_by:
  intent: "itd-185"
  spec: "spc-63"
---

The cold-reading ingest verb's orphan sweep deletes a concurrent invocation's ledger records. sweepOrphanStages and rollbackRun hold no lock, and nothing distinguishes a crashed stage from a live one, so a second ingest starting while the first sits between its ledger write and its commit marker finds the first's stage, sees no run.json, and removes both its records and its stage. The first then writes run.json and exits 0 naming records that no longer exist. capture.IngestReading takes the ledger flock for its own write, but the sweep sits outside it.

## Grounds

- pursued: the finding is closed by a test that fails without the change; a later review or mutation run finding the same shape again would show this wrong
