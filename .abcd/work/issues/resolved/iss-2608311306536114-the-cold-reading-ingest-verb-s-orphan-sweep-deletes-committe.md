---
schema_version: 1
id: "iss-2608311306536114"
slug: "the-cold-reading-ingest-verb-s-orphan-sweep-deletes-committe"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
resolution: "The sweep's unlink of committed reading records runs under capture's ledger lock, exported as capture.WithLedgerLock and taken around the unlink alone so it cannot nest inside IngestReading's own acquisition."
impact: fix
resolved_by:
  intent: "itd-185"
  spec: "spc-63"
---

The cold-reading ingest verb's orphan sweep deletes committed ledger records without holding capture's ledger lock. The stage lock serialises ingest against ingest only, so a concurrent capture disposition or promote in the same checkout can be reading a reading record while the sweep unlinks it.

## Grounds

- pursued: the two locks are always acquired stage-then-ledger so no cycle exists, and a concurrent ledger verb waits rather than reading a record as it disappears; a deadlock or an unlocked unlink would show this wrong
