---
schema_version: 1
id: "iss-2610262041419578"
slug: "the-step-intent-says-implement-step-and"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1018 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/step.md:18"
remedy: "none (filed automatically)"
related_intents: [itd-2609212103565953]
finding_ref: cfk-31c2799c90050366
---

The step intent says `implement step` and the spec's Steps section share one word; the step glossary entry says `implement step` performs a stage, not a step.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/step.md:18` — "`abcd implement step` performs one stage, not one step"
- End B: `.abcd/development/intents/shipped/itd-2609212103565953-a-spec-lists-its-steps-and-the-build.md:43` — "`implement step` (the loop's step interface) and this section share the word, and the command page says so."

The shipped intent rules that the loop's `implement step` interface and the spec's `## Steps` section share the word 'step', and that the command page says so. The glossary entry rules the opposite. In the build loop the word names only the spec's piece, and `abcd implement step` performs one stage of a lane. The two documents give the word one shared sense and two separate senses, so a reader of the intent will take `implement step` to land a spec step.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1018.
