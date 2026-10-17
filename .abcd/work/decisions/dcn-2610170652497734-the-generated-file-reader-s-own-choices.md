---
id: dcn-2610170652497734
slug: the-generated-file-reader-s-own-choices
date: 2026-10-17
---
2026-10-17 — The generated-file reader's own choices (facilitator-on-delegation, drain-many-accounts step 11, builder): a file counts as generated only when origin's default branch holds it as a blob, which is how a directory is never generated; .gitattributes is located with ls-tree -l, so absence, its type and its size are read before its blob; matching is case-sensitive whatever core.ignorecase says; a line git ignores (a negated pattern, an invalid or reserved attribute name) is ignored, and a line git reads but abcd does not understand voids every line before it when it could unset the attribute; a macro naming linguist-generated decides not generated; the reader is exported as claim.GeneratedFiles and Generated.Is for the drain's conflict check (grounds: git 2.52 check-attr agrees on every supported row of the differential test; iss-2610152125101468).
