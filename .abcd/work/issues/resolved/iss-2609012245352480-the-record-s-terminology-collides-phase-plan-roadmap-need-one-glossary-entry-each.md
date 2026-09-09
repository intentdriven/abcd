---
schema_version: 1
id: "iss-2609012245352480"
slug: "the-record-s-terminology-collides-phase-plan-roadmap-need-one-glossary-entry-each"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "design-interview-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/roadmap/phases/README.md"
resolution: "Nine core glossary entries name every sense of phase, plan, roadmap, record, surface, ledger, construal, reading-position and loop, each with the one spelling for that sense and a repo-relative link to the chapter it lives in; phases/README.md and commands/intent.md each point at the entry for the word they use. Term-linking from the site export stays unbuilt: internal/core/site reads no glossary."
impact: additive
resolved_by:
  commit: "5ddd96c8f634e76d3731e24c7c4ac71f0f344903"
---

The record uses several of its own words in more than one sense, and a product thinker cannot tell them apart from the brief alone. 'Phase' names a roadmap phase (an ordered stretch of work ending in a milestone, phases/phase-N-slug.md) and also the brief's plumbing phases that a roadmap phase bundles (phases/README.md, 'which intents and which brief plumbing-phases this phase bundles'); tonight two different documents both claimed the number 7 and two more the number 8. 'Plan' names the intent verb (abcd intent plan, the maintainer's sign-off that mints a spec) and also the roadmap as a build plan, and a session's own plan. 'Roadmap' holds phases and rfcs but the dashboard reads the spec store. 'Record' means the whole development record and also one file in a family. 'Surface' means a verb's front door and also the brief's surface chapters. 'Closed loop', 'ledger', 'construal' and 'reading position' appear with no first-use definition outside the chapter that coins them. Ask: one glossary entry per term in the brief's glossary (or a glossary chapter if none exists), each with the senses it carries and the one spelling used for each sense, referenced from phases/README.md and the intent surface page; the site export links terms to it. Surfaced at the 2026-09-01 interview when the maintainer could not follow questions that used these words unexplained.

## Grounds

- pursued: we expect a product thinker to follow a question that uses these words once the glossary fixes one spelling per sense; it is wrong if a reader still has to ask which phase or which plan is meant, or if the record keeps minting new senses the entries do not catch.
