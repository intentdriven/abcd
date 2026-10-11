---
schema_version: 1
id: "iss-2608220150157511"
slug: "lint-gated-indexes-and-append-logs-are"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "abcdev-site session close-out 2026-08-22"
found_at: ".abcd/development/brief/06-delivery/03-out-of-scope.md"
deferred_after: "v0.11.1"
deferral_reason: "lane owed (lapsed-deferral triage, run A 2026-09-29): a lane of its own (well over an hour): derive the five hand-edited registers (the out-of-scope drafts index, 04-surfaces/README.md, decisions/adrs/README.md, intents/README.md, release/surface.json) at build time and have lint compare against the derivation; plus a product-thinker ruling on whether to plan it as an intent first."
remedy: "Waits on the ruling on whether to plan it as an intent first; either way, replace the hand-edited enumerations in the out-of-scope drafts index, 04-surfaces/README.md, decisions/adrs/README.md and intents/README.md with regions one generator writes from the filesystem, on the surface-appendix precedent (adr-2609231028044006), so a conflicted region is resolved by rerunning the generator and index_drift compares each region with the derivation; release/surface.json stays committed as the compatibility contract, regenerated the same way. Prove it with a test per register that a stale region fails naming the regenerate command, and abcd decide leaving no index row to write by hand."
---

Merge contention is structural in the shared registers, not the prose: append logs (.abcd/work/DECISIONS.md, 154 changes; the decomposition-calibration corpus, conflicted 2026-08-22) and the lint-gated indexes that index_drift forces every branch to co-edit (brief/06-delivery/03-out-of-scope.md drafts index, conflicted the same day; 04-surfaces/README.md; decisions/adrs/README.md; intents/README.md; release/surface.json). The fix shape for the indexes is already written down: the enumeration command in out-of-scope.md derives the list from the filesystem, so the committed copy could become generated-at-build and the lint compare against the derivation instead of a hand-edited region. Append logs tolerate union merges; the indexes do not
**Corroboration (2026-09-18, Gropius managed-repo session gropiusllm-56, relayed
to abcd-17).** The ADR index half was met from the other side: `abcd decide
"<title>"` at v0.9.0 minted the record as documented (a decision record in
that repository's own store), and the session then found the two steps the verb leaves by
hand, `status: accepted` and the index row in the store README, and asked for a
`--accept` flag or an index write on `decide`. The status half is by design
(decide.go: the binary knows an id and a date and cannot know a decision is in
force); the index half is this record's fix shape, a derived index in place of
a hand-edited enumeration, now wanted in a managed repository as well as here.
The ledger already carries two resolved instances of the row being forgotten
(iss-2608211144555085, iss-2608220750029986).

## Remedy grounds (2026-09-29)

- Why: the record's own fix shape, and the base already holds its precedent: generated chapter appendices held by TestSurfaceAppendicesMatchCommandTree in internal/surface/cli. It also answers the managed-repository corroboration, where decide left the index row by hand.
- Rejected: a union merge for the indexes, which keeps both sides' lines but not the ordering and deduplication an index needs.
