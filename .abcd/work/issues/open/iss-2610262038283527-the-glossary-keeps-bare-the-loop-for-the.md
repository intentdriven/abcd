---
schema_version: 1
id: "iss-2610262038283527"
slug: "the-glossary-keeps-bare-the-loop-for-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 797 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/loop.md:38"
remedy: "none (filed automatically)"
finding_ref: cfk-48fc2f98d14d93dc
---

The glossary keeps bare "the loop" for the record loop and says abcd never owns a run loop; the agent catalog uses bare "the loop" for abcd's own implement loop.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/loop.md:38` — "abcd never owns the run loop."
- End B: `.abcd/development/brief/05-internals/01-agents.md:40` — "the loop names each in turn and records the"

loop.md registers three senses, keeps bare "the loop" for the record loop, wants the others named in full, and says abcd never owns the run loop because an adapter supplies it. 01-agents uses bare "the loop" for the implement loop that `build` and `drain` run in abcd's own binary. That is a sense the glossary does not register and says abcd does not have.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 797.
