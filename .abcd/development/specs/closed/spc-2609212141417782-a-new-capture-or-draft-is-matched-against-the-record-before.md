---
id: spc-2609212141417782
slug: a-new-capture-or-draft-is-matched-against-the-record-before
intent: itd-2609212137116617
origin: researcher-authored
production_mode: hand-written
---
# a-new-capture-or-draft-is-matched-against-the-record-before

## Summary

The design record for itd-2609212137116617: lexical matching at filing, a typed link, never a refusal.

## Scope

1. **The matcher** (`internal/core/record/match.go`): tokenises the new text and each candidate's title, press release or body, scores overlap, returns candidates above and below the threshold (criteria 1, 2, 4).
2. **The writers**: `capture` and `intent` create call it before the write and add the link field; the print (criteria 1, 2, 3).
3. **Configuration**: `match.threshold`, `match.fields` through the layered resolver (criterion 4).
4. **The discipline record** itd-84 edited to mark the rung (criterion 5).

## Out of scope

- Refusal; cross-repository; semantic matching.

## Approach

The overlap function is the one the embark ranking uses, moved to the record package as the canonical primitive; both callers run it under their existing locks.

## Footprint

- packages: internal/core/record, internal/core/capture, internal/core/intent
- tests: the scorer on fixtures; both writers with a planted double; the below-threshold json; the discipline edit

## How the criteria are satisfied

| Criterion | Where |
| --- | --- |
| 1 capture links | scope 1, 2 |
| 2 intent links | scope 1, 2 |
| 3 never refused | scope 2 |
| 4 threshold and json | scope 1, 3 |
| 5 itd-84 rung | scope 4 |
