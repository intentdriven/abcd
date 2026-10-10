---
schema_version: 1
id: "iss-2609281654467661"
slug: "the-macos-keychain-home-s-write-and-read"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-cred"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/credential/keychain.go"
remedy: "Waits on the technical facilitator's owed round trip, run once on a real Mac: store a MaxValueBytes value through abcd ahoy credential with the keychain home, read it back and compare byte for byte, delete the item, and record the result on this record. If the value truncates, cap the keychain home's accepted size below the measured limit in internal/core/credential/keychain.go and pin the cap with a fake-keychain test that refuses an oversized value before anything is written."
deferred_after: "v0.11.1"
deferral_reason: "owed to the technical facilitator as a human act outside the tree (lane drainFresh of autonomous run A, 2026-09-29): one real-machine round trip of a MaxValueBytes value through the macOS keychain home, read back byte for byte, then the item removed. An autonomous run must not write to the real keychain, so it cannot run here; until it runs, the fake keychain is the only proof."
---

The macOS keychain home's write and read have never run against the real security tool. Set stores a value through 'security -i' with the value hex-encoded behind -X on one stdin line (internal/core/credential/keychain.go), and Resolve reads it back with find-generic-password -w; a MaxValueBytes value (4096 bytes) makes that line about 8 KiB. Only the test binary's fake keychain (store_test.go TestMain) has proven the -X in, -w out round trip. If security's non-terminal line reader splits or truncates a long line, the read-back check after the write refuses loudly and echoes nothing, but a wrong item stays in the keychain for the person to remove by hand. One real-machine round trip is owed: a value of MaxValueBytes set through the keychain home, read back byte for byte, then the item removed. Surfaced by the review of the credential store lane (review-cred finding 5, unverified).

## Remedy grounds (2026-09-29)

- Why: the gap is evidence, not code; an autonomous run must not write to a real keychain, so the remedy is the one human act (option (a) of the owed item K+) with option (b), the size cap, as the fallback its result would trigger.
- Rejected: capping the size pre-emptively, which would narrow a limit nobody has shown to be wrong.
