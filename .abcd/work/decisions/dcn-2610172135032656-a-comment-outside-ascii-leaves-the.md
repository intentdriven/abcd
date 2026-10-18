---
id: dcn-2610172135032656
slug: a-comment-outside-ascii-leaves-the
date: 2026-10-17
---
2026-10-17 — A comment outside ASCII leaves the declaration plain (facilitator-on-delegation, drain-many-accounts step 11 follow-up, refines dcn-2610170900575176): the ASCII rule is judged only on the lines of the root .gitattributes that git reads, so a blank line or one whose first non-blank byte is '#' decides nothing, as git's attribute parser skips both whole; a byte-order mark, which git skips, still voids the declaration, since leaning to not generated only shares less (grounds: git 2.52 check-attr reads an indented comment naming -linguist-generated as no line at all; abcd's own .gitattributes has em dashes in its comments and so shared nothing; iss-2610172133042562)
