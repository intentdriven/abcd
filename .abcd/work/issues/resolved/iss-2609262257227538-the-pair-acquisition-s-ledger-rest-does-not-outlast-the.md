---
schema_version: 1
id: "iss-2609262257227538"
slug: "the-pair-acquisition-s-ledger-rest-does-not-outlast-the"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: verify-fix2-drainL"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/ledgerlock.go"
resolution: "fsutil.LockPollCeiling names the poll's ceiling and the loop holds at it; the pair's ledger rest is 2x the ceiling; two tests pin both"
impact: internal
resolved_by:
  commit: "58a513ac6"
---

The pair acquisition's ledger rest does not outlast the ledger writer's poll interval it was chosen to outlast. WithLedgerThenMintLock's comment says fsutil's lock-poll backoff tops out at 100ms and pairLedgerRest is 150ms to exceed it, but acquireFlock doubles the backoff after the sleep while it is below 100ms, so 80ms becomes 160ms and the real poll ceiling is 160ms: a ledger writer sleeping 160ms can miss the whole 150ms window the pair leaves the ledger free. No test pins the ceiling or the rest, so either can move and break the guarantee silently.

## Grounds

- pursued: a ledger writer polling through the pair's wait always wakes inside the window the pair leaves the ledger free; TestTheLockPollNeverSleepsPastItsCeiling or TestThePairRestOutlastsTheLedgerPoll failing would show it wrong
