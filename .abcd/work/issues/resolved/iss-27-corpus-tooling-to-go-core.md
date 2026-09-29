---
schema_version: 1
id: "iss-27"
slug: "corpus-tooling-to-go-core"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "sources-ingest session 2026-07-08"
related_intents: [itd-2609292108089653]
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Plan moving the corpus tooling (sources and ingest) into the Go core?"
resolution: "Already delivered at e792a2314 by itd-76 (spc-31 closed, 84b6eb9e5, shipped in v0.11.1): the corpus contract runs as the native abcd source verb family (init, add, ledger, declassify, sync-banlist, cite-check) in internal/core/source, the pre-commit guard calls sync-banlist instead of the user-tier script, and /abcd:consult and /abcd:ingest call the verbs. The record's suggested convergence with internal/core/provenance was not taken; the corpus is its own package, which itd-76 chose."
impact: internal
shipped_in: v0.11.1
resolved_by:
  intent: "itd-76"
  spec: "spc-31"
  commit: "84b6eb9e5"
---

corpus tooling graduates into the Go core: the sources-corpus contract (per-source folders with location-as-classification, CSL-JSON metadata, append-only provenance ledger, banlist projection) is proven by the user-tier script MVP; absorb it as a native abcd sources/ingest verb family, converging with the itd-36 provenance substrate (internal/core/provenance: licence detection, citation generation, source-hash registry) rather than porting standalone. The scripts remain the reference implementation until the native verbs are wired; /abcd:consult and /abcd:ingest keep one surface across the swap.

## Grounds

- pursued: the sources corpus needs no user-tier script any more, because every corpus operation the consult and ingest pages name is an abcd source verb; a page that still shells out to a script, or a corpus operation with no verb behind it, would show this wrong.
