---
id: dcn-2611260453108828
slug: accept-two-narrowings-of-the-guard-s
date: 2026-11-26
---
2026-11-26 — Accept two narrowings of the guard's round-5 holds while the person is away: a line that defines a function keeps the program-lookup reading unless the function is named after the looked-up program or a lookup command, or the line has an alias, eval, source or builtin; and an IFS set as the prefix of read, mapfile or readarray, a blank-only IFS value, or a restore of a saved IFS, does not hold every unquoted value. Grounds: the literal holds stopped the repository's own git hooks and every plugin hook script, which define unrelated functions and read lines with IFS= read; each narrowed case changes no word the guard reads as a program. Reverse by removing the exceptions in internal/core/guard (fixedprogram.go, carried.go) and the hooks will need rewriting.
