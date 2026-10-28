---
schema_version: 1
id: "iss-2610262038278710"
slug: "the-glossary-defines-a-step-as-a"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 796 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/step.md:4"
remedy: "none (filed automatically)"
finding_ref: cfk-f86aca1aac0fc17e
---

The glossary defines a step as a landable piece of a spec; the land chapter uses "step" for its own gate, push and enqueue acts while also citing a spec's step.

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/step.md:4` — "One of the ordered, independently landable pieces a spec lists under its Steps section"
- End B: `.abcd/development/brief/04-surfaces/38-land.md:30` — "Every step is idempotent, so running the verb again resumes where it stopped."

step.md gives step one sense: an ordered, independently landable piece a spec lists. 38-land calls its gate, push and enqueue acts steps (and its JSON carries `steps`), yet the same page cites spc-2610150728396006 step 4 in the spec sense. That is two senses in one document, and the glossary has no row for the second.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 796.
