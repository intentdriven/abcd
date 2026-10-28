---
schema_version: 1
id: "iss-2610170657019661"
slug: "a-message-addressed-to-the-product"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "autonomous drain build, 2026-10-17"
origin: researcher-authored
production_mode: hand-written
found_at: "agents/question-drafter.md"
remedy: "Make every message addressed to the product thinker state the decision by its outcomes in plain words with the mechanism cut out: add this as a rule in the GRILL asking rules and in the question-drafter agent's checks, so a draft naming files, flags or internals in a product question is redrafted before it is put"
refines: [itd-201]
---

A message addressed to the product thinker, a question above all, still carries mechanism details (file names, flags, internal terms, how the code does it) where the product thinker needs only the choice and what each answer leads to, in plain words. Questions put to the product thinker must always be simplified: cut the mechanism and state the decision by its outcomes.

## Checked at the default branch 2026-10-28

Partly met since: abcd's own interview questions now address the product thinker in outcomes, with no record ids, code or internals (internal/core/question/asking.go, agents/question-drafter.md), and the question check enforces part of it. Still open: messages to the product thinker that are not questions, and file and flag names in them.
