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
---

The glossary entry for step (.abcd/development/brief/glossary/core/step.md) says the loop's step interface, abcd implement step, uses the same word for the same thing as a spec's steps: the next piece of the current spec. The loop does not work that way: implement step performs one stage of the current lane (worktree, brief, implement, validate, land), and a lane lands one spec step, so the state file, the JSON payloads (step, performed, refusal.step) and the command pages already use one word for two things and qualify the spec's as spec step. itd-2609212103565953 criterion 5 asks the command page to describe them as one word for one thing, which cannot be written truthfully until someone rules which of the two keeps the word (rename the lane's stages, or change what implement step means) or restates the criterion.
