---
schema_version: 1
id: "iss-2609281654467661"
slug: "the-macos-keychain-home-s-write-and-read-have-never-run"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-cred"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/credential/keychain.go"
deferred_after: "v0.11.0"
deferral_reason: "Deferred out loud by autonomous run A (lane fix2-cred, 2026-09-28): the check is a real-machine round trip against the macOS security tool, which the run must not touch, so it is owed to the technical facilitator as a human act outside the tree (rulings-owed section K). Until it runs, the fake keychain is the only proof of the -X in, -w out round trip."
---

The macOS keychain home's write and read have never run against the real security tool. Set stores a value through 'security -i' with the value hex-encoded behind -X on one stdin line (internal/core/credential/keychain.go), and Resolve reads it back with find-generic-password -w; a MaxValueBytes value (4096 bytes) makes that line about 8 KiB. Only the test binary's fake keychain (store_test.go TestMain) has proven the -X in, -w out round trip. If security's non-terminal line reader splits or truncates a long line, the read-back check after the write refuses loudly and echoes nothing, but a wrong item stays in the keychain for the person to remove by hand. One real-machine round trip is owed: a value of MaxValueBytes set through the keychain home, read back byte for byte, then the item removed. Surfaced by the review of the credential store lane (review-cred finding 5, unverified).
