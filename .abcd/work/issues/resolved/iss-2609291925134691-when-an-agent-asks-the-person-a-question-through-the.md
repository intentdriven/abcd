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
resolution: "GRILL gains the rule: the example that makes a question answerable goes in the question text and in each option's preview, because prose before the question tool call is invisible while the question shows; the rule is pinned by TestGrillDomainCarriesQuestionVisibilityAndAddresseeRules. The plugin interview pages (commands/intent.md) take the matching line in the lane that holds them; the eval or lint the remedy floats for an option with no example is not built here."
impact: internal
resolved_by:
  commit: "5357c4c68128e1d656e0cb105391ea3893bcfb8a"
---

When an agent asks the person a question through the interactive question tool, the example that makes the question answerable must be visible IN the question: in the question text and in each option's preview. Prose written before the tool call is not visible while the question shows. On 2026-09-29 the person twice answered 'explain in laypersons terms' / 'show me the example, I can't see it' because the example sat in prose above the question.

## Grounds

- pursued: guidance a person needs in order to answer must stay visible while they answer, not sit where the answering surface hides it. Primary sources: Nielsen Norman Group, K. Sherwin, 'Placeholders in Form Fields Are Harmful' (2014, https://www.nngroup.com/articles/form-design-placeholders/): disappearing guidance strains short-term memory and essential information belongs where it is visible at all times; GOV.UK Design System, Radios (https://design-system.service.gov.uk/components/radios/): the question sits in the legend and each option carries its own short hint. Shown wrong if a product thinker answering a question built to this rule still asks to see the example.
