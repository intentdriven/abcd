---
schema_version: 1
id: "iss-2609290825240166"
slug: "itd-2609221017023290-ac-3-promises-that-the-store-s-write"
severity: "minor"
category: "drift"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: fidelity audit itd-2609221017023290"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/credential/store.go"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (lane drainFresh of autonomous run A, 2026-09-29): Do adr-2609221017021499 ruling 4 and itd-2609221017023290 ac-3 get amended to say the index is what is scanned and the value write is what is refused inside a working tree, or does a ruling say why the wording stands? The invariant holds and is tested; only the record's wording differs from what ships."
---

itd-2609221017023290 ac-3 promises that the store's write path runs the scanner and refuses a write that would land in a tracked path, and adr-2609221017021499 ruling 4 says the scanner runs on any file the write path touches; the delivered store scans the index (~/.abcd/credential-homes.json) alone, writes ~/.abcd/credentials.json, the file that holds the value, unscanned by construction (internal/core/credential/credential.go SetMachine), and refuses only the abcd home's value write inside a git working tree while the index is still written there (internal/core/credential/store.go Set, ruled at review in 278e266d8 and stated on commands/ahoy.md). The invariant the intent guards, no value in a tracked path or the harness's settings, holds and is tested; the record's wording does not match what ships, and a scan of the value file is unsatisfiable, since a real key is exactly what the scanner flags. The ADR and the intent should say the index is what is scanned and the value write is what is refused, or a ruling should say why the wording stands. Fidelity audit verdict: ac-3 MET_WITH_CONCERNS.
