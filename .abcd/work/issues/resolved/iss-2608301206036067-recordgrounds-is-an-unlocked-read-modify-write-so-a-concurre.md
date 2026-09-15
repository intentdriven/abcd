---
schema_version: 1
id: "iss-2608301206036067"
slug: "recordgrounds-is-an-unlocked-read-modify-write-so-a-concurre"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "itd-179-round-2-security"
found_at: "internal/core/intent/grounds.go"
resolution: "RecordGrounds now runs read -> append -> read-back -> write inside withIntentMintLock, so a second concurrent grounds write can no longer discard the first"
impact: fix
resolved_by:
  intent: "itd-179"
---

RecordGrounds is an unlocked read-modify-write so a concurrent second grounds write silently discards the first

Found by the round-2 adversarial security review of build/itd-179.

Two sessions each running `abcd intent ready itd-N --grounds "..."` both read
the same record, each appends onto its own snapshot, and the later write wins.
Both processes print the "recorded grounds" receipt and exit clean, so nothing
reports the loss: the `groundsWriteIsReadable` count check compares each
writer's own before/after pair, which stays consistent. Reproduced in 40 trials
of two concurrent `RecordGrounds` calls on one record; 39 lost an entry.

This falsifies the append-only guarantee the same file documents at
`internal/core/intent/grounds.go:50-55` ("Recording is APPEND-ONLY: a second
gate decision adds an entry beside the first rather than replacing it").

It also reopens a defect already resolved one day earlier in this same package:
iss-2608300235388164, whose body records that `stampPlanned` was a
read-modify-write with no lock, and whose fix put the sequence under
`withIntentMintLock` at `internal/core/intent/lifecycle.go:319` with the
comment naming exactly this failure. The branch adds a second writer to the
record and does not take the lock.

Remedy: wrap the readRepoFile -> appendGroundsBullet -> writeIntentFile
sequence in `withIntentMintLock`, as `stampPlanned` does.

CORROBORATED independently by the round-2 ruthless review, which graded it
FIX-FIRST, reproduced it 20/20 with two goroutines (zero errors returned), and
verified the remedy: applying `withIntentMintLock` around read -> append ->
`groundsWriteIsReadable` -> `writeIntentFile` in a scratch copy takes the
reproduction to 20/20 green with `go test` and `go test -race
./internal/core/intent/` both staying green.

Two adversarial reviewers, in fresh contexts and by different routes, landed on
the same defect and the same one-line remedy. That is the strongest signal this
cycle has produced for a single finding.

## Grounds

- pursued: we expect an append-only contract to hold only where one advisory lock spans the whole read-modify-write, and a concurrent-loss reproduction that stays green after the lock is what would show it wrong
