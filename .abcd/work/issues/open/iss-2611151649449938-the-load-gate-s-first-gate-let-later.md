---
schema_version: 1
id: "iss-2611151649449938"
slug: "the-load-gate-s-first-gate-let-later"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "autonomous run, 2026-11-15 load check"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/machineload/queue.go"
remedy: "Find why the later waiters did not count the earlier one as queued ahead (candidates: the earlier waiter was built from an older abcd, a start-order tie read differently by the two accounts, or the crossed-waiter cap), fix that cause, and add a test with two waiters under different accounts and different settings that asserts the earlier one opens first."
---

The load gate's first gate let later waiters of one account open ahead of an earlier waiter of another account for four hours. On 2026-11-15 one account's make preflight waiter started at 12:45 UTC with that account at one preflight at a time; the other account, at two, had waiters started at 14:08, 15:25 and 16:03 UTC open ahead of it while the first stayed shut until at least 16:49. The LOAD rule says a waiter queued ahead of yours delays yours under any account until it opens or reaches its own timeout, and the queue's shut test (Ahead.Shut in internal/core/machineload/queue.go) already counts a queued waiter at any setting, so the cause is not yet known. Until it is found, an account at one can be starved for as long as the other keeps preflights coming.
