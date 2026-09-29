---
schema_version: 1
id: "iss-2608301301041887"
slug: "the-lock-assertion-in-the-grounds-test-is-wall-clock-so-a-lo"
severity: "nitpick"
category: "tech-debt"
source: "user-observation"
found_during: "itd-179-round-3-ruthless"
found_at: "internal/core/intent/grounds_test.go"
resolution: "Both mint-lock proofs in internal/core/intent assert an observed ordering instead of a wait: withIntentMintLockWithin calls the onIntentMintLockBusy test seam when an attempt finds the lock held, the holder releases only on that event, and a flag raised inside its critical section tells a writer that waited from one that took no lock. TestRecordGroundsHoldsTheMintLock and its sibling TestStampPlannedHoldsTheMintLock (claims_fence_test.go), which carried the same wall-clock shape, both fail a scratch mutant that sleeps 200ms and skips the lock, which the old wait bar passed."
impact: internal
resolved_by:
  commit: "749fcf915"
---

the lock assertion in the grounds test is wall-clock so a loaded runner could pass it with no lock held

Found by the round-3 adversarial ruthless review of build/itd-179.

The assertion is wall-clock: `waited < 100ms` against a 150ms hold. On the
reviewer's machine the unlocked mutant completed in 34ms, a ~3x margin, so a
sufficiently loaded runner could pass this test with no lock held at all.

It is NOT vacuous, and that distinction is the record: the reviewer proved the
guard independently via `TestRecordGroundsConcurrentAppendsBothLand`, which
loses entries deterministically without the lock. So the lock is genuinely
covered; this test is redundancy of a weaker kind sitting beside a
deterministic one.

Worth fixing when the file is next touched, because a timing assertion that can
pass for the wrong reason is the seed of the mutation-vacuous class this
workstream has now found four times.

## Grounds

- pursued: we expect both lock tests to fail any writer that skips the mint lock, however slow; a no-lock mutant that passes either test would show it wrong
