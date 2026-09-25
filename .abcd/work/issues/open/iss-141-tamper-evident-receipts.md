---
schema_version: 1
id: "iss-141"
slug: "tamper-evident-receipts"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "itd-100 terminology crosswalk"
found_at: ".abcd/development/intents/drafts/itd-16-hash-chain-merkle-audit.md"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Plan tamper-evident receipts through draft itd-16, and by signing or by a verifiable log?"
---

tamper-evident receipts: receipts and review directories are hash-anchored and manifests are verified, but nothing is cryptographically signed and there is no append-only chain over the receipt history — integrity evidence, not proof against a capable insider. itd-16 (hash-chain/Merkle audit umbrella, draft) is the design home; this capture tracks the narrower question of whether the existing receipt flow should become tamper-evident (RFC 9162-style verifiable log or signing) independent of the full audit umbrella. The itd-100 crosswalk's tamper-evidence row cites this id.