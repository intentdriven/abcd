---
schema_version: 1
id: "iss-2608220150157511"
slug: "lint-gated-indexes-and-append-logs-are-structural-conflict-sites"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "abcdev-site session close-out 2026-08-22"
found_at: ".abcd/development/brief/06-delivery/03-out-of-scope.md"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (planning owed: generated-at-build indexes compared by lint instead of hand-edited regions, across five registers). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
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
