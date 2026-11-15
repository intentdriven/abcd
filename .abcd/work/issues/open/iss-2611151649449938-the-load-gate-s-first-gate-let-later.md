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
found_at: "internal"
remedy: "Make the first gate at a setting above one count a waiter queued ahead under any account, whose own limit keeps it shut, as holding every later waiter until it opens or times out, as the rule states; add a test with two waiters of different settings and assert the earlier one opens first."
---

The load gate's first gate let later waiters of one account open ahead of an earlier waiter of another account for four hours. On 2026-11-15 one account's make preflight waiter started at 12:45 UTC with that account at one preflight at a time; the other account, at two, had waiters started at 14:08, 15:25 and 16:03 UTC open ahead of it while the first stayed shut until at least 16:50. The LOAD rule says a waiter queued ahead of yours delays yours under any account until it opens or reaches its own timeout, so the admission at a setting above one does not do what the rule says, and an account at one can be starved for as long as the other keeps preflights coming.
