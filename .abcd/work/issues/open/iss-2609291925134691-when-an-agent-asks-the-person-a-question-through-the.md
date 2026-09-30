---
schema_version: 1
id: "iss-2609291925134691"
slug: "when-an-agent-asks-the-person-a-question-through-the"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "rulings interview with the product thinker, 2026-09-29 (abcd-23 [b1e81b])"
origin: researcher-authored
production_mode: hand-written
remedy: "Put the context sentence and a concrete example inside every question, and show each option's consequence in its preview; add this to the GRILL rules and to the grill/intent interview command pages, with an eval or lint that flags a question whose options carry no example."
---

When an agent asks the person a question through the interactive question tool, the example that makes the question answerable must be visible IN the question: in the question text and in each option's preview. Prose written before the tool call is not visible while the question shows. On 2026-09-29 the person twice answered 'explain in laypersons terms' / 'show me the example, I can't see it' because the example sat in prose above the question.
