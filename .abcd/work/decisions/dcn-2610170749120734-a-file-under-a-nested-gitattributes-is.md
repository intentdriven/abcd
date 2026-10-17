---
id: dcn-2610170749120734
slug: a-file-under-a-nested-gitattributes-is
date: 2026-10-17
---
2026-10-17 — A file under a nested .gitattributes is never generated (facilitator-on-delegation, drain-many-accounts step 11, refines dcn-2610170652464473): generated files are declared in the root .gitattributes of origin's default branch alone, and since a nested .gitattributes could unset the attribute below it, every file at or below a directory holding one is left out of the generated set, found from the same file listing with no extra git read (iss-2610152125101468).
