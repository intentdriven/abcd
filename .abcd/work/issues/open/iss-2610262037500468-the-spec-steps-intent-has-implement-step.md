---
schema_version: 1
id: "iss-2610262037500468"
slug: "the-spec-steps-intent-has-implement-step"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 752 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/34-build.md:388"
remedy: "none (filed automatically)"
related_intents: [itd-2609212103565953]
finding_ref: cfk-9a087f86d607ceb5
---

The spec-steps intent has `implement step` and the spec's steps share one word; the build chapter says steps and stages are two words for two things.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/34-build.md:388` — "A spec's steps and a lane's stages are two words for two things"
- End B: `.abcd/development/intents/shipped/itd-2609212103565953-a-spec-lists-its-steps-and-the-build.md:43` — "`implement step` (the loop's step interface) and this section share the word, and the command page says so"

itd-2609212103565953 puts 'one word' in scope: the loop's step interface, implement step, and the spec's ## Steps section share the word step, and the command page says so. 34-build.md records ruling BU1 the other way: a spec's steps and a lane's stages are two words for two things, and state version 4 renamed the lane's step to stage so that step names only the spec's steps. One document makes the shared word the rule; the other removes it as a confusion.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 752.
