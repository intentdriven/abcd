---
id: spc-2609211905174684
slug: memory-unification
intent: itd-36
origin: researcher-authored
production_mode: hand-written
---
# memory-unification

## Summary

The design record for itd-36 as delivered: `/abcd:memory` with `ingest`,
`ask` and `lint` over the per-project store at `.abcd/memory/`, shipped in
v0.1.0 (`internal/core/memory`, `commands/memory.md`). Written on
2026-09-21 to close a record that shipped without one.

## Scope, as delivered

1. **`memory ingest <path-or-url> [--keep-original]`**: reads a source,
   distils it into typed pages with `source.class`, citation and licence,
   discards the original by default and stores it under `sources/` by hash
   with the flag; the URL fetch masks its origin in every failure message.
2. **`memory ask <question>`**: answers from the store with citations by
   class, citation and source hash; term-safe over an empty question.
3. **`memory lint`**: MQ001 (per-page quotation span), MQ002 (cumulative
   coverage per source, refusing further quotation), MQ003, MS001 (advisory:
   single-class synthesis), MS002 (cross-class synthesis without a weighting
   note blocks), ML001 (undeclared licence blocks; `unknown` is explicit).
4. **The bare render**: store presence, last ingest, contradictions and
   per-source headroom.
5. **Legacy pages**: pre-existing flat-named pages are not renamed; the
   index is generated over them and they read as `session_memory`.

## Out of scope, captured as issues on 2026-09-21

- The dredge synthesiser's output landing as `dredge_synthesis` pages
  (itd-25 is a draft).
- One registry entry shared with the code-vendoring path (itd-26 is a draft).
- The lifeboat's restrictive-licence gate refusing to surface a kept
  original.

## How the criteria are satisfied

Criteria 1, 2, 4 to 9 and 13 by scope 1 to 5 above as shipped; criteria 3,
10, 11 and 12 are the out-of-scope items, each an issue in the ledger (iss-2609211905340006, iss-2609211905346507, iss-2609211905347458).
