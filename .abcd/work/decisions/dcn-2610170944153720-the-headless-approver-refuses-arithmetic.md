---
id: dcn-2610170944153720
slug: the-headless-approver-refuses-arithmetic
date: 2026-10-17
---
2026-10-17 — The headless approver refuses arithmetic on anything but written numbers. Bash evaluates a name it meets in an arithmetic context as arithmetic in turn, so a value the line does not fix can hold an array index that runs a command the line never names. The approver reads the whole command text, quotes and here-documents included, and refuses a name, a quote or an expansion in (( )), $(( )), $[ ] and a for (( )) header, the number tests and -v of [[ ]], let and integer or name-reference declarations, a non-numeric array index or substring offset, ${!name}, and a variable name not written in the line handed to read, unset, printf -v, wait -p, test -v or a declaration. [ and test compare numbers without evaluating them and stay approved. Reading the text whole over-refuses a script that only quotes such a construct, which the agent can write another way; a reader that tracked quoting could be misled by what a quote hides. This refines dcn-2610170937000859.
