---
schema_version: 1
id: "iss-2610262035274211"
slug: "the-reading-chapter-derives-a-position-s"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 573 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/23-reading.md:187"
remedy: "In 01-agents.md:317 say the locator derives the file from the position (agents/cold-reading-POSITION.md) and checks the frontmatter position and regime against it, as 23-reading.md:187 says."
finding_ref: cfk-ba15b3bc97f11d24
---

The reading chapter derives a position's definition file from its name instead of looking it up; the agent catalogue says the locator finds the definition by its `position` frontmatter field.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/23-reading.md:187` — "The file is derived from the position, `agents/cold-reading-<position>.md`,"
- End B: `.abcd/development/brief/05-internals/01-agents.md:317` — "the locator resolves a position to its definition by this field"

23-reading says the definition file is derived from the position as `agents/cold-reading-<position>.md`, 'rather than looked up', and that the frontmatter's `position` is only checked against it afterwards. 01-agents' frontmatter table says the locator resolves a position to its definition by the `position` field, which is a lookup over the prompts. A definition whose filename and frontmatter disagree would be found by one mechanism and missed by the other, so both accounts of resolution cannot be right.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 573.
