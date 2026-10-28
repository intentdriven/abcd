---
schema_version: 1
id: "iss-2610262040551548"
slug: "the-glossary-keeps-step-in-the-build"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 965 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/step.md:18"
remedy: "none (filed automatically)"
related_intents: [itd-2609201916151817]
finding_ref: cfk-259609c8848ec806
---

The glossary keeps 'step' in the build loop for a spec's piece only; the build intent uses it for one invocation of the loop.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/step.md:18` — "In the build loop the word names the spec's piece and nothing else"
- End B: `.abcd/development/intents/planned/itd-2609201916151817-one-verb-takes-a-single-intent-from.md:21` — "every invocation reads the state, does one step, writes the state and exits"

step.md fixes that in the build loop the word names the spec's step and nothing else, and that one `implement step` call performs a stage, not a step. The build intent's press release says each invocation 'does one step', using the word for one loop advance (a stage), the sense the glossary rules out.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 965.
