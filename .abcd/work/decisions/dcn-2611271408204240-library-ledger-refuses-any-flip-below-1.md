---
id: dcn-2611271408204240
slug: library-ledger-refuses-any-flip-below-1
date: 2026-11-27
---
2026-11-27 — library ledger refuses any --flip below 1, a negative included, at the front door before the corpus is read, with one message that names --flip and 'a ledger line number, 1 or more' and never the value typed. Grounds: a ledger line is 1-based (source.Flip refuses line < 1), the core's refusal for a negative echoed the number, and a usage error does not depend on whether a corpus exists, so --flip 0 with no corpus is exit 2 rather than the no-corpus exit 3. Reverse by moving the check below sourceCorpusDir in newSourceLedgerCommand, or narrowing it to flip == 0 so negatives reach source.Flip again (iss-2611271406524430).
