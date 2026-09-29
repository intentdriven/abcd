---
schema_version: 1
id: "iss-2609291313276243"
slug: "the-glossary-entry-for-step-abcd-development-brief-glossary"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/step.md"
resolution: "Resolved by the lane-stage rename ruled as BU1 (2026-09-29): spec pieces keep 'step', the implement loop's lane stages are 'stage' in the state file (schema version 4, older versions migrated on read), the step and receipt results (performed_stage, stage), every loop refusal (refusal.stage) and the board's lane; the glossary entry for step now says the word names the spec's piece only."
impact: breaking
resolved_by:
  spec: "spc-2609212138246060"
  commit: "3ebb7b0e36c74df65853fc39632079c4b08bc8b9"
---

The glossary entry for step (.abcd/development/brief/glossary/core/step.md) says the loop's step interface, abcd implement step, uses the same word for the same thing as a spec's steps: the next piece of the current spec. The loop does not work that way: implement step performs one stage of the current lane (worktree, brief, implement, validate, land), and a lane lands one spec step, so the state file, the JSON payloads (step, performed, refusal.step) and the command pages already use one word for two things and qualify the spec's as spec step. itd-2609212103565953 criterion 5 asks the command page to describe them as one word for one thing, which cannot be written truthfully until someone rules which of the two keeps the word (rename the lane's stages, or change what implement step means) or restates the criterion.

## Grounds

- pursued: with one word for one thing, a reader of the loop's payloads and pages never mistakes a lane stage for a spec step; shown wrong if a loop payload or command page still says step for a lane's stage
