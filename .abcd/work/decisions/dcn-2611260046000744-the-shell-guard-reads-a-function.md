---
id: dcn-2611260046000744
slug: the-shell-guard-reads-a-function
date: 2026-11-26
---
2026-11-26 — The shell guard reads a function definition only in its two ordinary forms (a name followed by () and a body, or the function keyword followed by one plain name and an unquoted brace), and holds every other form of the function keyword until it is rewritten in one of those, instead of modelling the shell's whole function grammar. Grounds: each fix that taught the guard more of the grammar opened another corner and, last, a cost that grew faster than the line; a hold costs an agent one rewrite, and agents write the two ordinary forms. Reverse: drop the hold and read the further forms, each with its own test and a cost bound.
