---
id: dcn-2611131259525105
slug: the-spot-check-store-s-cleaning-rule
date: 2026-11-13
---
2026-11-13 — The spot-check store's cleaning rule (spc-2610301925429250 step 1) keeps a word that holds a slash only when three things hold: it begins with an ASCII letter or digit, every slash has an ASCII letter or digit right before it, and every character of it is an ASCII letter, a digit, a dot, an underscore, a dash or a slash. Any other word that holds a slash is replaced whole, and the text still holds no backslash and no tilde. The third clause supersedes the two-clause statement of the rule in the drain pages and the store's package comment, and it widens what dcn-2611131203280652 says a stored reason loses: besides a path set in brackets, quotes or backticks and a path that begins with a dot, a stored reason now loses an option or a name written straight against a path and a path with a line number or punctuation written against it. A kept word is a bare repository-relative path and nothing else; a path with a line number is replaced, and the card takes file names from the issue record, as that decision already says.
