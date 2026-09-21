---
id: spc-2609211921272106
slug: intent-fidelity-reviewer-roles-2-3
intent: itd-48
origin: researcher-authored
production_mode: hand-written
---
# intent-fidelity-reviewer-roles-2-3

## Summary

The design record for itd-48 as re-scoped on 2026-09-21: one on-demand
consistency pass over the brief and every intent, producing a dated report
and a capture per finding.

## Scope

1. **`abcd intent consistency [<itd-N>]`**: assembles the input (the brief's
   chapters, every intent's press release, scope and decisions, the glossary
   entries), hands it to the host with the five drift classes named, and
   validates the returned findings (class, both ends quoted with paths,
   severity) before anything is written (criteria 1, 3, 4).
2. **The report**: `.abcd/work/reviews/<date>-consistency[-<itd-N>]/
   00-summary.md` with `review_of_commit` (itd-28's pin) and one row per
   finding (criterion 1).
3. **The captures**: one `capture` per finding (category inconsistency,
   source agent-finding, found-during naming the report), after a ledger
   search for an open record naming either end; a hit links the report to it
   instead (criterion 2).
4. **Read-only over the record**: no write outside the report and the ledger
   (criterion 4).

## Out of scope

- The shape role (itd-34's lint) and the overlap question (itd-42).
- A pre-commit hook or a scheduled run; it is on demand.

## Approach

`internal/core/intent/consistency.go` assembles and validates; the host pass
follows the intent-audit request/ingest shape (a request file, a validated
JSON return); the report writer reuses the reviews-charter template; the
captures go through `capture`'s core with the dedup search over open records.

## How the criteria are satisfied

| Criterion | Where |
| --- | --- |
| 1 dated report, both ends quoted | scope 1, 2 |
| 2 capture per finding, deduplicated | scope 3 |
| 3 scoped run | scope 1 |
| 4 read-only, host-judged | scope 1, 4 |
