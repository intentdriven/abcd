---
id: dcn-2608300000000018
slug: itd-179-round-5-correction-to-the
date: 2026-08-30
---
2026-08-30 — itd-179 round-5, correction to the round-4 letter-floor entry above: `unicode.Other_Default_Ignorable_Code_Point` does NOT "name exactly those four letters and nothing else", and the remainder is not "format characters and variation selectors". Measured over the whole code space: the table holds 3776 code points, of which exactly seven are assigned — the four Hangul filler letters (Lo) and three non-spacing marks (Mn) — and the other 3769 are unassigned; it carries zero format characters and zero variation selectors, which reach `Default_Ignorable_Code_Point` through the other contributors to that derived property. The conclusion the round-4 entry drew is unchanged and was verified exhaustively — the only LETTERS the table can exclude are those four — but the description of the rest of it was wrong, so it is corrected here rather than left to be re-derived (iss-2608301657350399).
