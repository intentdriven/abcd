---
id: dcn-2611130858310110
slug: the-docs-lint-s-crosswalk-rule-keeps
date: 2026-11-13
---
2026-11-13 — The docs lint's crosswalk rule keeps pairing each table row's code spans on the row alone, while the link checks, the footnote and URL rules and the forbidden-synonym check read whole the lines below a backtick left open, to the next blank line (iss-2611130850006749). Grounds: the crosswalk rule reports a row with no footnote, so reading a span as prose is the side that lets an uncited row through, not the safe one; and the page parses each table row on its own, so the line-alone pairing is the page's own for a row. The footnote rule's unreferenced-definition direction is left reading whole with the rest, and a code-quoted marker there can quiet that finding. Reversal: a page reader that parses tables, or a ruling that the crosswalk rule should read rows as the other checks do.
