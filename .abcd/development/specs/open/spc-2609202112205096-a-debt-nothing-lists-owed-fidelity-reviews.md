---
id: spc-2609202112205096
slug: a-debt-nothing-lists-owed-fidelity-reviews
intent: itd-2609150819445595
origin: researcher-authored
production_mode: hand-written
---
# a-debt-nothing-lists-owed-fidelity-reviews

## Summary

The design record for itd-2609150819445595, from the four decisions on the
intent (2026-09-20).

## Scope

1. **One reader of the review marker across `shipped/`** in
   `internal/core/intent`: for every shipped intent, the first marker line
   (`existingMarker` is the authority) yields one of OWED, INGESTED,
   DEAD_LETTER, or none; the reader returns the set with each intent's
   receipt id where one exists and the dead-letter reason where one was
   recorded. It reads; it never rewrites a request (decision 2).
2. **The owed set** (decision 3): OWED plus none; DEAD_LETTER is its own
   heading with the reason and is not counted; INGESTED is absent. A
   no-marker entry says a receipt is minted on re-emit.
3. **Three surfaces** (decision 4): bare `abcd intent audit` renders the
   listing in text and `--json` (one entry per shipped intent: id, state,
   receipt, the re-emit command; no local-tier path); bare `abcd intent`
   adds the owed count beside its bucket counts from the same reader;
   the record dispatcher's next move for a shipped intent reads the marker
   and says "fidelity review owed, receipt rcp-…, re-emit with `abcd intent
   audit <itd-N>`" in place of the current "none".
4. **No gate** (decision 1): nothing in `lint`, `record-lint` or the release
   cut reads the owed set.

## Out of scope

Running the auditor (host-delegated); a legacy marker; any stamp on a
no-marker intent (re-emit mints the receipt as it does today).

## Approach

Test-first: fixtures with one shipped intent per state, plus a shipped
intent with a duplicated marker to pin that the first marker wins. One lane.
The reader is the one canonical primitive; the count on the lifecycle board
and the dispatcher's next move call it rather than re-scanning.

## How the criteria are satisfied

1 to 4 by pieces 1 and 2; 5 and 8 by piece 3's listing; 6 by piece 3's
board count; 7 by piece 3's dispatcher move.

