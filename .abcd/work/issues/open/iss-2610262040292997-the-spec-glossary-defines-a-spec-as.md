---
schema_version: 1
id: "iss-2610262040292997"
slug: "the-spec-glossary-defines-a-spec-as"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 934 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/spec.md:4"
remedy: "none (filed automatically)"
finding_ref: cfk-17100ab65b6f05f7
---

The spec glossary defines a spec as broken into tasks; the step glossary forbids 'task' for the unit below a spec.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/spec.md:4` — "broken into ordered tasks with acceptance criteria"
- End B: `.abcd/development/brief/glossary/core/step.md:6` — "forbidden_synonyms: ["task", "sub-task", "scope"]"

The spec entry's definition breaks a spec into ordered tasks, and its body says numbered tasks. The step entry makes the step the unit below a spec and lists 'task' and 'sub-task' as forbidden synonyms. So the glossary's own definition of spec uses a word the glossary forbids.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 934.
