---
schema_version: 1
id: "iss-2610262035432356"
slug: "the-glossary-says-the-reading-chapter"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 593 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/reading-position.md:34"
remedy: "none (filed automatically)"
finding_ref: cfk-09e1ccaa9f3198ea
---

The glossary says the reading chapter writes 'reading position' in full on first use; the chapter opens with bare 'position' and never writes the full phrase

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/reading-position.md:34` — "The record writes the full phrase on first use and drops to bare "position" inside the reading chapter"
- End B: `.abcd/development/brief/04-surfaces/23-reading.md:40` — "Assembly takes a position and a target state, and nothing else."

The reading-position entry describes how the record spells the term: the full phrase on first use, then bare 'position' inside the reading chapter. The reading chapter's first use is bare ('Assembly takes a position and a target state') and the phrase 'reading position' does not appear in it, so the spelling convention the glossary records for this chapter is not the one the chapter follows.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 593.
