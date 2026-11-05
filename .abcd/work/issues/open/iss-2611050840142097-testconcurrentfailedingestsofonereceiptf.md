---
schema_version: 1
id: "iss-2611050840142097"
slug: "testconcurrentfailedingestsofonereceiptf"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "coordinating build lanes, 2026-11-05"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/auditowed_test.go"
remedy: "Wait as long as a holder needs on a loaded machine rather than a fixed 5 seconds: a longer bound, or waiting while the lock's holder is a live process, refusing only past a bound a loaded runner does not reach. A test holds the lock past 5 seconds and asserts the waiter still gets it."
---

TestConcurrentFailedIngestsOfOneReceiptFileOneIssue failed on a macOS shard with 'could not acquire mint lock within 5s', and its queue entry was dropped with no defect behind it. Four concurrent ingests contend for the intent store's mint lock, whose wait is a fixed 5 seconds (mintLockTimeout in internal/core/intent/create.go), and on a loaded runner a holder can outlast it. The wait is the production default, so a real ingest on a loaded machine can lose the same race.
