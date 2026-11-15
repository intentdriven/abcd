---
id: dcn-2611151742181388
slug: the-masking-of-u-2028-line-separator-and
date: 2026-11-15
---
2026-11-15 — The masking of U+2028 LINE SEPARATOR and U+2029 PARAGRAPH SEPARATOR in spc-2610301950037469 step 3 moves out of termsafe.Sanitize into a new termsafe.SanitizeLine (Sanitize, then both runes masked), used by diagnosticLine, the guard's refused-limit deny line and the rules loader's refused-limit note. Grounds: the step 3 correctness delta review at 47088a0a7 found that widening Sanitize changes what read-side checks compare against, so a memory note or drain state an earlier abcd stored with either rune would stop reading; run state already guards against exactly that with a frozen rune set. The rule text itself needs neither mask since the refused-limit sentence is built from closed fields (dcn-2611151647301368). Reverse: fold the two runes into Sanitize and give memnote and drain the frozen read-side set run state uses.
