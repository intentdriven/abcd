---
schema_version: 1
id: "iss-2610262035450179"
slug: "the-glossary-forbids-bare-the-loop-for"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 595 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/loop.md:57"
remedy: "none (filed automatically)"
finding_ref: cfk-b151ab1d96510144
---

The glossary forbids bare 'the loop' for the agent loop; the mode chapter's title uses bare 'the Loop' for exactly that loop

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/loop.md:57` — "Do not use it for the agent loop of the wider literature"
- End B: `.abcd/development/brief/04-surfaces/26-mode.md:1` — "Whose Answer the Loop Waits On, and the Away Mode"

loop.md reserves 'the loop' for the record loop and says not to use it for the agent loop of the wider literature, with the other loops always named in full. The mode chapter is titled 'Whose Answer the Loop Waits On' and its first line says the facts it holds are what 'the agent loop' is judged by, so its bare 'Loop' means the agent loop. That is the substitution the glossary rules out.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 595.
