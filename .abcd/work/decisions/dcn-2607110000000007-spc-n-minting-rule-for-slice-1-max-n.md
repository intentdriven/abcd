---
id: dcn-2607110000000007
slug: spc-n-minting-rule-for-slice-1-max-n
date: 2026-07-11
---
2026-07-11 — `spc-N` minting rule for slice 1: `max(N over spec-store files ∪
N over every intent's spec_id) + 1`, so the first mint is spc-2 (itd-3's
reserved spc-1 is respected without a backing spec file). Reconciling the
store's sequential minting with the brief's aspirational spc-numbering is
deferred to the richer spec-store slice. Reviewer roles 2/3 (itd-48),
loop-to-acceptance (itd-50), bundle/discipline lifecycles, and the spec
dependency graph are all explicitly deferred.
