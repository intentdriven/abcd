---
schema_version: 1
id: "iss-2610262041427822"
slug: "the-spec-glossary-entry-says-a-spec"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1019 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/spec.md:21"
remedy: "none (filed automatically)"
related_intents: [itd-2609211913453478]
finding_ref: cfk-2db147baa35b7f84
---

The spec glossary entry says a spec breaks down into numbered tasks; the vocabulary intent made the step the unit below a spec.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/spec.md:21` — "decomposed into numbered tasks"
- End B: `.abcd/development/intents/shipped/itd-2609211913453478-one-page-in-the-glossary-maps-abcd-s.md:57` — "The unit below a spec is the step, a section, not a record family"

itd-2609211913453478 rules that the unit below a spec is the step, which is a section and not a record family. The step entry also lists 'task' as a forbidden synonym. The spec entry still describes a spec as 'decomposed into numbered tasks', with ids such as spc-3.1, which is the word the ruling replaced. A reader cannot tell from the glossary whether a spec is broken into tasks or into steps.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1019.
