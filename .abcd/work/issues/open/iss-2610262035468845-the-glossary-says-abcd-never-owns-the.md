---
schema_version: 1
id: "iss-2610262035468845"
slug: "the-glossary-says-abcd-never-owns-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 596 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/loop.md:38"
remedy: "none (filed automatically)"
finding_ref: cfk-56c4c3786bd91129
---

The glossary says abcd never owns the run loop; the site chapter reads the state file of abcd's own implement loop

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/loop.md:38` — "abcd never owns the run loop."
- End B: `.abcd/development/brief/04-surfaces/22-site.md:118` — "implement loop's state file for Now's lane rows through the reader its front door hands it, the loop's own"

loop.md states that abcd never owns the run loop: it defines the contract and a pluggable adapter supplies the loop. The site chapter says the site build reads the implement loop's state file through the loop's own reader to render Now's lane rows. That is abcd's own loop, with its own state file and reader, iterating over ready work. The two statements cannot both hold, and 'the implement loop' is also a fourth loop sense the glossary's senses table does not carry.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 596.
