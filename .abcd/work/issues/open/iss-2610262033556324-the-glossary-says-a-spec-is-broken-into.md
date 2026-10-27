---
schema_version: 1
id: "iss-2610262033556324"
slug: "the-glossary-says-a-spec-is-broken-into"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 451 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/spec.md:4"
remedy: "none (filed automatically)"
finding_ref: cfk-c0b08563c9500da2
---

The glossary says a spec is broken into ordered tasks with acceptance criteria; the intent chapter's spec stub carries Summary, Footprint and Steps sections and no criteria.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/spec.md:4` — "broken into ordered tasks with acceptance criteria"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:203` — "an empty `## Steps` section (a spec listing no step is built as one step)"

glossary/core/spec.md defines a spec as 'broken into ordered tasks with acceptance criteria', decomposed into numbered tasks. 05-intent says planning mints a stub with a Summary placeholder, a Footprint section and 'an empty ## Steps section (a spec listing no step is built as one step)'. One page calls the unit below a spec a task and the other a step, which the glossary's own step entry forbids calling a task, and the stub carries no acceptance criteria.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 451.
